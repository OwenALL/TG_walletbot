package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// TransferRepo 转账记录 Repository GORM 实现
type TransferRepo struct {
	db *gorm.DB
}

// NewTransferRepo 创建转账记录 Repository
func NewTransferRepo(db *gorm.DB) *TransferRepo {
	return &TransferRepo{db: db}
}

// Create 创建转账记录
func (r *TransferRepo) Create(ctx context.Context, transfer *entity.Transfer) error {
	return r.db.WithContext(ctx).Create(transfer).Error
}

// FindByID 根据 ID 查询
func (r *TransferRepo) FindByID(ctx context.Context, id uint64) (*entity.Transfer, error) {
	var t entity.Transfer
	err := r.db.WithContext(ctx).First(&t, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

// Update 更新转账记录
func (r *TransferRepo) Update(ctx context.Context, transfer *entity.Transfer) error {
	return r.db.WithContext(ctx).Save(transfer).Error
}

// ListByUserID 获取用户相关的转账列表
func (r *TransferRepo) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Transfer, int64, error) {
	var list []*entity.Transfer
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Transfer{}).
		Where("from_user_id = ? OR to_user_id = ?", userID, userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
