package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// DepositRepo 充值记录 Repository GORM 实现
type DepositRepo struct {
	db *gorm.DB
}

// NewDepositRepo 创建充值记录 Repository
func NewDepositRepo(db *gorm.DB) *DepositRepo {
	return &DepositRepo{db: db}
}

// Create 创建充值记录
func (r *DepositRepo) Create(ctx context.Context, deposit *entity.Deposit) error {
	return r.db.WithContext(ctx).Create(deposit).Error
}

// FindByTxHash 根据交易哈希查询
func (r *DepositRepo) FindByTxHash(ctx context.Context, txHash string) (*entity.Deposit, error) {
	var deposit entity.Deposit
	err := r.db.WithContext(ctx).Where("tx_hash = ?", txHash).First(&deposit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &deposit, err
}

// UpdateStatus 更新充值状态
func (r *DepositRepo) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	return r.db.WithContext(ctx).Model(&entity.Deposit{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateConfirmations 更新确认数
func (r *DepositRepo) UpdateConfirmations(ctx context.Context, id uint64, confirmations int) error {
	return r.db.WithContext(ctx).Model(&entity.Deposit{}).Where("id = ?", id).Update("confirmations", confirmations).Error
}

// ListPending 获取待确认的充值列表
func (r *DepositRepo) ListPending(ctx context.Context) ([]*entity.Deposit, error) {
	var deposits []*entity.Deposit
	err := r.db.WithContext(ctx).Where("status IN (?)", []int8{
		entity.DepositStatusPending, entity.DepositStatusConfirmed,
	}).Find(&deposits).Error
	return deposits, err
}

// ListByUserID 获取用户充值列表 (分页)
func (r *DepositRepo) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Deposit, int64, error) {
	var deposits []*entity.Deposit
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.Deposit{}).Where("user_id = ?", userID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&deposits).Error; err != nil {
		return nil, 0, err
	}
	return deposits, total, nil
}
