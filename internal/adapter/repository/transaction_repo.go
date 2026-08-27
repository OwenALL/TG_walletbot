package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"gorm.io/gorm"
)

// TransactionRepo 交易流水 Repository GORM 实现
type TransactionRepo struct {
	db *gorm.DB
}

// NewTransactionRepo 创建交易流水 Repository
func NewTransactionRepo(db *gorm.DB) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// Create 创建交易记录
func (r *TransactionRepo) Create(ctx context.Context, tx *entity.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// FindByID 根据 ID 查询
func (r *TransactionRepo) FindByID(ctx context.Context, id uint64) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.db.WithContext(ctx).First(&tx, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &tx, err
}

// ListByUserID 获取用户交易列表 (分页、筛选)
func (r *TransactionRepo) ListByUserID(ctx context.Context, userID uint64, filter *port.TransactionFilter, offset, limit int) ([]*entity.Transaction, int64, error) {
	var txs []*entity.Transaction
	var total int64

	db := r.db.WithContext(ctx).Model(&entity.Transaction{}).Where("user_id = ?", userID)

	if filter != nil {
		if filter.Currency != "" {
			db = db.Where("currency = ?", filter.Currency)
		}
		if filter.Type != "" {
			db = db.Where("type = ?", filter.Type)
		}
		if filter.HideFinance {
			db = db.Where("type NOT IN (?)", []string{
				entity.TxTypeFinanceIn, entity.TxTypeFinanceOut, entity.TxTypeFinanceProfit,
			})
		}
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&txs).Error; err != nil {
		return nil, 0, err
	}
	return txs, total, nil
}
