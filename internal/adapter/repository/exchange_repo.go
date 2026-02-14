package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// ExchangeRepo 闪兑记录 Repository GORM 实现
type ExchangeRepo struct {
	db *gorm.DB
}

// NewExchangeRepo 创建闪兑记录 Repository
func NewExchangeRepo(db *gorm.DB) *ExchangeRepo {
	return &ExchangeRepo{db: db}
}

// Create 创建闪兑记录
func (r *ExchangeRepo) Create(ctx context.Context, exchange *entity.Exchange) error {
	return r.db.WithContext(ctx).Create(exchange).Error
}

// FindByID 根据 ID 查询
func (r *ExchangeRepo) FindByID(ctx context.Context, id uint64) (*entity.Exchange, error) {
	var e entity.Exchange
	err := r.db.WithContext(ctx).First(&e, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &e, err
}

// ListByUserID 获取用户闪兑列表
func (r *ExchangeRepo) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Exchange, int64, error) {
	var list []*entity.Exchange
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Exchange{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
