package app

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// newTestExchangeUseCase 创建测试用的 ExchangeUseCase 及其 mock 依赖
func newTestExchangeUseCase() (*ExchangeUseCase, *mocks.MockExchangeRateRepository, *mocks.MockWalletRepository, *mocks.MockExchangeRepository, *mocks.MockTransactionRepository, *mocks.MockCacheRepository) {
	exchangeRateRepo := new(mocks.MockExchangeRateRepository)
	exchangeRepo := new(mocks.MockExchangeRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	cacheRepo := new(mocks.MockCacheRepository)

	exchangeSvc := service.NewExchangeService(nil, exchangeRateRepo, exchangeRepo, walletRepo, txRepo, cacheRepo, zap.NewNop())
	walletSvc := service.NewWalletService(walletRepo)

	uc := NewExchangeUseCase(exchangeSvc, walletSvc, zap.NewNop())
	return uc, exchangeRateRepo, walletRepo, exchangeRepo, txRepo, cacheRepo
}

// ==================== FindDirection 测试 ====================

func TestExchangeUseCase_FindDirection_成功(t *testing.T) {
	uc, _, _, _, _, _ := newTestExchangeUseCase()

	tests := []struct {
		name       string
		action     string
		wantFrom   string
		wantTo     string
		wantNil    bool
	}{
		{"USDT转CNY", "usdt_to_cny", entity.CurrencyUSDT, entity.CurrencyCNY, false},
		{"CNY转USDT", "cny_to_usdt", entity.CurrencyCNY, entity.CurrencyUSDT, false},
		{"USDT转TRX", "usdt_to_trx", entity.CurrencyUSDT, entity.CurrencyTRX, false},
		{"TRX转USDT", "trx_to_usdt", entity.CurrencyTRX, entity.CurrencyUSDT, false},
		{"USDT转TON", "usdt_to_ton", entity.CurrencyUSDT, "TON", false},
		{"不存在的方向", "btc_to_usdt", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.FindDirection(tt.action)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantFrom, result.FromCurrency)
				assert.Equal(t, tt.wantTo, result.ToCurrency)
			}
		})
	}
}

// ==================== ValidateExchangeAmount 测试 ====================

