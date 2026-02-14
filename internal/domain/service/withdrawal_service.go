package service

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// WithdrawalService 提币领域服务
type WithdrawalService struct {
	withdrawalRepo port.WithdrawalRepository
	walletRepo     port.WalletRepository
	txRepo         port.TransactionRepository
	configRepo     port.SystemConfigRepository
}

// NewWithdrawalService 创建提币领域服务
func NewWithdrawalService(
	withdrawalRepo port.WithdrawalRepository,
	walletRepo port.WalletRepository,
	txRepo port.TransactionRepository,
	configRepo port.SystemConfigRepository,
) *WithdrawalService {
	return &WithdrawalService{
		withdrawalRepo: withdrawalRepo,
		walletRepo:     walletRepo,
		txRepo:         txRepo,
		configRepo:     configRepo,
	}
}

// WithdrawConfig 提币配置 (从 system_configs 读取)
type WithdrawConfig struct {
	MinAmount      decimal.Decimal // 最低提币
	MaxAmount      decimal.Decimal // 单笔上限
	DailyMaxAmount decimal.Decimal // 单日上限
	Fee            decimal.Decimal // 手续费/笔
	AutoThreshold  decimal.Decimal // 自动审核阈值
}

// GetWithdrawConfig 获取指定币种的提币配置
func (s *WithdrawalService) GetWithdrawConfig(ctx context.Context, currency string) *WithdrawConfig {
	cfg := &WithdrawConfig{}
	switch currency {
	case entity.CurrencyUSDT:
		cfg.MinAmount = s.getConfigDecimal(ctx, entity.ConfigWithdrawUSDTMin, "5")
		cfg.MaxAmount = s.getConfigDecimal(ctx, entity.ConfigWithdrawUSDTMax, "50000")
		cfg.DailyMaxAmount = s.getConfigDecimal(ctx, entity.ConfigWithdrawUSDTDailyMax, "100000")
		cfg.Fee = s.getConfigDecimal(ctx, entity.ConfigWithdrawUSDTFee, "1")
		cfg.AutoThreshold = s.getConfigDecimal(ctx, entity.ConfigWithdrawUSDTAutoThreshold, "1000")
	case entity.CurrencyTRX:
		cfg.MinAmount = s.getConfigDecimal(ctx, entity.ConfigWithdrawTRXMin, "50")
		cfg.MaxAmount = s.getConfigDecimal(ctx, entity.ConfigWithdrawTRXMax, "500000")
		cfg.DailyMaxAmount = decimal.NewFromInt(1000000) // TRX 日限额默认
		cfg.Fee = s.getConfigDecimal(ctx, entity.ConfigWithdrawTRXFee, "1")
		cfg.AutoThreshold = decimal.NewFromInt(10000) // TRX 自动审核阈值默认
	default:
		// 不支持的币种，设置极高阈值
		cfg.MinAmount = decimal.NewFromInt(999999)
		cfg.MaxAmount = decimal.Zero
	}
	return cfg
}

// ValidateAmount 验证提币金额
// 返回: 错误信息 (nil 表示通过)
func (s *WithdrawalService) ValidateAmount(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) error {
	cfg := s.GetWithdrawConfig(ctx, currency)

	// 检查最低限额
	if amount.LessThan(cfg.MinAmount) {
		return apperrors.NewWithdrawBelowMin(cfg.MinAmount.String(), currency)
	}

	// 检查单笔上限
	if amount.GreaterThan(cfg.MaxAmount) {
		return apperrors.NewWithdrawExceedMax(cfg.MaxAmount.String(), currency)
	}

	// 检查余额
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, userID, currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return apperrors.NewNotFound("钱包", currency)
	}

	// 需要金额 + 手续费
	totalNeeded := amount.Add(cfg.Fee)
	if !wallet.HasSufficientBalance(totalNeeded) {
		return apperrors.NewInsufficientBalance(currency)
	}

	// 检查日限额
	todaySpent, err := s.withdrawalRepo.SumTodayByUserID(ctx, userID, currency)
	if err != nil {
		return apperrors.Wrap(err, "查询今日提币总额失败")
	}
	if todaySpent.Add(amount).GreaterThan(cfg.DailyMaxAmount) {
		return apperrors.NewWithdrawDailyExceeded()
	}

	return nil
}

// CreateWithdrawal 创建提币请求
// 1. 冻结余额 (金额 + 手续费)
// 2. 创建提币记录
// 3. 判断自动/人工审核
func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID uint64, currency, toAddress string, amount decimal.Decimal) (*entity.Withdrawal, error) {
	cfg := s.GetWithdrawConfig(ctx, currency)
	fee := cfg.Fee
	actualAmount := amount.Sub(fee)

	// 获取钱包并冻结余额
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, userID, currency)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return nil, apperrors.NewNotFound("钱包", currency)
	}

	// 冻结金额 (原始金额，手续费从金额中扣除)
	if err := s.walletRepo.FreezeBalance(ctx, wallet.ID, amount); err != nil {
		return nil, apperrors.NewInsufficientBalance(currency)
	}

	// 创建提币记录
	withdrawal := &entity.Withdrawal{
		UserID:       userID,
		Currency:     currency,
		Amount:       amount,
		Fee:          fee,
		ActualAmount: actualAmount,
		ToAddress:    toAddress,
		Status:       entity.WithdrawalStatusPending,
	}

	if err := s.withdrawalRepo.Create(ctx, withdrawal); err != nil {
		// 创建失败需要解冻
		_ = s.walletRepo.UnfreezeBalance(ctx, wallet.ID, amount)
		return nil, apperrors.Wrap(err, "创建提币记录失败")
	}

	return withdrawal, nil
}

