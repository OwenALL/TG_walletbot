package app

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
	pkgtron "github.com/TGlimmer/TG_walletbot/pkg/tron"
)

// WithdrawalUseCase 提币用例层
type WithdrawalUseCase struct {
	withdrawSvc      *service.WithdrawalService
	walletSvc        *service.WalletService
	userSvc          *service.UserService
	withdrawalSender port.WithdrawalSender // 可为 nil (TRON 未启用时)
	logger           *zap.Logger
}

// NewWithdrawalUseCase 创建提币用例
// withdrawalSender 可传 nil (TRON 未启用时不自动发送)
func NewWithdrawalUseCase(
	withdrawSvc *service.WithdrawalService,
	walletSvc *service.WalletService,
	userSvc *service.UserService,
	withdrawalSender port.WithdrawalSender,
	logger *zap.Logger,
) *WithdrawalUseCase {
	return &WithdrawalUseCase{
		withdrawSvc:      withdrawSvc,
		walletSvc:        walletSvc,
		userSvc:          userSvc,
		withdrawalSender: withdrawalSender,
		logger:           logger,
	}
}

// GetBalance 获取指定币种余额
func (uc *WithdrawalUseCase) GetBalance(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	wallet, err := uc.walletSvc.GetWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, err
	}
	return wallet.AvailableBalance(), nil
}

// ValidateAmount 验证提币金额
func (uc *WithdrawalUseCase) ValidateAmount(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) error {
	return uc.withdrawSvc.ValidateAmount(ctx, userID, currency, amount)
}

// ValidateAddress 验证提币地址
func (uc *WithdrawalUseCase) ValidateAddress(address string) bool {
	return pkgtron.ValidateAddress(address)
}

// GetWithdrawFee 获取提币手续费
func (uc *WithdrawalUseCase) GetWithdrawFee(ctx context.Context, currency string) decimal.Decimal {
	cfg := uc.withdrawSvc.GetWithdrawConfig(ctx, currency)
	return cfg.Fee
}

// WithdrawConfirmInfo 提币确认页信息
type WithdrawConfirmInfo struct {
	Currency     string
	Amount       decimal.Decimal
	Fee          decimal.Decimal
	ActualAmount decimal.Decimal
	ToAddress    string
	NeedPIN      bool
}

// BuildConfirmInfo 构建提币确认信息
func (uc *WithdrawalUseCase) BuildConfirmInfo(ctx context.Context, userID uint64, currency string, amount decimal.Decimal, address string) (*WithdrawConfirmInfo, error) {
	fee := uc.GetWithdrawFee(ctx, currency)
	actualAmount := amount.Sub(fee)

	if actualAmount.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("提币金额必须大于手续费 %s %s", fee.String(), currency))
	}

	// 判断是否需要 PIN 验证
	needPIN, _ := uc.userSvc.NeedPIN(ctx, userID, currency, amount)

	return &WithdrawConfirmInfo{
		Currency:     currency,
		Amount:       amount,
		Fee:          fee,
		ActualAmount: actualAmount,
		ToAddress:    address,
		NeedPIN:      needPIN,
	}, nil
}

// SubmitWithdrawal 提交提币请求
func (uc *WithdrawalUseCase) SubmitWithdrawal(ctx context.Context, userID uint64, currency, toAddress string, amount decimal.Decimal) (*entity.Withdrawal, error) {
	// 再次验证金额
	if err := uc.withdrawSvc.ValidateAmount(ctx, userID, currency, amount); err != nil {
		return nil, err
	}

	// 验证地址
	if !pkgtron.ValidateAddress(toAddress) {
		return nil, apperrors.NewInvalidAddress()
	}

	// 创建提币
	withdrawal, err := uc.withdrawSvc.CreateWithdrawal(ctx, userID, currency, toAddress, amount)
	if err != nil {
		uc.logger.Error("创建提币失败",
			zap.Uint64("user_id", userID),
			zap.String("currency", currency),
			zap.String("amount", amount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info("提币请求已创建",
		zap.Uint64("user_id", userID),
		zap.Uint64("withdrawal_id", withdrawal.ID),
		zap.String("currency", currency),
		zap.String("amount", amount.String()),
		zap.String("to_address", toAddress),
	)

	// 判断是否自动审核并发送
	if uc.withdrawSvc.IsAutoApprove(ctx, currency, amount) && uc.withdrawalSender != nil {
		uc.logger.Info("提币自动审核通过，启动异步发送",
			zap.Uint64("withdrawal_id", withdrawal.ID),
		)
		go uc.processAutoWithdrawal(withdrawal)
	}

	return withdrawal, nil
}

// processAutoWithdrawal 异步处理自动提币: 审核通过 -> 链上发送 -> 更新状态
func (uc *WithdrawalUseCase) processAutoWithdrawal(withdrawal *entity.Withdrawal) {
	ctx := context.Background()

	// 先标记为处理中
	if err := uc.withdrawSvc.ApproveWithdrawal(ctx, withdrawal.ID, 0); err != nil {
		uc.logger.Error("自动审核标记处理中失败",
			zap.Uint64("withdrawal_id", withdrawal.ID),
			zap.Error(err),
		)
		return
	}

	// 调用 WithdrawalSender 发送链上交易
	txHash, isManual, err := uc.withdrawalSender.SendWithdrawal(ctx, withdrawal.ToAddress, withdrawal.Currency, withdrawal.ActualAmount)
	if err != nil {
		uc.logger.Error("自动提币发送失败",
			zap.Uint64("withdrawal_id", withdrawal.ID),
			zap.Error(err),
		)
		// 标记为失败并退回余额
		_ = uc.withdrawSvc.FailWithdrawal(ctx, withdrawal.ID, "链上发送失败: "+err.Error())
		return
	}

	if isManual {
		// 超过阈值，需要人工处理，保持 processing 状态
		uc.logger.Info("提币金额超过阈值，等待人工处理",
			zap.Uint64("withdrawal_id", withdrawal.ID),
		)
		return
	}

	// 链上发送成功，完成提币
	if err := uc.withdrawSvc.CompleteWithdrawal(ctx, withdrawal.ID, txHash); err != nil {
		uc.logger.Error("更新提币完成状态失败",
			zap.Uint64("withdrawal_id", withdrawal.ID),
			zap.String("tx_hash", txHash),
			zap.Error(err),
		)
		return
	}

	uc.logger.Info("自动提币完成",
		zap.Uint64("withdrawal_id", withdrawal.ID),
		zap.String("tx_hash", txHash),
	)
}
