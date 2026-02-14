package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// InvestmentSummary 用户投资汇总信息
type InvestmentSummary struct {
	TotalAmount decimal.Decimal // 总持仓金额
	TotalProfit decimal.Decimal // 累计收益
	Count       int             // 持仓笔数
}

// FinanceConfig 余额宝配置参数
type FinanceConfig struct {
	MinInvest  decimal.Decimal // 最低投资金额
	MaxInvest  decimal.Decimal // 最高投资金额
	AnnualRate decimal.Decimal // 年化利率 (百分比)
}

// FinanceService 余额宝领域服务
type FinanceService struct {
	db               *gorm.DB
	financeRepo      port.FinanceInvestmentRepository
	walletRepo       port.WalletRepository
	transactionRepo  port.TransactionRepository
	systemConfigRepo port.SystemConfigRepository
	logger           *zap.Logger
}

// NewFinanceService 创建余额宝领域服务
func NewFinanceService(
	db *gorm.DB,
	financeRepo port.FinanceInvestmentRepository,
	walletRepo port.WalletRepository,
	transactionRepo port.TransactionRepository,
	systemConfigRepo port.SystemConfigRepository,
	logger *zap.Logger,
) *FinanceService {
	return &FinanceService{
		db:               db,
		financeRepo:      financeRepo,
		walletRepo:       walletRepo,
		transactionRepo:  transactionRepo,
		systemConfigRepo: systemConfigRepo,
		logger:           logger,
	}
}

// GetFinanceConfig 从系统配置获取余额宝参数
func (s *FinanceService) GetFinanceConfig(ctx context.Context) (*FinanceConfig, error) {
	configs, err := s.systemConfigRepo.GetMultiple(ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, "读取余额宝配置失败")
	}

	cfg := &FinanceConfig{
		MinInvest:  decimal.NewFromInt(10),
		MaxInvest:  decimal.NewFromInt(500000),
		AnnualRate: decimal.NewFromInt(8),
	}

	if v, ok := configs[entity.ConfigFinanceMin]; ok && v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil {
			cfg.MinInvest = parsed
		}
	}
	if v, ok := configs[entity.ConfigFinanceMax]; ok && v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil {
			cfg.MaxInvest = parsed
		}
	}
	if v, ok := configs[entity.ConfigFinanceAnnualRate]; ok && v != "" {
		if parsed, err := decimal.NewFromString(v); err == nil {
			cfg.AnnualRate = parsed
		}
	}

	return cfg, nil
}