// ApproveWithdrawal 审核通过提币 (管理后台调用)
func (s *WithdrawalService) ApproveWithdrawal(ctx context.Context, withdrawalID uint64, reviewerID uint64) error {
	withdrawal, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if withdrawal == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}
	if !withdrawal.IsPending() {
		return apperrors.NewBadRequest("该提币请求已处理")
	}

	now := time.Now()
	withdrawal.Status = entity.WithdrawalStatusProcessing
	withdrawal.ReviewerID = &reviewerID
	withdrawal.ReviewedAt = &now

	return s.withdrawalRepo.Update(ctx, withdrawal)
}

// CompleteWithdrawal 完成提币 (链上发送成功后调用)
// 1. 扣减余额 (解冻 + 扣款)
// 2. 更新提币记录
// 3. 创建交易流水
func (s *WithdrawalService) CompleteWithdrawal(ctx context.Context, withdrawalID uint64, txHash string) error {
	withdrawal, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if withdrawal == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}

	// 获取钱包
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, withdrawal.UserID, withdrawal.Currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return apperrors.NewNotFound("钱包", withdrawal.Currency)
	}

	balanceBefore := wallet.Balance

	// 解冻余额
	if err := s.walletRepo.UnfreezeBalance(ctx, wallet.ID, withdrawal.Amount); err != nil {
		return apperrors.Wrap(err, "解冻余额失败")
	}

	// 扣减余额
	negAmount := withdrawal.Amount.Neg()
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, negAmount); err != nil {
		return apperrors.Wrap(err, "扣减余额失败")
	}

	// 创建交易流水
	tx := &entity.Transaction{
		UserID:        withdrawal.UserID,
		Type:          entity.TxTypeWithdraw,
		Currency:      withdrawal.Currency,
		Amount:        withdrawal.Amount,
		Fee:           withdrawal.Fee,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceBefore.Sub(withdrawal.Amount),
		RelatedID:     &withdrawal.ID,
		RelatedType:   "withdrawal",
		Memo:          "提币 " + withdrawal.ToAddress,
	}
	if err := s.txRepo.Create(ctx, tx); err != nil {
		return apperrors.Wrap(err, "创建提币流水失败")
	}

	// 更新提币记录状态
	now := time.Now()
	withdrawal.Status = entity.WithdrawalStatusCompleted
	withdrawal.TxHash = txHash
	withdrawal.CompletedAt = &now

	return s.withdrawalRepo.Update(ctx, withdrawal)
}

// RejectWithdrawal 拒绝提币 (解冻余额)
func (s *WithdrawalService) RejectWithdrawal(ctx context.Context, withdrawalID uint64, reviewerID uint64, reason string) error {
	withdrawal, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if withdrawal == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}
	if !withdrawal.IsPending() {
		return apperrors.NewBadRequest("该提币请求已处理")
	}

	// 解冻余额
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, withdrawal.UserID, withdrawal.Currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet != nil {
		_ = s.walletRepo.UnfreezeBalance(ctx, wallet.ID, withdrawal.Amount)
	}

	// 更新状态
	now := time.Now()
	withdrawal.Status = entity.WithdrawalStatusRejected
	withdrawal.ReviewerID = &reviewerID
	withdrawal.ReviewNote = reason
	withdrawal.ReviewedAt = &now

	return s.withdrawalRepo.Update(ctx, withdrawal)
}

// FailWithdrawal 标记提币失败 (链上发送失败)
func (s *WithdrawalService) FailWithdrawal(ctx context.Context, withdrawalID uint64, reason string) error {
	withdrawal, err := s.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if withdrawal == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}

	// 解冻余额
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, withdrawal.UserID, withdrawal.Currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet != nil {
		_ = s.walletRepo.UnfreezeBalance(ctx, wallet.ID, withdrawal.Amount)
	}

	withdrawal.Status = entity.WithdrawalStatusFailed
	withdrawal.ReviewNote = reason
	return s.withdrawalRepo.Update(ctx, withdrawal)
}

// IsAutoApprove 判断金额是否可自动审核
func (s *WithdrawalService) IsAutoApprove(ctx context.Context, currency string, amount decimal.Decimal) bool {
	cfg := s.GetWithdrawConfig(ctx, currency)
	return amount.LessThan(cfg.AutoThreshold)
}

// getConfigDecimal 从系统配置获取 decimal 值
func (s *WithdrawalService) getConfigDecimal(ctx context.Context, key, defaultVal string) decimal.Decimal {
	val, err := s.configRepo.Get(ctx, key)
	if err != nil || val == "" {
		val = defaultVal
	}
	result, err := decimal.NewFromString(val)
	if err != nil {
		result, _ = decimal.NewFromString(defaultVal)
	}
	return result
}

