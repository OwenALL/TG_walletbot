package exchange

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
)

// ratePair 定义一组汇率对的计算规则
type ratePair struct {
	From string
	To   string
	// calcFn 根据 PriceResult 计算汇率
	calcFn func(p *PriceResult) float64
}

// allRatePairs 所有需要自动更新的汇率对
var allRatePairs = []ratePair{
	{
		From: "USDT", To: "CNY",
		calcFn: func(p *PriceResult) float64 {
			// USDT->CNY = Tether.CNY / Tether.USD
			return p.Tether["cny"] / p.Tether["usd"]
		},
	},
	{
		From: "CNY", To: "USDT",
		calcFn: func(p *PriceResult) float64 {
			// CNY->USDT = Tether.USD / Tether.CNY
			return p.Tether["usd"] / p.Tether["cny"]
		},
	},
	{
		From: "USDT", To: "TRX",
		calcFn: func(p *PriceResult) float64 {
			// USDT->TRX = Tether.USD / TRON.USD
			return p.Tether["usd"] / p.TRON["usd"]
		},
	},
	{
		From: "TRX", To: "USDT",
		calcFn: func(p *PriceResult) float64 {
			// TRX->USDT = TRON.USD / Tether.USD
			return p.TRON["usd"] / p.Tether["usd"]
		},
	},
	{
		From: "TRX", To: "CNY",
		calcFn: func(p *PriceResult) float64 {
			// TRX->CNY = TRON.CNY / TRON.USD
			return p.TRON["cny"] / p.TRON["usd"]
		},
	},
	{
		From: "CNY", To: "TRX",
		calcFn: func(p *PriceResult) float64 {
			// CNY->TRX = TRON.USD / TRON.CNY
			return p.TRON["usd"] / p.TRON["cny"]
		},
	},
}

// 需要清除的缓存键列表
var rateCacheKeys = []string{
	"rate:USDT:CNY",
	"rate:CNY:USDT",
	"rate:USDT:TRX",
	"rate:TRX:USDT",
	"rate:TRX:CNY",
	"rate:CNY:TRX",
	"exchange:all_rates_display",
}

// ExchangeUpdater 汇率自动更新器
type ExchangeUpdater struct {
	client           *CoinGeckoClient
	exchangeRateRepo port.ExchangeRateRepository
	cacheRepo        port.CacheRepository
	logger           *zap.Logger
	interval         time.Duration
}

// NewExchangeUpdater 创建汇率更新器
func NewExchangeUpdater(
	exchangeRateRepo port.ExchangeRateRepository,
	cacheRepo port.CacheRepository,
	logger *zap.Logger,
	interval time.Duration,
) *ExchangeUpdater {
	return &ExchangeUpdater{
		client:           NewCoinGeckoClient(logger),
		exchangeRateRepo: exchangeRateRepo,
		cacheRepo:        cacheRepo,
		logger:           logger,
		interval:         interval,
	}
}

// Start 启动后台定时更新（阻塞，通过 ctx 取消）
func (u *ExchangeUpdater) Start(ctx context.Context) {
	u.logger.Info("汇率自动更新器已启动",
		zap.Duration("interval", u.interval),
	)

	// 启动后立即执行一次更新
	u.doUpdate(ctx)

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("汇率自动更新器已停止")
			return
		case <-ticker.C:
			u.doUpdate(ctx)
		}
	}
}

// doUpdate 执行一次汇率更新
func (u *ExchangeUpdater) doUpdate(ctx context.Context) {
	u.logger.Debug("开始获取最新汇率...")

	prices, err := u.client.GetPrices(ctx)
	if err != nil {
		u.logger.Error("获取 CoinGecko 汇率失败", zap.Error(err))
		return
	}

	u.logger.Info("CoinGecko 价格获取成功",
		zap.Float64("tron_usd", prices.TRON["usd"]),
		zap.Float64("tron_cny", prices.TRON["cny"]),
		zap.Float64("tether_usd", prices.Tether["usd"]),
		zap.Float64("tether_cny", prices.Tether["cny"]),
	)

	updatedCount := 0
	for _, pair := range allRatePairs {
		newRate := pair.calcFn(prices)
		if err := u.updateRate(ctx, pair.From, pair.To, newRate); err != nil {
			u.logger.Error("更新汇率失败",
				zap.String("pair", fmt.Sprintf("%s->%s", pair.From, pair.To)),
				zap.Error(err),
			)
			continue
		}
		updatedCount++
		u.logger.Info("汇率已更新",
			zap.String("pair", fmt.Sprintf("%s->%s", pair.From, pair.To)),
			zap.Float64("rate", newRate),
		)
	}

	// 清除缓存
	u.clearCache(ctx)

	u.logger.Info("汇率更新完成",
		zap.Int("updated", updatedCount),
		zap.Int("total", len(allRatePairs)),
	)
}

// updateRate 更新单个汇率对
// 如果记录已存在，仅更新 rate 字段（保留 spread/fee/min_amount/max_amount 不变）
// 如果记录不存在，创建新记录
func (u *ExchangeUpdater) updateRate(ctx context.Context, from, to string, newRate float64) error {
	rateDecimal := decimal.NewFromFloat(newRate)

	// 查找现有记录
	existing, err := u.exchangeRateRepo.FindByPair(ctx, from, to)
	if err != nil {
		return fmt.Errorf("查询汇率记录失败 (%s->%s): %w", from, to, err)
	}

	if existing != nil {
		// 记录存在，仅更新 rate 字段，保留其他字段不变
		existing.Rate = rateDecimal
		return u.exchangeRateRepo.Upsert(ctx, existing)
	}

	// 记录不存在，创建新记录（使用默认值）
	newRecord := &entity.ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rateDecimal,
		Spread:       decimal.NewFromFloat(0.02), // 默认 0.02% 加点
		MinAmount:    decimal.NewFromFloat(10),
		MaxAmount:    decimal.NewFromFloat(500000),
		Enabled:      true,
	}
	return u.exchangeRateRepo.Upsert(ctx, newRecord)
}

// clearCache 清除所有汇率相关的 Redis 缓存
func (u *ExchangeUpdater) clearCache(ctx context.Context) {
	for _, key := range rateCacheKeys {
		if err := u.cacheRepo.Delete(ctx, key); err != nil {
			u.logger.Warn("清除缓存失败",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}
	u.logger.Debug("汇率缓存已清除", zap.Int("keys", len(rateCacheKeys)))
}
