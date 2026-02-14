package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// MerchantRepo 商户 Repository GORM 实现
type MerchantRepo struct {
	db *gorm.DB
}

// NewMerchantRepo 创建商户 Repository
func NewMerchantRepo(db *gorm.DB) *MerchantRepo {
	return &MerchantRepo{db: db}
}

// Create 创建商户
func (r *MerchantRepo) Create(ctx context.Context, merchant *entity.Merchant) error {
	return r.db.WithContext(ctx).Create(merchant).Error
}

// FindByID 根据 ID 查询商户
func (r *MerchantRepo) FindByID(ctx context.Context, id uint64) (*entity.Merchant, error) {
	var m entity.Merchant
	err := r.db.WithContext(ctx).First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

// ListByUserID 获取用户的所有商户列表
func (r *MerchantRepo) ListByUserID(ctx context.Context, userID uint64) ([]*entity.Merchant, error) {
	var list []*entity.Merchant
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("id ASC").Find(&list).Error
	return list, err
}

// CountByUserID 统计用户商户数量
func (r *MerchantRepo) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Merchant{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// FindByAPIKey 根据 API Key 查询商户
func (r *MerchantRepo) FindByAPIKey(ctx context.Context, apiKey string) (*entity.Merchant, error) {
	var m entity.Merchant
	err := r.db.WithContext(ctx).Where("api_key = ?", apiKey).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &m, err
}

// Update 更新商户
func (r *MerchantRepo) Update(ctx context.Context, merchant *entity.Merchant) error {
	return r.db.WithContext(ctx).Save(merchant).Error
}

// ListByStatus 获取指定状态的商户列表 (分页)
func (r *MerchantRepo) ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Merchant, int64, error) {
	var list []*entity.Merchant
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Merchant{}).Where("status = ?", status)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
