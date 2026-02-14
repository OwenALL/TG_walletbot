package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 充值状态枚举
const (
	DepositStatusPending   = 0 // 待确认
	DepositStatusConfirmed = 1 // 已确认
	DepositStatusCredited  = 2 // 已到账
)

// Deposit 充值记录实体
type Deposit struct {
	ID            uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint64          `json:"user_id" gorm:"not null;index"`
	Currency      string          `json:"currency" gorm:"size:10;not null"`
	Amount        decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	FromAddress   string          `json:"from_address" gorm:"size:128;not null"`
	ToAddress     string          `json:"to_address" gorm:"size:128;not null"`
	TxHash        string          `json:"tx_hash" gorm:"size:128;not null;uniqueIndex"`
	Confirmations int             `json:"confirmations" gorm:"default:0"`
	Status        int8            `json:"status" gorm:"default:0;index"`
	Notified      bool            `json:"notified" gorm:"default:false"`
	CreatedAt     time.Time       `json:"created_at"`
	ConfirmedAt   *time.Time      `json:"confirmed_at" gorm:"default:null"`
}

// TableName 指定表名
func (Deposit) TableName() string {
	return "deposits"
}

// IsCredited 判断是否已到账
func (d *Deposit) IsCredited() bool {
	return d.Status == DepositStatusCredited
}