// Invest 买入余额宝
// 从 USDT 余额扣除指定金额，创建投资记录和交易流水
func (s *FinanceService) Invest(ctx context.Context, userID uint64, amount decimal.Decimal) (*entity.FinanceInvestment, error) {
	// 读取配置
	cfg, err := s.GetFinanceConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 验证金额范围
	if amount.LessThan(cfg.MinInvest) {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("最低投资金额为 %s USDT", cfg.MinInvest.String()))
	}
	if amount.GreaterThan(cfg.MaxInvest) {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("最高投资金额为 %s USDT", cfg.MaxInvest.String()))
	}

	var investment *entity.FinanceInvestment

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 1. 查询并锁定 USDT 钱包
		wallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, entity.CurrencyUSDT)
		if err != nil {
			return apperrors.Wrap(err, "查询 USDT 钱包失败")
		}
		if wallet == nil {
			return apperrors.NewNotFound("钱包", entity.CurrencyUSDT)
		}
		if !wallet.HasSufficientBalance(amount) {
			return apperrors.NewInsufficientBalance(entity.CurrencyUSDT)
		}

		// 2. 扣减 USDT 余额
		balanceBefore := wallet.Balance
		if err := walletRepo.UpdateBalance(ctx, wallet.ID, amount.Neg()); err != nil {
			return apperrors.Wrap(err, "扣减 USDT 余额失败")
		}
		balanceAfter := balanceBefore.Sub(amount)

		// 3. 创建投资记录
		now := time.Now()
		investment = &entity.FinanceInvestment{
			UserID:        userID,
			Amount:        amount,
			AnnualRate:    cfg.AnnualRate,
			Status:        entity.FinanceStatusHolding,
			InterestStart: now.Add(24 * time.Hour),
			TotalProfit:   decimal.Zero,
			CreatedAt:     now,
		}
		if err := tx.WithContext(ctx).Create(investment).Error; err != nil {
			return apperrors.Wrap(err, "创建投资记录失败")
		}

		// 4. 创建 finance_in 交易流水
		transaction := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeFinanceIn,
			Currency:      entity.CurrencyUSDT,
			Amount:        amount,
			Fee:           decimal.Zero,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedID:     &investment.ID,
			RelatedType:   "finance_investment",
			Memo:          fmt.Sprintf("余额宝买入 %s USDT", amount.String()),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建交易流水失败")
		}

		return nil
	})

	if err != nil {
		s.logger.Error("余额宝买入失败",
			zap.Uint64("user_id", userID),
			zap.String("amount", amount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("余额宝买入成功",
		zap.Uint64("user_id", userID),
		zap.String("amount", amount.String()),
		zap.Uint64("investment_id", investment.ID),
	)

	return investment, nil
}

// Withdraw 取出余额宝投资
// 将投资金额归还到 USDT 余额，更新投资状态为已取出
func (s *FinanceService) Withdraw(ctx context.Context, investmentID uint64, userID uint64) (*entity.FinanceInvestment, error) {
	// 查询投资记录
	investment, err := s.financeRepo.FindByID(ctx, investmentID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询投资记录失败")
	}
	if investment == nil {
		return nil, apperrors.NewNotFound("投资记录", investmentID)
	}

	// 验证归属
	if investment.UserID != userID {
		return nil, apperrors.NewForbidden("无权操作该投资记录")
	}

	// 验证状态
	if !investment.IsHolding() {
		return nil, apperrors.NewBadRequest("该投资已取出")
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 1. 查询并锁定 USDT 钱包
		wallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, entity.CurrencyUSDT)
		if err != nil {
			return apperrors.Wrap(err, "查询 USDT 钱包失败")
		}
		if wallet == nil {
			return apperrors.NewNotFound("钱包", entity.CurrencyUSDT)
		}

		// 2. 更新投资状态为已取出
		now := time.Now()
		investment.Status = entity.FinanceStatusWithdrawn
		investment.WithdrawnAt = &now
		if err := tx.WithContext(ctx).Save(investment).Error; err != nil {
			return apperrors.Wrap(err, "更新投资状态失败")
		}

		// 3. 增加 USDT 余额 (归还本金)
		balanceBefore := wallet.Balance
		if err := walletRepo.UpdateBalance(ctx, wallet.ID, investment.Amount); err != nil {
			return apperrors.Wrap(err, "归还 USDT 余额失败")
		}
		balanceAfter := balanceBefore.Add(investment.Amount)

		// 4. 创建 finance_out 交易流水
		transaction := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeFinanceOut,
			Currency:      entity.CurrencyUSDT,
			Amount:        investment.Amount,
			Fee:           decimal.Zero,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedID:     &investment.ID,
			RelatedType:   "finance_investment",
			Memo:          fmt.Sprintf("余额宝取出 %s USDT", investment.Amount.String()),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建交易流水失败")
		}

		return nil
	})

	if err != nil {
		s.logger.Error("余额宝取出失败",
			zap.Uint64("user_id", userID),
			zap.Uint64("investment_id", investmentID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("余额宝取出成功",
		zap.Uint64("user_id", userID),
		zap.Uint64("investment_id", investmentID),
		zap.String("amount", investment.Amount.String()),
	)

	return investment, nil
}

// GetUserInvestments 获取用户持有中的投资列表
func (s *FinanceService) GetUserInvestments(ctx context.Context, userID uint64) ([]*entity.FinanceInvestment, error) {
	investments, err := s.financeRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询投资列表失败")
	}
	return investments, nil
}

// GetUserInvestmentSummary 获取用户投资汇总统计
func (s *FinanceService) GetUserInvestmentSummary(ctx context.Context, userID uint64) (*InvestmentSummary, error) {
	investments, err := s.financeRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询投资列表失败")
	}

	summary := &InvestmentSummary{
		TotalAmount: decimal.Zero,
		TotalProfit: decimal.Zero,
		Count:       len(investments),
	}

	for _, inv := range investments {
		summary.TotalAmount = summary.TotalAmount.Add(inv.Amount)
		summary.TotalProfit = summary.TotalProfit.Add(inv.TotalProfit)
	}

	return summary, nil
}

