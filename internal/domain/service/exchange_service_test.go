package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// 注意: ExchangeService.ExecuteExchange 使用了 gorm.DB 事务，无法直接通过 mock port 接口测试。
// 这里测试非事务方法: GetAllRatesDisplay, GetExchangePair, CalculateExchange

// newTestExchangeServiceDeps 创建 ExchangeService 非事务方法测试所需的 mock
func newTestExchangeServiceDeps() (*ExchangeService, *mocks.MockExchangeRateRepository, *mocks.MockExchangeRepository, *mocks.MockWalletRepository, *mocks.MockTransactionRepository, *mocks.MockCacheRepository) {
	exchangeRateRepo := new(mocks.MockExchangeRateRepository)
	exchangeRepo := new(mocks.MockExchangeRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	cacheRepo := new(mocks.MockCacheRepository)

	svc := &ExchangeService{
		db:               nil, // 非事务方法不需要 db
		exchangeRateRepo: exchangeRateRepo,
		exchangeRepo:     exchangeRepo,
		walletRepo:       walletRepo,
		transactionRepo:  txRepo,
		cacheRepo:        cacheRepo,
		logger:           nil, // 测试中日志不关键
	}
	return svc, exchangeRateRepo, exchangeRepo, walletRepo, txRepo, cacheRepo
}

// ==================== GetExchangePair 测试 ====================

func TestExchangeService_GetExchangePair_缓存命中(t *testing.T) {
	svc, _, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	pair := &ExchangePair{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
		MinAmount:    decimal.NewFromInt(10),
		MaxAmount:    decimal.NewFromInt(50000),
	}
	data, _ := json.Marshal(pair)
	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return(string(data), nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "TRX")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT", result.FromCurrency)
	assert.Equal(t, "TRX", result.ToCurrency)
	assert.True(t, decimal.NewFromFloat(6.5).Equal(result.Rate))
}

func TestExchangeService_GetExchangePair_缓存未命中_从DB读取(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	// 缓存未命中
	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))

	// 从 DB 读取
	rate := &entity.ExchangeRate{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
		Spread:       decimal.Zero,
		MinAmount:    decimal.NewFromInt(10),
		MaxAmount:    decimal.NewFromInt(50000),
		Enabled:      true,
	}
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(rate, nil)
	// 写入缓存
	cacheRepo.On("Set", ctx, "rate:USDT:TRX", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "TRX")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT", result.FromCurrency)
	assert.True(t, decimal.NewFromFloat(6.5).Equal(result.Rate))
	cacheRepo.AssertCalled(t, "Set", ctx, "rate:USDT:TRX", mock.AnythingOfType("string"), 5*time.Minute)
}

func TestExchangeService_GetExchangePair_汇率含加点(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:CNY").Return("", errors.New("cache miss"))

	rate := &entity.ExchangeRate{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         decimal.NewFromFloat(7.2),
		Spread:       decimal.NewFromFloat(2), // 2% 加点
		MinAmount:    decimal.NewFromInt(10),
		MaxAmount:    decimal.NewFromInt(50000),
		Enabled:      true,
	}
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "CNY").Return(rate, nil)
	cacheRepo.On("Set", ctx, "rate:USDT:CNY", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "CNY")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 有效汇率 = 7.2 * (1 - 2/100) = 7.2 * 0.98 = 7.056
	expectedRate := decimal.NewFromFloat(7.2).Mul(decimal.NewFromFloat(0.98))
	assert.True(t, expectedRate.Equal(result.Rate))
}

func TestExchangeService_GetExchangePair_汇率不存在(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:BTC").Return("", errors.New("cache miss"))
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "BTC").Return(nil, nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "BTC")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "暂不可用")
}

func TestExchangeService_GetExchangePair_汇率已禁用(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))
	rate := &entity.ExchangeRate{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
		Enabled:      false, // 已禁用
	}
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(rate, nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "TRX")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "暂不可用")
}

func TestExchangeService_GetExchangePair_DB查询失败(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(nil, errors.New("db error"))

	result, err := svc.GetExchangePair(ctx, "USDT", "TRX")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询汇率失败")
}

