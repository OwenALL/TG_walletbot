package task

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
)

// FinanceProfitTask 余额宝每日派息定时任务
// 定期检查并发放投资收益到用户 USDT 钱包
type FinanceProfitTask struct {
	financeSvc *service.FinanceService
	rdb        *redis.Client
	logger     *zap.Logger
	interval   time.Duration
}

// NewFinanceProfitTask 创建余额宝派息任务
// interval: 检查间隔 (建议 1 小时)
func NewFinanceProfitTask(
	financeSvc *service.FinanceService,
	rdb *redis.Client,
	logger *zap.Logger,
	interval time.Duration,
) *FinanceProfitTask {
	return &FinanceProfitTask{
		financeSvc: financeSvc,
		rdb:        rdb,
		logger:     logger,
		interval:   interval,
	}
}

// Start 启动定时派息任务 (阻塞运行，通过 ctx 取消退出)
func (t *FinanceProfitTask) Start(ctx context.Context) {
	t.logger.Info("余额宝派息任务启动",
		zap.Duration("interval", t.interval),
	)

	// 首次启动延迟 30 秒，等待其他服务就绪
	select {
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
		t.logger.Info("余额宝派息任务收到退出信号 (启动阶段)")
		return
	}

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	// 启动后立即执行一次
	t.execute(ctx)

	for {
		select {
		case <-ticker.C:
			t.execute(ctx)
		case <-ctx.Done():
			t.logger.Info("余额宝派息任务已停止")
			return
		}
	}
}

// execute 执行一次派息
func (t *FinanceProfitTask) execute(ctx context.Context) {
	// 使用 Redis 分布式锁防止重复派息 (同一天只派一次)
	today := time.Now().Format("2006-01-02")
	lockKey := fmt.Sprintf("finance:profit:lock:%s", today)

	// 尝试获取锁 (24 小时过期)
	acquired, err := t.rdb.SetNX(ctx, lockKey, "1", 24*time.Hour).Result()
	if err != nil {
		t.logger.Error("获取派息锁失败", zap.Error(err))
		return
	}
	if !acquired {
		// 今天已经派过息了
		t.logger.Debug("今日已完成派息，跳过", zap.String("date", today))
		return
	}

	t.logger.Info("开始执行每日派息", zap.String("date", today))

	count, err := t.financeSvc.DistributeDailyProfits(ctx)
	if err != nil {
		t.logger.Error("每日派息执行失败",
			zap.String("date", today),
			zap.Error(err),
		)
		// 派息失败，删除锁，允许下次重试
		t.rdb.Del(ctx, lockKey)
		return
	}

	t.logger.Info("每日派息完成",
		zap.String("date", today),
		zap.Int("processed_count", count),
	)
}