func TestExchangeUseCase_ValidateExchangeAmount(t *testing.T) {
	uc, _, _, _, _, _ := newTestExchangeUseCase()

	direction := &ExchangeDirection{
		FromCurrency:  entity.CurrencyUSDT,
		ToCurrency:    entity.CurrencyTRX,
		MinAmount:     decimal.NewFromInt(10),
		MinAmountText: "10 USDT",
	}

	tests := []struct {
		name    string
		amount  decimal.Decimal
		wantErr bool
		errMsg  string
	}{
		{"正常金额", decimal.NewFromInt(100), false, ""},
		{"等于最低金额", decimal.NewFromInt(10), false, ""},
		{"低于最低金额", decimal.NewFromInt(5), true, "最小兑换金额"},
		{"金额为0", decimal.Zero, true, "大于 0"},
		{"负数金额", decimal.NewFromInt(-1), true, "大于 0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.ValidateExchangeAmount(direction, tt.amount)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== GetBalance 测试 ====================

func TestExchangeUseCase_GetBalance_成功(t *testing.T) {
	uc, _, walletRepo, _, _, _ := newTestExchangeUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(500)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	balance, err := uc.GetBalance(ctx, 1, "USDT")

	assert.NoError(t, err)
	assert.True(t, decimal.NewFromInt(500).Equal(balance))
}

func TestExchangeUseCase_GetBalance_查询失败(t *testing.T) {
	uc, _, walletRepo, _, _, _ := newTestExchangeUseCase()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(nil, errors.New("db error"))

	_, err := uc.GetBalance(ctx, 1, "USDT")

	assert.Error(t, err)
}

// ==================== CheckBalance 测试 ====================

func TestExchangeUseCase_CheckBalance_充足(t *testing.T) {
	uc, _, walletRepo, _, _, _ := newTestExchangeUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(500), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	err := uc.CheckBalance(ctx, 1, "USDT", decimal.NewFromInt(100))

	assert.NoError(t, err)
}

func TestExchangeUseCase_CheckBalance_不足(t *testing.T) {
	uc, _, walletRepo, _, _, _ := newTestExchangeUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(50), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	err := uc.CheckBalance(ctx, 1, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "余额不足")
}

// ==================== CalculateExchange 测试 ====================

func TestExchangeUseCase_CalculateExchange(t *testing.T) {
	uc, _, _, _, _, _ := newTestExchangeUseCase()

	pair := &service.ExchangePair{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         decimal.NewFromFloat(6.5),
	}

	toAmount, fee := uc.CalculateExchange(pair, decimal.NewFromInt(100))

	assert.True(t, decimal.NewFromFloat(650).Equal(toAmount))
	assert.True(t, decimal.Zero.Equal(fee))
}

// ==================== SupportedDirections 测试 ====================

func TestSupportedDirections_完整性(t *testing.T) {
	// 验证支持的方向不为空且每个方向都有完整数据
	assert.Greater(t, len(SupportedDirections), 0)

	for i, d := range SupportedDirections {
		assert.NotEmpty(t, d.FromCurrency, "第 %d 个方向缺少 FromCurrency", i)
		assert.NotEmpty(t, d.ToCurrency, "第 %d 个方向缺少 ToCurrency", i)
		assert.NotEmpty(t, d.CallbackData, "第 %d 个方向缺少 CallbackData", i)
		assert.NotEmpty(t, d.ButtonText, "第 %d 个方向缺少 ButtonText", i)
		assert.True(t, d.MinAmount.GreaterThan(decimal.Zero), "第 %d 个方向 MinAmount 应大于 0", i)
	}
}

// ==================== GetAllRatesDisplay 测试 ====================

func TestExchangeUseCase_GetAllRatesDisplay_成功(t *testing.T) {
	uc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeUseCase()
	ctx := context.Background()

	// 缓存未命中
	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return("", errors.New("cache miss"))

	// 各汇率查询
	usdtCnyRate := &entity.ExchangeRate{Rate: decimal.NewFromFloat(7.2), Spread: decimal.Zero}
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyCNY).Return(usdtCnyRate, nil)

	usdtTrxRate := &entity.ExchangeRate{Rate: decimal.NewFromFloat(6.5), Spread: decimal.Zero}
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, entity.CurrencyTRX).Return(usdtTrxRate, nil)

	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyUSDT, "TON").Return(nil, nil)

	trxUsdtRate := &entity.ExchangeRate{Rate: decimal.NewFromFloat(0.15), Spread: decimal.Zero}
	exchangeRateRepo.On("FindByPair", ctx, entity.CurrencyTRX, entity.CurrencyUSDT).Return(trxUsdtRate, nil)

	// 缓存结果
	cacheRepo.On("Set", ctx, "exchange:all_rates_display", mock.Anything, mock.Anything).Return(nil)

	result, err := uc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestExchangeUseCase_GetAllRatesDisplay_从缓存读取(t *testing.T) {
	uc, _, _, _, _, cacheRepo := newTestExchangeUseCase()
	ctx := context.Background()

	// 返回有效的 JSON 缓存
	cached := `{"usdt_to_cny":"720","usdt_to_trx":"650"}`
	cacheRepo.On("Get", ctx, "exchange:all_rates_display").Return(cached, nil)

	result, err := uc.GetAllRatesDisplay(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// ==================== GetExchangePair 测试 ====================

func TestExchangeUseCase_GetExchangePair_成功(t *testing.T) {
	uc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeUseCase()
	ctx := context.Background()

	// 缓存未命中
	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))

	rate := &entity.ExchangeRate{
		Rate:      decimal.NewFromFloat(6.5),
		Spread:    decimal.Zero,
		MinAmount: decimal.NewFromInt(10),
		MaxAmount: decimal.NewFromInt(50000),
		Enabled:   true,
	}
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(rate, nil)
	cacheRepo.On("Set", ctx, "rate:USDT:TRX", mock.Anything, mock.Anything).Return(nil)

	result, err := uc.GetExchangePair(ctx, "USDT", "TRX")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT", result.FromCurrency)
	assert.Equal(t, "TRX", result.ToCurrency)
}

func TestExchangeUseCase_GetExchangePair_查询失败(t *testing.T) {
	uc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeUseCase()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(nil, errors.New("db error"))

	result, err := uc.GetExchangePair(ctx, "USDT", "TRX")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestExchangeUseCase_GetExchangePair_汇率不可用(t *testing.T) {
	uc, exchangeRateRepo, _, _, _, cacheRepo := newTestExchangeUseCase()
	ctx := context.Background()

	cacheRepo.On("Get", ctx, "rate:USDT:TRX").Return("", errors.New("cache miss"))
	// 返回 nil 表示不存在
	exchangeRateRepo.On("FindByPair", ctx, "USDT", "TRX").Return(nil, nil)

	result, err := uc.GetExchangePair(ctx, "USDT", "TRX")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== 构造函数测试 ====================

func TestNewExchangeUseCase(t *testing.T) {
	exchangeSvc := service.NewExchangeService(nil, nil, nil, nil, nil, nil, zap.NewNop())
	walletSvc := service.NewWalletService(nil)

	uc := NewExchangeUseCase(exchangeSvc, walletSvc, zap.NewNop())
	assert.NotNil(t, uc)
}