func TestExchangeService_GetExchangePair_缓存JSON损坏_回退DB(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	// 缓存返回无效 JSON
	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("invalid json{", nil)

	rate := &entity.ExchangeRate{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
		Spread:       decimal.Zero,
		MinAmount:    decimal.NewFromInt(10),
		MaxAmount:    decimal.NewFromInt(50000),
		Enabled:      true,
	}
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(rate, nil)
	cacheRepo.On("Set", ctx, "rate:USDT:TRX", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetExchangePair(ctx, "USDT", "TRX")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(6.5).Equal(result.Rate))
}

// ==================== CalculateExchange 测试 ====================

func TestExchangeService_CalculateExchange(t *testing.T) {
	tests := []struct {
		name           string
		pair           *ExchangePair
		fromAmount     decimal.Decimal
		wantToAmount   decimal.Decimal
		wantFee        decimal.Decimal
	}{
		{
			name: "USDT兑TRX",
			pair: &ExchangePair{
				FromCurrency: "USDT",
				ToCurrency:   "TRX",
				Rate:         decimal.NewFromFloat(6.5),
			},
			fromAmount:   decimal.NewFromInt(100),
			wantToAmount: decimal.NewFromFloat(650),
			wantFee:      decimal.Zero,
		},
		{
			name: "USDT兑CNY",
			pair: &ExchangePair{
				FromCurrency: "USDT",
				ToCurrency:   "CNY",
				Rate:         decimal.NewFromFloat(7.06),
			},
			fromAmount:   decimal.NewFromInt(50),
			wantToAmount: decimal.NewFromFloat(353),
			wantFee:      decimal.Zero,
		},
		{
			name: "小额兑换",
			pair: &ExchangePair{
				FromCurrency: "USDT",
				ToCurrency:   "TRX",
				Rate:         decimal.NewFromFloat(6.5),
			},
			fromAmount:   decimal.NewFromFloat(0.1),
			wantToAmount: decimal.NewFromFloat(0.65),
			wantFee:      decimal.Zero,
		},
		{
			name: "零金额",
			pair: &ExchangePair{
				FromCurrency: "USDT",
				ToCurrency:   "TRX",
				Rate:         decimal.NewFromFloat(6.5),
			},
			fromAmount:   decimal.Zero,
			wantToAmount: decimal.Zero,
			wantFee:      decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, _, _ := newTestExchangeServiceDeps()

			toAmount, fee := svc.CalculateExchange(tt.pair, tt.fromAmount)

			assert.True(t, tt.wantToAmount.Equal(toAmount), "期望 toAmount=%s, 实际=%s", tt.wantToAmount, toAmount)
			assert.True(t, tt.wantFee.Equal(fee), "期望 fee=%s, 实际=%s", tt.wantFee, fee)
		})
	}
}

// ==================== GetAllRatesDisplay 测试 ====================

func TestExchangeService_GetAllRatesDisplay_缓存命中(t *testing.T) {
	svc, _, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	display := &AllRatesDisplay{
		USDTtoCNY: decimal.NewFromFloat(706),
		USDTtoTRX: decimal.NewFromFloat(650),
		USDTtoTON: decimal.NewFromFloat(300),
		TRXtoUSDT: decimal.NewFromFloat(150),
	}
	data, _ := json.Marshal(display)
	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return(string(data), nil)

	result, err := svc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(706).Equal(result.USDTtoCNY))
	assert.True(t, decimal.NewFromFloat(650).Equal(result.USDTtoTRX))
}

func TestExchangeService_GetAllRatesDisplay_缓存未命中_从DB读取(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	// 缓存未命中
	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return("", errors.New("cache miss"))

	// USDT => CNY
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyCNY).Return(&entity.ExchangeRate{
		Rate: decimal.NewFromFloat(7.2), Spread: decimal.Zero, Enabled: true,
	}, nil)
	// USDT => TRX
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyTRX).Return(&entity.ExchangeRate{
		Rate: decimal.NewFromFloat(6.5), Spread: decimal.Zero, Enabled: true,
	}, nil)
	// USDT => TON
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, "TON").Return(nil, nil)
	// TRX => USDT
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyTRX, entity.CurrencyUSDT).Return(&entity.ExchangeRate{
		Rate: decimal.NewFromFloat(0.15), Spread: decimal.Zero, Enabled: true,
	}, nil)

	// 写入缓存
	cacheRepo.On("Set", ctx, "exchange:all_rates_display", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 100 USDT = 720 CNY
	assert.True(t, decimal.NewFromFloat(720).Equal(result.USDTtoCNY))
	// 100 USDT = 650 TRX
	assert.True(t, decimal.NewFromFloat(650).Equal(result.USDTtoTRX))
	// TON 不存在，应为 0
	assert.True(t, decimal.Zero.Equal(result.USDTtoTON))
	// 1000 TRX = 150 USDT
	assert.True(t, decimal.NewFromFloat(150).Equal(result.TRXtoUSDT))
}