// DistributeDailyProfits 每日派息
// 查询所有 status=HOLDING 且 interest_start <= now 的投资，计算并发放日收益
// 使用 Redis 防止同一天重复派息
// 返回成功处理的投资数量
func (s *FinanceService) DistributeDailyProfits(ctx context.Context) (int, error) {
	now := time.Now()

	// 获取当前年化利率 (从 DB 读取，支持动态调整)
	rateStr, err := s.systemConfigRepo.Get(ctx, entity.ConfigFinanceAnnualRate)
	if err != nil {
		s.logger.Warn("读取年化利率配置失败，使用默认值 8%", zap.Error(err))
		rateStr = "8"
	}
	currentRate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		currentRate = 8.0
	}
	annualRate := decimal.NewFromFloat(currentRate)

	// 查询所有持有中的投资
	investments, err := s.financeRepo.ListAllActive(ctx)
	if err != nil {
		return 0, apperrors.Wrap(err, "查询活跃投资列表失败")
	}

	processedCount := 0

	for _, inv := range investments {
		// 检查是否已到计息时间
		if now.Before(inv.InterestStart) {
			continue
		}

		// 计算日收益: 持仓金额 * 年化 / 100 / 365
		dailyProfit := inv.Amount.Mul(annualRate).Div(decimal.NewFromInt(100)).Div(decimal.NewFromInt(365))
		if dailyProfit.LessThanOrEqual(decimal.Zero) {
			continue
		}

		// 每笔投资单独事务，失败不影响其他
		if err := s.distributeSingleProfit(ctx, inv, dailyProfit); err != nil {
			s.logger.Error("单笔派息失败",
				zap.Uint64("investment_id", inv.ID),
				zap.Uint64("user_id", inv.UserID),
				zap.Error(err),
			)
			continue
		}

		processedCount++
	}

	return processedCount, nil
}

// distributeSingleProfit 为单笔投资发放收益
func (s *FinanceService) distributeSingleProfit(ctx context.Context, inv *entity.FinanceInvestment, dailyProfit decimal.Decimal) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 1. 查询并锁定 USDT 钱包
		wallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, inv.UserID, entity.CurrencyUSDT)
		if err != nil {
			return apperrors.Wrap(err, "查询钱包失败")
		}
		if wallet == nil {
			return apperrors.NewNotFound("钱包", entity.CurrencyUSDT)
		}

		// 2. 增加 USDT 余额 (收益到账)
		now := time.Now()
		balanceBefore := wallet.Balance
		if err := walletRepo.UpdateBalance(ctx, wallet.ID, dailyProfit); err != nil {
			return apperrors.Wrap(err, "增加收益余额失败")
		}
		balanceAfter := balanceBefore.Add(dailyProfit)

		// 3. 创建 finance_profit 交易流水
		transaction := &entity.Transaction{
			UserID:        inv.UserID,
			Type:          entity.TxTypeFinanceProfit,
			Currency:      entity.CurrencyUSDT,
			Amount:        dailyProfit,
			Fee:           decimal.Zero,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedID:     &inv.ID,
			RelatedType:   "finance_investment",
			Memo:          fmt.Sprintf("余额宝日收益 (本金 %s USDT)", inv.Amount.String()),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建收益流水失败")
		}

		// 4. 更新投资记录的累计收益
		newTotalProfit := inv.TotalProfit.Add(dailyProfit)
		if err := tx.WithContext(ctx).Model(&entity.FinanceInvestment{}).
			Where("id = ?", inv.ID).
			Update("total_profit", newTotalProfit).Error; err != nil {
			return apperrors.Wrap(err, "更新累计收益失败")
		}

		return nil
	})
}
