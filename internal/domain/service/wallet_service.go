package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// WalletService 钱包领域服务
type WalletService struct {
	walletRepo port.WalletRepository
}

// NewWalletService 创建钱包领域服务
func NewWalletService(walletRepo port.WalletRepository) *WalletService {
	return &WalletService{walletRepo: walletRepo}
}

// GetWallets 获取用户所有钱包
func (s *WalletService) GetWallets(ctx context.Context, userID uint64) ([]*entity.Wallet, error) {
	wallets, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}
	return wallets, nil
}

// GetWallet 获取用户指定币种钱包
func (s *WalletService) GetWallet(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, userID, currency)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return nil, apperrors.NewNotFound("钱包", currency)
	}
	return wallet, nil
}

// GetBalanceMap 获取用户各币种余额 (用于显示主页)
func (s *WalletService) GetBalanceMap(ctx context.Context, userID uint64) (map[string]decimal.Decimal, error) {
	wallets, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}
	result := make(map[string]decimal.Decimal)
	for _, w := range wallets {
		result[w.Currency] = w.Balance
	}
	// 确保三种币种都存在
	for _, c := range []string{entity.CurrencyUSDT, entity.CurrencyTRX, entity.CurrencyCNY} {
		if _, ok := result[c]; !ok {
			result[c] = decimal.Zero
		}
	}
	return result, nil
}

// UpdateBalance 更新钱包余额 (原子操作)
func (s *WalletService) UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	return s.walletRepo.UpdateBalance(ctx, walletID, amount)
}

// FreezeBalance 冻结余额
func (s *WalletService) FreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	return s.walletRepo.FreezeBalance(ctx, walletID, amount)
}

// UnfreezeBalance 解冻余额
func (s *WalletService) UnfreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	return s.walletRepo.UnfreezeBalance(ctx, walletID, amount)
}

// CheckBalance 检查余额是否充足
func (s *WalletService) CheckBalance(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) error {
	wallet, err := s.walletRepo.FindByUserIDAndCurrency(ctx, userID, currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return apperrors.NewNotFound("钱包", currency)
	}
	if !wallet.HasSufficientBalance(amount) {
		return apperrors.NewInsufficientBalance(currency)
	}
	return nil
}