func TestExchangeService_GetAllRatesDisplay_含加点汇率(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return("", errors.New("cache miss"))

	// USDT => CNY，加点 2%
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyCNY).Return(&entity.ExchangeRate{
		Rate: decimal.NewFromFloat(7.2), Spread: decimal.NewFromFloat(2), Enabled: true,
	}, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyTRX).Return(nil, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, "TON").Return(nil, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyTRX, entity.CurrencyUSDT).Return(nil, nil)
	cacheRepo.On("Set", ctx, "exchange:all_rates_display", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// 有效汇率 = 7.2 * (1 - 2/100) = 7.056
	// 100 USDT = 705.6 CNY
	expected := decimal.NewFromFloat(7.2).Mul(decimal.NewFromFloat(0.98)).Mul(decimal.NewFromInt(100))
	assert.True(t, expected.Equal(result.USDTtoCNY), "期望 %s, 实际 %s", expected, result.USDTtoCNY)
}

func TestExchangeService_GetAllRatesDisplay_缓存JSON损坏(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	// 缓存返回无效 JSON
	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return("corrupted{json", nil)

	// 回退到 DB 查询
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyCNY).Return(&entity.ExchangeRate{
		Rate: decimal.NewFromFloat(7.2), Spread: decimal.Zero, Enabled: true,
	}, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyTRX).Return(nil, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, "TON").Return(nil, nil)
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyTRX, entity.CurrencyUSDT).Return(nil, nil)
	cacheRepo.On("Set", ctx, "exchange:all_rates_display", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.NewFromFloat(720).Equal(result.USDTtoCNY))
}

func TestExchangeService_GetAllRatesDisplay_DB全部失败(t *testing.T) {
	svc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeServiceDeps()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return("", errors.New("cache miss"))

	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyCNY).Return(nil, errors.New("db error"))
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyTRX).Return(nil, errors.New("db error"))
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, "TON").Return(nil, errors.New("db error"))
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyTRX, entity.CurrencyUSDT).Return(nil, errors.New("db error"))
	cacheRepo.On("Set", ctx, "exchange:all_rates_display", mock.AnythingOfType("string"), 5*time.Minute).Return(nil)

	result, err := svc.GetAllRatesDisplay(ctx)

	// GetAllRatesDisplay 内部不返回错误，只是值为零
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.Zero.Equal(result.USDTtoCNY))
	assert.True(t, decimal.Zero.Equal(result.USDTtoTRX))
}

// ==================== NewExchangeService 构造测试 ====================

func TestNewExchangeService(t *testing.T) {
	exchangeRateRepo := new(mocks.MockExchangeRateRepository)
	exchangeRepo := new(mocks.MockExchangeRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	cacheRepo := new(mocks.MockCacheRepository)
	logger := zap.NewNop()

	svc := NewExchangeService(nil, exchangeRateRepo, exchangeRepo, walletRepo, txRepo, cacheRepo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, exchangeRateRepo, svc.exchangeRateRepo)
	assert.Equal(t, exchangeRepo, svc.exchangeRepo)
	assert.Equal(t, walletRepo, svc.walletRepo)
	assert.Equal(t, txRepo, svc.transactionRepo)
	assert.Equal(t, cacheRepo, svc.cacheRepo)
}

// ==================== ExchangePair 结构验证 ====================

func TestExchangePair_结构完整性(t *testing.T) {
	pair := &ExchangePair{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
		MinAmount:    decimal.NewFromInt(10),
		MaxAmount:    decimal.NewFromInt(50000),
	}

	assert.Equal(t, "USDT", pair.FromCurrency)
	assert.Equal(t, "TRX", pair.ToCurrency)
	assert.True(t, decimal.NewFromFloat(6.5).Equal(pair.Rate))
	assert.True(t, decimal.NewFromInt(10).Equal(pair.MinAmount))
	assert.True(t, decimal.NewFromInt(50000).Equal(pair.MaxAmount))
}
