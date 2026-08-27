package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// ExchangeService 闪兑领域服务
type ExchangeService struct {
	db               *gorm.DB
	exchangeRateRepo port.ExchangeRateRepository
	exchangeRepo     port.ExchangeRepository
	walletRepo       port.WalletRepository
	transactionRepo  port.TransactionRepository
	cacheRepo        port.CacheRepository
	logger           *zap.Logger
}

// NewExchangeService 创建闪兑领域服务
func NewExchangeService(
	db *gorm.DB,
	exchangeRateRepo port.ExchangeRateRepository,
	exchangeRepo port.ExchangeRepository,
	walletRepo port.WalletRepository,
	transactionRepo port.TransactionRepository,
	cacheRepo port.CacheRepository,
	logger *zap.Logger,
) *ExchangeService {
	return &ExchangeService{
		db:               db,
		exchangeRateRepo: exchangeRateRepo,
		exchangeRepo:     exchangeRepo,
		walletRepo:       walletRepo,
		transactionRepo:  transactionRepo,
		cacheRepo:        cacheRepo,
		logger:           logger,
	}
}

// ExchangePair 兑换对信息
type ExchangePair struct {
	FromCurrency string
	ToCurrency   string
	Rate         decimal.Decimal // 有效汇率 (含加点)
	MinAmount    decimal.Decimal
	MaxAmount    decimal.Decimal
}

// ExchangeResult 兑换执行结果
type ExchangeResult struct {
	Exchange            *entity.Exchange
	FromNewBalance      decimal.Decimal
	ToNewBalance        decimal.Decimal
}

// AllRatesDisplay 所有汇率展示数据
type AllRatesDisplay struct {
	USDTtoCNY  decimal.Decimal // 100 USDT = ? CNY
	USDTtoTRX  decimal.Decimal // 100 USDT = ? TRX
	USDTtoTON  decimal.Decimal // 100 USDT = ? TON
	TRXtoUSDT  decimal.Decimal // 1000 TRX = ? USDT
}

// GetAllRatesDisplay 获取所有汇率展示数据 (Redis 缓存 5 分钟)
func (s *ExchangeService) GetAllRatesDisplay(ctx context.Context) (*AllRatesDisplay, error) {
	// 尝试从缓存读取
	cacheKey := "exchange:all_rates_display"
	cached, err := s.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var display AllRatesDisplay
		if json.Unmarshal([]byte(cached), &display) == nil {
			return &display, nil
		}
	}

	// 从数据库读取各汇率
	display := &AllRatesDisplay{}

	// USDT => CNY
	if rate, err := s.exchangeRateRepo.FindByPair(ctx, entity.CurrencyUSDT, entity.CurrencyCNY); err == nil && rate != nil {
		display.USDTtoCNY = rate.EffectiveRate().Mul(decimal.NewFromInt(100))
	}

	// USDT => TRX
	if rate, err := s.exchangeRateRepo.FindByPair(ctx, entity.CurrencyUSDT, entity.CurrencyTRX); err == nil && rate != nil {
		display.USDTtoTRX = rate.EffectiveRate().Mul(decimal.NewFromInt(100))
	}

	// USDT => TON
	if rate, err := s.exchangeRateRepo.FindByPair(ctx, entity.CurrencyUSDT, "TON"); err == nil && rate != nil {
		display.USDTtoTON = rate.EffectiveRate().Mul(decimal.NewFromInt(100))
	}

	// TRX => USDT
	if rate, err := s.exchangeRateRepo.FindByPair(ctx, entity.CurrencyTRX, entity.CurrencyUSDT); err == nil && rate != nil {
		display.TRXtoUSDT = rate.EffectiveRate().Mul(decimal.NewFromInt(1000))
	}

	// 缓存结果
	if data, err := json.Marshal(display); err == nil {
		_ = s.cacheRepo.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return display, nil
}

// GetExchangePair 获取兑换对信息 (含缓存)
func (s *ExchangeService) GetExchangePair(ctx context.Context, fromCurrency, toCurrency string) (*ExchangePair, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("rate:%s:%s", fromCurrency, toCurrency)
	cached, err := s.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		var pair ExchangePair
		if json.Unmarshal([]byte(cached), &pair) == nil {
			return &pair, nil
		}
	}

	// 从数据库读取
	rate, err := s.exchangeRateRepo.FindByPair(ctx, fromCurrency, toCurrency)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询汇率失败")
	}
	if rate == nil || !rate.Enabled {
		return nil, apperrors.NewExchangeRateUnavailable(fromCurrency + "/" + toCurrency)
	}

	pair := &ExchangePair{
		FromCurrency: fromCurrency,
		ToCurrency:   toCurrency,
		Rate:         rate.EffectiveRate(),
		MinAmount:    rate.MinAmount,
		MaxAmount:    rate.MaxAmount,
	}

	// 缓存 5 分钟
	if data, err := json.Marshal(pair); err == nil {
		_ = s.cacheRepo.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return pair, nil
}

