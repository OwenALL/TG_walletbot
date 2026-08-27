package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// SubAccountRepo 备用账号 Repository GORM 实现
type SubAccountRepo struct {
	db *gorm.DB
}

// NewSubAccountRepo 创建备用账号 Repository
func NewSubAccountRepo(db *gorm.DB) *SubAccountRepo {
	return &SubAccountRepo{db: db}
}

// Create 创建备用账号关联
func (r *SubAccountRepo) Create(ctx context.Context, sub *entity.SubAccount) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

// FindByID 根据 ID 查询
func (r *SubAccountRepo) FindByID(ctx context.Context, id uint64) (*entity.SubAccount, error) {
	var s entity.SubAccount
	err := r.db.WithContext(ctx).First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

// Update 更新
func (r *SubAccountRepo) Update(ctx context.Context, sub *entity.SubAccount) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

// ListByMasterUserID 获取主账号的备用账号列表
func (r *SubAccountRepo) ListByMasterUserID(ctx context.Context, masterUserID uint64) ([]*entity.SubAccount, error) {
	var list []*entity.SubAccount
	err := r.db.WithContext(ctx).
		Where("master_user_id = ? AND status = ?", masterUserID, entity.SubAccountStatusConfirmed).
		Find(&list).Error
	return list, err
}

// ListBySubUserID 获取备用账号关联的主账号列表
func (r *SubAccountRepo) ListBySubUserID(ctx context.Context, subUserID uint64) ([]*entity.SubAccount, error) {
	var list []*entity.SubAccount
	err := r.db.WithContext(ctx).
		Where("sub_user_id = ? AND status = ?", subUserID, entity.SubAccountStatusConfirmed).
		Find(&list).Error
	return list, err
}
