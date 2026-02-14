package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// CNYWithdrawalRepo CNY 提现记录 Repository GORM 实现
type CNYWithdrawalRepo struct {
	db *gorm.DB
}

// NewCNYWithdrawalRepo 创建 CNY 提现记录 Repository
func NewCNYWithdrawalRepo(db *gorm.DB) *CNYWithdrawalRepo {
	return &CNYWithdrawalRepo{db: db}
}

// Create 创建 CNY 提现记录
func (r *CNYWithdrawalRepo) Create(ctx context.Context, withdrawal *entity.CNYWithdrawal) error {
	return r.db.WithContext(ctx).Create(withdrawal).Error
}

// FindByID 根据 ID 查询
func (r *CNYWithdrawalRepo) FindByID(ctx context.Context, id uint64) (*entity.CNYWithdrawal, error) {
	var w entity.CNYWithdrawal
	err := r.db.WithContext(ctx).First(&w, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

// Update 更新记录
func (r *CNYWithdrawalRepo) Update(ctx context.Context, withdrawal *entity.CNYWithdrawal) error {
	return r.db.WithContext(ctx).Save(withdrawal).Error
}

// ListByUserID 获取用户提现列表 (分页)
func (r *CNYWithdrawalRepo) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.CNYWithdrawal, int64, error) {
	var list []*entity.CNYWithdrawal
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.CNYWithdrawal{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByStatus 获取指定状态的列表
func (r *CNYWithdrawalRepo) ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.CNYWithdrawal, int64, error) {
	var list []*entity.CNYWithdrawal
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.CNYWithdrawal{}).Where("status = ?", status)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