// CalculateExchange 计算兑换结果
// 返回: 兑入金额, 手续费 (手续费包含在汇率加点中，显示为 0)
func (s *ExchangeService) CalculateExchange(pair *ExchangePair, fromAmount decimal.Decimal) (toAmount, fee decimal.Decimal) {
	toAmount = fromAmount.Mul(pair.Rate)
	fee = decimal.Zero
	return toAmount, fee
}

// ExecuteExchange 执行兑换 (事务)
func (s *ExchangeService) ExecuteExchange(ctx context.Context, userID uint64, fromCurrency, toCurrency string, fromAmount decimal.Decimal, pair *ExchangePair) (*ExchangeResult, error) {
	toAmount, fee := s.CalculateExchange(pair, fromAmount)

	var result ExchangeResult

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)

		// 1. 查询并锁定源币种钱包
		fromWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, fromCurrency)
		if err != nil {
			return apperrors.Wrap(err, "查询源币种钱包失败")
		}
		if fromWallet == nil {
			return apperrors.NewNotFound("钱包", fromCurrency)
		}
		if !fromWallet.HasSufficientBalance(fromAmount) {
			return apperrors.NewInsufficientBalance(fromCurrency)
		}

		// 2. 查询并锁定目标币种钱包
		toWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, toCurrency)
		if err != nil {
			return apperrors.Wrap(err, "查询目标币种钱包失败")
		}
		if toWallet == nil {
			return apperrors.NewNotFound("钱包", toCurrency)
		}

		// 3. 创建闪兑记录
		now := time.Now()
		exchange := &entity.Exchange{
			UserID:       userID,
			FromCurrency: fromCurrency,
			ToCurrency:   toCurrency,
			FromAmount:   fromAmount,
			ToAmount:     toAmount,
			Rate:         pair.Rate,
			Fee:          fee,
			Status:       1, // 已完成
			CreatedAt:    now,
		}
		if err := tx.WithContext(ctx).Create(exchange).Error; err != nil {
			return apperrors.Wrap(err, "创建闪兑记录失败")
		}

		// 4. 扣减源币种余额
		fromBalanceBefore := fromWallet.Balance
		fromBalanceAfter := fromBalanceBefore.Sub(fromAmount)
		if err := walletRepo.UpdateBalance(ctx, fromWallet.ID, fromAmount.Neg()); err != nil {
			return apperrors.Wrap(err, "扣减源币种余额失败")
		}

		// 5. 增加目标币种余额
		toBalanceBefore := toWallet.Balance
		toBalanceAfter := toBalanceBefore.Add(toAmount)
		if err := walletRepo.UpdateBalance(ctx, toWallet.ID, toAmount); err != nil {
			return apperrors.Wrap(err, "增加目标币种余额失败")
		}

		// 6. 创建兑出交易记录
		txRepo := newTxTransactionRepo(tx)
		outTx := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeExchangeOut,
			Currency:      fromCurrency,
			Amount:        fromAmount,
			Fee:           fee,
			BalanceBefore: fromBalanceBefore,
			BalanceAfter:  fromBalanceAfter,
			RelatedID:     &exchange.ID,
			RelatedType:   "exchange",
			Memo:          fmt.Sprintf("闪兑 %s => %s", fromCurrency, toCurrency),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, outTx); err != nil {
			return apperrors.Wrap(err, "创建兑出交易记录失败")
		}

		// 7. 创建兑入交易记录
		inTx := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeExchangeIn,
			Currency:      toCurrency,
			Amount:        toAmount,
			Fee:           decimal.Zero,
			BalanceBefore: toBalanceBefore,
			BalanceAfter:  toBalanceAfter,
			RelatedID:     &exchange.ID,
			RelatedType:   "exchange",
			Memo:          fmt.Sprintf("闪兑 %s => %s", fromCurrency, toCurrency),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, inTx); err != nil {
			return apperrors.Wrap(err, "创建兑入交易记录失败")
		}

		result.Exchange = exchange
		result.FromNewBalance = fromBalanceAfter
		result.ToNewBalance = toBalanceAfter
		return nil
	})

	if err != nil {
		s.logger.Error("闪兑执行失败",
			zap.Uint64("user_id", userID),
			zap.String("pair", fromCurrency+"/"+toCurrency),
			zap.String("amount", fromAmount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("闪兑成功",
		zap.Uint64("user_id", userID),
		zap.String("pair", fromCurrency+"/"+toCurrency),
		zap.String("from_amount", fromAmount.String()),
		zap.String("to_amount", toAmount.String()),
		zap.Uint64("exchange_id", result.Exchange.ID),
	)

	return &result, nil
}
