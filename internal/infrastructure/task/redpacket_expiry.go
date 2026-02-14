package task

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
)

// RedPacketExpiryTask 红包过期退款定时任务
// 定期检查已过期的红包，自动退回未领取的金额到发送者钱包
type RedPacketExpiryTask struct {
	redPacketSvc *service.RedPacketService
	logger       *zap.Logger
	interval     time.Duration
}

// NewRedPacketExpiryTask 创建红包过期退款任务
// interval: 检查间隔 (建议 1 分钟)
func NewRedPacketExpiryTask(
	redPacketSvc *service.RedPacketService,
	logger *zap.Logger,
	interval time.Duration,
) *RedPacketExpiryTask {
	return &RedPacketExpiryTask{
		redPacketSvc: redPacketSvc,
		logger:       logger,
		interval:     interval,
	}
}

// Start 启动定时任务 (阻塞运行，通过 ctx 取消退出)
func (t *RedPacketExpiryTask) Start(ctx context.Context) {
	t.logger.Info("红包过期退款任务启动",
		zap.Duration("interval", t.interval),
	)

	// 首次启动延迟 10 秒，等待其他服务就绪
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		t.logger.Info("红包过期退款任务收到退出信号 (启动阶段)")
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
			t.logger.Info("红包过期退款任务已停止")
			return
		}
	}
}

// execute 执行一次过期检查和退款
func (t *RedPacketExpiryTask) execute(ctx context.Context) {
	if err := t.redPacketSvc.ExpireRedPackets(ctx); err != nil {
		t.logger.Error("红包过期退款执行失败", zap.Error(err))
	}
}
