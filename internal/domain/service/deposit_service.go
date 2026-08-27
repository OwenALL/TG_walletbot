package service

import (
	"context"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// DepositService 充值领域服务
type DepositService struct {
	depositRepo  port.DepositRepository
	walletRepo   port.WalletRepository
	txRepo       port.TransactionRepository
	configRepo   port.SystemConfigRepository
}

// NewDepositService 创建充值领域服务
func NewDepositService(
	depositRepo port.DepositRepository,
	walletRepo port.WalletRepository,
	txRepo port.TransactionRepository,
	configRepo port.SystemConfigRepository,
) *DepositService {
	return &DepositService{
		depositRepo: depositRepo,
		walletRepo:  walletRepo,
		txRepo:      txRepo,
		configRepo:  configRepo,
	}
}

// GetRequiredConfirmations 获取充值所需确认数
func (s *DepositService) GetRequiredConfirmations(ctx context.Context) int {
	defaultConf := 3
	val, err := s.configRepo.Get(ctx, entity.ConfigDepositConfirmations)
	if err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return defaultConf
}

// ProcessDeposit 处理充值到账
// 1. 检查交易是否已存在 (防重复)
// 2. 创建充值记录
// 3. 等待确认后更新余额 + 创建交易流水
// 返回: 充值记录, 是否为新记录, 错误
func (s *DepositService) ProcessDeposit(ctx context.Context, txHash, fromAddress, toAddress, currency string, amount decimal.Decimal, confirmations int) (*entity.Deposit, bool, error) {
	// 查重
	existing, err := s.depositRepo.FindByTxHash(ctx, txHash)
	if err != nil {
		return nil, false, apperrors.Wrap(err, "查询充值记录失败")
	}
	if existing != nil {
		// 已存在，更新确认数
		if existing.IsCredited() {
			return existing, false, nil
		}
		return s.updateConfirmations(ctx, existing, confirmations)
	}

	// 根据充值地址查找对应的钱包
	wallet, err := s.walletRepo.FindByDepositAddress(ctx, toAddress)
	if err != nil {
		return nil, false, apperrors.Wrap(err, "查询充值地址失败")
	}
	if wallet == nil {
		return nil, false, apperrors.NewBadRequest("未知的充值地址")
	}

	// 创建充值记录
	deposit := &entity.Deposit{
		UserID:        wallet.UserID,
		Currency:      currency,
		Amount:        amount,
		FromAddress:   fromAddress,
		ToAddress:     toAddress,
		TxHash:        txHash,
		Confirmations: confirmations,
		Status:        entity.DepositStatusPending,
	}

	requiredConf := s.GetRequiredConfirmations(ctx)
	if confirmations >= requiredConf {
		deposit.Status = entity.DepositStatusConfirmed
	}

	if err := s.depositRepo.Create(ctx, deposit); err != nil {
		return nil, false, apperrors.Wrap(err, "创建充值记录失败")
	}

	// 如果确认数已满足，直接入账
	if deposit.Status == entity.DepositStatusConfirmed {
		if err := s.creditDeposit(ctx, deposit, wallet); err != nil {
			return deposit, true, err
		}
	}

	return deposit, true, nil
}

// updateConfirmations 更新充值确认数
func (s *DepositService) updateConfirmations(ctx context.Context, deposit *entity.Deposit, confirmations int) (*entity.Deposit, bool, error) {
	if confirmations <= deposit.Confirmations {
		return deposit, false, nil
	}

	if err := s.depositRepo.UpdateConfirmations(ctx, deposit.ID, confirmations); err != nil {
		return deposit, false, apperrors.Wrap(err, "更新确认数失败")
	}
	deposit.Confirmations = confirmations

	requiredConf := s.GetRequiredConfirmations(ctx)
	if confirmations >= requiredConf && deposit.Status == entity.DepositStatusPending {
		// 确认数满足，入账
		deposit.Status = entity.DepositStatusConfirmed
		if err := s.depositRepo.UpdateStatus(ctx, deposit.ID, entity.DepositStatusConfirmed); err != nil {
			return deposit, false, apperrors.Wrap(err, "更新充值状态失败")
		}

		wallet, err := s.walletRepo.FindByDepositAddress(ctx, deposit.ToAddress)
		if err != nil || wallet == nil {
			return deposit, false, apperrors.Wrap(err, "查询钱包失败")
		}

		if err := s.creditDeposit(ctx, deposit, wallet); err != nil {
			return deposit, false, err
		}
	}

	return deposit, false, nil
}

// creditDeposit 执行充值入账: 更新余额 + 创建交易流水
func (s *DepositService) creditDeposit(ctx context.Context, deposit *entity.Deposit, wallet *entity.Wallet) error {
	// 获取当前余额 (入账前)
	balanceBefore := wallet.Balance

	// 更新余额
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, deposit.Amount); err != nil {
		return apperrors.Wrap(err, "充值入账失败")
	}

	// 创建交易流水
	tx := &entity.Transaction{
		UserID:        deposit.UserID,
		Type:          entity.TxTypeDeposit,
		Currency:      deposit.Currency,
		Amount:        deposit.Amount,
		Fee:           decimal.Zero,
		BalanceBefore: balanceBefore,
		BalanceAfter:  balanceBefore.Add(deposit.Amount),
		RelatedID:     &deposit.ID,
		RelatedType:   "deposit",
		Memo:          "充值到账 " + deposit.TxHash,
	}
	if err := s.txRepo.Create(ctx, tx); err != nil {
		return apperrors.Wrap(err, "创建充值流水失败")
	}

	// 更新充值状态为已到账
	now := time.Now()
	deposit.Status = entity.DepositStatusCredited
	deposit.ConfirmedAt = &now
	if err := s.depositRepo.UpdateStatus(ctx, deposit.ID, entity.DepositStatusCredited); err != nil {
		return apperrors.Wrap(err, "更新充值状态失败")
	}

	return nil
}

// GetUserDepositAddress 获取用户的充值地址
func (s *DepositService) GetUserDepositAddress(ctx context.Context, userID uint64) (string, error) {
	// 所有币种共享同一个充值地址，取 USDT 钱包的地址即可
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, userID, entity.CurrencyUSDT)
	if err != nil {
		return "", apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return "", apperrors.NewNotFound("钱包", userID)
	}
	return wallet.DepositAddress, nil
}

// SetUserDepositAddress 设置用户充值地址 (注册时调用)
// 更新该用户所有链上钱包 (USDT/TRX) 的充值地址
func (s *DepositService) SetUserDepositAddress(ctx context.Context, userID uint64, address string) error {
	if err := s.walletRepo.UpdateDepositAddress(ctx, userID, address); err != nil {
		return apperrors.Wrap(err, "更新充值地址失败")
	}
	return nil
}

// ListPendingDeposits 获取待确认的充值列表 (链上监控用)
func (s *DepositService) ListPendingDeposits(ctx context.Context) ([]*entity.Deposit, error) {
	return s.depositRepo.ListPending(ctx)
}

// DepositCreditResult 充值入账结果
type DepositCreditResult struct {
	Deposit    *entity.Deposit
	UserID     uint64
	Currency   string
	Amount     decimal.Decimal
	NewBalance decimal.Decimal
}
