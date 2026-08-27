package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExchangeRateRepo 汇率配置 Repository GORM 实现
type ExchangeRateRepo struct {
	db *gorm.DB
}

// NewExchangeRateRepo 创建汇率配置 Repository
func NewExchangeRateRepo(db *gorm.DB) *ExchangeRateRepo {
	return &ExchangeRateRepo{db: db}
}

// FindByPair 根据交易对查询汇率
func (r *ExchangeRateRepo) FindByPair(ctx context.Context, fromCurrency, toCurrency string) (*entity.ExchangeRate, error) {
	var rate entity.ExchangeRate
	err := r.db.WithContext(ctx).
		Where("from_currency = ? AND to_currency = ? AND enabled = ?", fromCurrency, toCurrency, true).
		First(&rate).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rate, err
}

// FindAll 获取所有汇率配置
func (r *ExchangeRateRepo) FindAll(ctx context.Context) ([]*entity.ExchangeRate, error) {
	var rates []*entity.ExchangeRate
	err := r.db.WithContext(ctx).Find(&rates).Error
	return rates, err
}

// Upsert 创建或更新汇率
func (r *ExchangeRateRepo) Upsert(ctx context.Context, rate *entity.ExchangeRate) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "from_currency"}, {Name: "to_currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"rate", "spread", "min_amount", "max_amount", "enabled", "updated_at"}),
	}).Create(rate).Error
}
