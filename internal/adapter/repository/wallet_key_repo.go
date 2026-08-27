package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// WalletKeyRepo 钱包私钥 Repository GORM 实现
type WalletKeyRepo struct {
	db *gorm.DB
}

// NewWalletKeyRepo 创建钱包私钥 Repository
func NewWalletKeyRepo(db *gorm.DB) *WalletKeyRepo {
	return &WalletKeyRepo{db: db}
}

// Create 创建钱包私钥记录
func (r *WalletKeyRepo) Create(ctx context.Context, key *entity.WalletKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

// FindByUserID 根据用户 ID 查询钱包私钥
func (r *WalletKeyRepo) FindByUserID(ctx context.Context, userID uint64) (*entity.WalletKey, error) {
	var key entity.WalletKey
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &key, err
}
