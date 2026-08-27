package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	"github.com/OwenALL/TG_walletbot/pkg/tron"
)

// DepositUseCase 充值用例层
type DepositUseCase struct {
	depositSvc *service.DepositService
	walletSvc  *service.WalletService
	logger     *zap.Logger
}

// NewDepositUseCase 创建充值用例
func NewDepositUseCase(depositSvc *service.DepositService, walletSvc *service.WalletService, logger *zap.Logger) *DepositUseCase {
	return &DepositUseCase{
		depositSvc: depositSvc,
		walletSvc:  walletSvc,
		logger:     logger,
	}
}

// DepositInfo 充值页面展示数据
type DepositInfo struct {
	Address string // 充值地址
	QRCode  []byte // QR Code PNG 字节数据
}

// GetDepositInfo 获取充值信息 (地址 + 二维码)
func (uc *DepositUseCase) GetDepositInfo(ctx context.Context, userID uint64) (*DepositInfo, error) {
	address, err := uc.depositSvc.GetUserDepositAddress(ctx, userID)
	if err != nil {
		uc.logger.Error("获取充值地址失败", zap.Uint64("user_id", userID), zap.Error(err))
		return nil, err
	}

	if address == "" {
		// 尚未分配充值地址
		return &DepositInfo{Address: ""}, nil
	}

	// 生成 QR Code
	qrData, err := tron.GenerateQRCode(address, 256)
	if err != nil {
		uc.logger.Error("生成二维码失败", zap.Uint64("user_id", userID), zap.Error(err))
		// 二维码生成失败不影响主流程，返回地址即可
		return &DepositInfo{Address: address}, nil
	}

	return &DepositInfo{
		Address: address,
		QRCode:  qrData,
	}, nil
}

// ProcessChainDeposit 处理链上监控到的充值
// 返回: 入账结果 (用于发送通知), 错误
func (uc *DepositUseCase) ProcessChainDeposit(ctx context.Context, txHash, fromAddress, toAddress, currency string, amount decimal.Decimal, confirmations int) (*DepositNotification, error) {
	deposit, isNew, err := uc.depositSvc.ProcessDeposit(ctx, txHash, fromAddress, toAddress, currency, amount, confirmations)
	if err != nil {
		uc.logger.Error("处理链上充值失败",
			zap.String("tx_hash", txHash),
			zap.String("currency", currency),
			zap.String("amount", amount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	// 只有新入账的充值才需要通知
	if deposit.IsCredited() && (isNew || deposit.Status == entity.DepositStatusCredited) && !deposit.Notified {
		// 获取入账后的余额
		balances, err := uc.walletSvc.GetBalanceMap(ctx, deposit.UserID)
		if err != nil {
			uc.logger.Error("获取余额失败", zap.Uint64("user_id", deposit.UserID), zap.Error(err))
			return nil, err
		}

		return &DepositNotification{
			UserID:     deposit.UserID,
			Currency:   deposit.Currency,
			Amount:     deposit.Amount,
			TxHash:     deposit.TxHash,
			NewBalance: balances[deposit.Currency],
		}, nil
	}

	return nil, nil
}

// DepositNotification 充值入账通知数据
type DepositNotification struct {
	UserID     uint64
	Currency   string
	Amount     decimal.Decimal
	TxHash     string
	NewBalance decimal.Decimal
}

// ListPendingDeposits 获取待确认的充值列表 (链上监控用)
func (uc *DepositUseCase) ListPendingDeposits(ctx context.Context) ([]*entity.Deposit, error) {
	return uc.depositSvc.ListPendingDeposits(ctx)
}
