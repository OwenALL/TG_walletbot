package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// ExchangeUseCase 闪兑用例层
type ExchangeUseCase struct {
	exchangeSvc *service.ExchangeService
	walletSvc   *service.WalletService
	logger      *zap.Logger
}

// NewExchangeUseCase 创建闪兑用例
func NewExchangeUseCase(
	exchangeSvc *service.ExchangeService,
	walletSvc *service.WalletService,
	logger *zap.Logger,
) *ExchangeUseCase {
	return &ExchangeUseCase{
		exchangeSvc: exchangeSvc,
		walletSvc:   walletSvc,
		logger:      logger,
	}
}

// ExchangeDirection 兑换方向定义
type ExchangeDirection struct {
	FromCurrency  string
	ToCurrency    string
	CallbackData  string
	ButtonText    string
	MinAmount     decimal.Decimal
	MinAmountText string // 展示用最低金额文本
}

// SupportedDirections 所有支持的兑换方向
var SupportedDirections = []ExchangeDirection{
	{
		FromCurrency:  entity.CurrencyUSDT,
		ToCurrency:    entity.CurrencyCNY,
		CallbackData:  "Exchange--usdt_to_cny",
		ButtonText:    "USDT => CNY",
		MinAmount:     decimal.NewFromInt(10),
		MinAmountText: "10 USDT",
	},
	{
		FromCurrency:  entity.CurrencyCNY,
		ToCurrency:    entity.CurrencyUSDT,
		CallbackData:  "Exchange--cny_to_usdt",
		ButtonText:    "CNY => USDT",
		MinAmount:     decimal.NewFromInt(10),
		MinAmountText: "10 CNY",
	},
	{
		FromCurrency:  entity.CurrencyUSDT,
		ToCurrency:    "TON",
		CallbackData:  "Exchange--usdt_to_ton",
		ButtonText:    "USDT => TON",
		MinAmount:     decimal.NewFromInt(10),
		MinAmountText: "10 USDT",
	},
	{
		FromCurrency:  entity.CurrencyUSDT,
		ToCurrency:    entity.CurrencyTRX,
		CallbackData:  "Exchange--usdt_to_trx",
		ButtonText:    "USDT => TRX",
		MinAmount:     decimal.NewFromInt(10),
		MinAmountText: "10 USDT",
	},
	{
		FromCurrency:  entity.CurrencyTRX,
		ToCurrency:    entity.CurrencyUSDT,
		CallbackData:  "Exchange--trx_to_usdt",
		ButtonText:    "TRX => USDT",
		MinAmount:     decimal.NewFromInt(100),
		MinAmountText: "100 TRX",
	},
}

// GetAllRatesDisplay 获取所有汇率展示
func (uc *ExchangeUseCase) GetAllRatesDisplay(ctx context.Context) (*service.AllRatesDisplay, error) {
	return uc.exchangeSvc.GetAllRatesDisplay(ctx)
}

// GetExchangePair 获取兑换对信息
func (uc *ExchangeUseCase) GetExchangePair(ctx context.Context, fromCurrency, toCurrency string) (*service.ExchangePair, error) {
	return uc.exchangeSvc.GetExchangePair(ctx, fromCurrency, toCurrency)
}

// FindDirection 根据 callback_data 查找兑换方向
func (uc *ExchangeUseCase) FindDirection(callbackAction string) *ExchangeDirection {
	for i := range SupportedDirections {
		// 从 callback_data 中提取 action 部分 (去掉 Exchange-- 前缀)
		if SupportedDirections[i].CallbackData == "Exchange--"+callbackAction {
			return &SupportedDirections[i]
		}
	}
	return nil
}

// ValidateExchangeAmount 验证兑换金额
func (uc *ExchangeUseCase) ValidateExchangeAmount(direction *ExchangeDirection, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return apperrors.NewBadRequest("金额必须大于 0")
	}
	if amount.LessThan(direction.MinAmount) {
		return apperrors.NewBadRequest("最小兑换金额为 " + direction.MinAmountText)
	}
	return nil
}

// CheckBalance 检查余额
func (uc *ExchangeUseCase) CheckBalance(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) error {
	return uc.walletSvc.CheckBalance(ctx, userID, currency, amount)
}

// CalculateExchange 计算兑换结果
func (uc *ExchangeUseCase) CalculateExchange(pair *service.ExchangePair, fromAmount decimal.Decimal) (toAmount, fee decimal.Decimal) {
	return uc.exchangeSvc.CalculateExchange(pair, fromAmount)
}

// ExecuteExchange 执行兑换
func (uc *ExchangeUseCase) ExecuteExchange(ctx context.Context, userID uint64, fromCurrency, toCurrency string, fromAmount decimal.Decimal, pair *service.ExchangePair) (*service.ExchangeResult, error) {
	return uc.exchangeSvc.ExecuteExchange(ctx, userID, fromCurrency, toCurrency, fromAmount, pair)
}

// GetBalance 获取用户余额
func (uc *ExchangeUseCase) GetBalance(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	wallet, err := uc.walletSvc.GetWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, err
	}
	return wallet.Balance, nil
}
