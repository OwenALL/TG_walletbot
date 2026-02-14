package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 转账类型枚举
const (
	TransferTypeDirect  = 1 // 主动转账
	TransferTypeRequest = 2 // 收款请求
)

// 转账状态枚举
const (
	TransferStatusPending   = 0 // 待确认 (收款请求)
	TransferStatusCompleted = 1 // 已完成
	TransferStatusRejected  = 2 // 已拒绝
	TransferStatusCancelled = 3 // 已取消
)

// Transfer 转账记录实体
type Transfer struct {
	ID          uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	FromUserID  uint64          `json:"from_user_id" gorm:"not null;index"`
	ToUserID    uint64          `json:"to_user_id" gorm:"not null;index"`
	Currency    string          `json:"currency" gorm:"size:10;not null"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	Type        int8            `json:"type" gorm:"default:1"`
	Status      int8            `json:"status" gorm:"default:0"`
	Memo        string          `json:"memo" gorm:"size:255;default:''"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at" gorm:"default:null"`
}

// TableName 指定表名
func (Transfer) TableName() string {
	return "transfers"
}

// IsCompleted 判断是否已完成
func (t *Transfer) IsCompleted() bool {
	return t.Status == TransferStatusCompleted
}
