package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// FinanceInvestmentRepo 余额宝投资 Repository GORM 实现
type FinanceInvestmentRepo struct {
	db *gorm.DB
}

// NewFinanceInvestmentRepo 创建余额宝投资 Repository
func NewFinanceInvestmentRepo(db *gorm.DB) *FinanceInvestmentRepo {
	return &FinanceInvestmentRepo{db: db}
}

// Create 创建投资记录
func (r *FinanceInvestmentRepo) Create(ctx context.Context, investment *entity.FinanceInvestment) error {
	return r.db.WithContext(ctx).Create(investment).Error
}

// FindByID 根据 ID 查询
func (r *FinanceInvestmentRepo) FindByID(ctx context.Context, id uint64) (*entity.FinanceInvestment, error) {
	var f entity.FinanceInvestment
	err := r.db.WithContext(ctx).First(&f, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &f, err
}

// Update 更新
func (r *FinanceInvestmentRepo) Update(ctx context.Context, investment *entity.FinanceInvestment) error {
	return r.db.WithContext(ctx).Save(investment).Error
}

// ListActiveByUserID 获取用户持有中的投资列表
func (r *FinanceInvestmentRepo) ListActiveByUserID(ctx context.Context, userID uint64) ([]*entity.FinanceInvestment, error) {
	var list []*entity.FinanceInvestment
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, entity.FinanceStatusHolding).
		Order("id DESC").Find(&list).Error
	return list, err
}

// ListAllActive 获取所有持有中的投资 (每日派息用)
func (r *FinanceInvestmentRepo) ListAllActive(ctx context.Context) ([]*entity.FinanceInvestment, error) {
	var list []*entity.FinanceInvestment
	err := r.db.WithContext(ctx).
		Where("status = ?", entity.FinanceStatusHolding).
		Find(&list).Error
	return list, err
}
