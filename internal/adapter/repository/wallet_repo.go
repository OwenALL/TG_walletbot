package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// WalletRepo 钱包 Repository GORM 实现
type WalletRepo struct {
	db *gorm.DB
}

// NewWalletRepo 创建钱包 Repository
func NewWalletRepo(db *gorm.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

// Create 创建钱包
func (r *WalletRepo) Create(ctx context.Context, wallet *entity.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

// FindByID 根据 ID 查询钱包
func (r *WalletRepo) FindByID(ctx context.Context, id uint64) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).First(&wallet, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &wallet, err
}

// FindByUserIDAndCurrency 根据用户 ID 和币种查询钱包
func (r *WalletRepo) FindByUserIDAndCurrency(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).Where("user_id = ? AND currency = ?", userID, currency).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &wallet, err
}

// FindByUserID 获取用户所有钱包
func (r *WalletRepo) FindByUserID(ctx context.Context, userID uint64) ([]*entity.Wallet, error) {
	var wallets []*entity.Wallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&wallets).Error
	return wallets, err
}

// FindByDepositAddress 根据充值地址查询钱包
func (r *WalletRepo) FindByDepositAddress(ctx context.Context, address string) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).Where("deposit_address = ? AND deposit_address != ''", address).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &wallet, err
}

// UpdateBalance 更新余额 (原子操作，使用 SQL 表达式)
func (r *WalletRepo) UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("钱包不存在: %d", walletID)
	}
	return nil
}

// FreezeBalance 冻结余额
func (r *WalletRepo) FreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&entity.Wallet{}).
		Where("id = ? AND balance >= frozen_balance + ?", walletID, amount).
		Updates(map[string]interface{}{
			"frozen_balance": gorm.Expr("frozen_balance + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("冻结余额失败，可用余额不足")
	}
	return nil
}

// UnfreezeBalance 解冻余额
func (r *WalletRepo) UnfreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	result := r.db.WithContext(ctx).Model(&entity.Wallet{}).
		Where("id = ? AND frozen_balance >= ?", walletID, amount).
		Updates(map[string]interface{}{
			"frozen_balance": gorm.Expr("frozen_balance - ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("解冻余额失败")
	}
	return nil
}

// UpdateDepositAddress 更新用户所有链上钱包的充值地址 (USDT/TRX 共用)
func (r *WalletRepo) UpdateDepositAddress(ctx context.Context, userID uint64, address string) error {
	result := r.db.WithContext(ctx).Model(&entity.Wallet{}).
		Where("user_id = ? AND currency IN ?", userID, []string{entity.CurrencyUSDT, entity.CurrencyTRX}).
		Update("deposit_address", address)
	return result.Error
}

// ListAllWithDepositAddress 获取所有已分配充值地址的钱包
// 返回去重后的钱包列表 (同一用户 USDT/TRX 共用地址，只取一条)
func (r *WalletRepo) ListAllWithDepositAddress(ctx context.Context) ([]*entity.Wallet, error) {
	var wallets []*entity.Wallet
	err := r.db.WithContext(ctx).
		Where("deposit_address != '' AND deposit_address IS NOT NULL").
		Group("deposit_address").
		Find(&wallets).Error
	return wallets, err
}

// ListDepositAddressesCreatedAfter 获取指定时间之后创建的充值地址 (增量刷新用)
// 返回去重后的钱包列表
func (r *WalletRepo) ListDepositAddressesCreatedAfter(ctx context.Context, after time.Time) ([]*entity.Wallet, error) {
	var wallets []*entity.Wallet
	err := r.db.WithContext(ctx).
		Where("deposit_address != '' AND deposit_address IS NOT NULL AND created_at > ?", after).
		Group("deposit_address").
		Find(&wallets).Error
	return wallets, err
}
