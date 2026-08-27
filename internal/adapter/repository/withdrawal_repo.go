package repository

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// WithdrawalRepo 提币记录 Repository GORM 实现
type WithdrawalRepo struct {
	db *gorm.DB
}

// NewWithdrawalRepo 创建提币记录 Repository
func NewWithdrawalRepo(db *gorm.DB) *WithdrawalRepo {
	return &WithdrawalRepo{db: db}
}

// Create 创建提币记录
func (r *WithdrawalRepo) Create(ctx context.Context, withdrawal *entity.Withdrawal) error {
	return r.db.WithContext(ctx).Create(withdrawal).Error
}

// FindByID 根据 ID 查询
func (r *WithdrawalRepo) FindByID(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
	var w entity.Withdrawal
	err := r.db.WithContext(ctx).First(&w, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &w, err
}

// Update 更新提币记录
func (r *WithdrawalRepo) Update(ctx context.Context, withdrawal *entity.Withdrawal) error {
	return r.db.WithContext(ctx).Save(withdrawal).Error
}

// ListByUserID 获取用户提币列表 (分页)
func (r *WithdrawalRepo) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Withdrawal, int64, error) {
	var list []*entity.Withdrawal
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Withdrawal{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByStatus 获取指定状态的提币列表
func (r *WithdrawalRepo) ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Withdrawal, int64, error) {
	var list []*entity.Withdrawal
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Withdrawal{}).Where("status = ?", status)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// SumTodayByUserID 统计用户今日提币总额
func (r *WithdrawalRepo) SumTodayByUserID(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}
	today := time.Now().Format("2006-01-02")
	err := r.db.WithContext(ctx).Model(&entity.Withdrawal{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("user_id = ? AND currency = ? AND DATE(created_at) = ? AND status NOT IN (?)", userID, currency, today, []int8{
			entity.WithdrawalStatusRejected, entity.WithdrawalStatusFailed,
		}).
		Scan(&result).Error
	return result.Total, err
}
