package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// Exchange 闪兑记录实体
type Exchange struct {
	ID           uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64          `json:"user_id" gorm:"not null;index"`
	FromCurrency string          `json:"from_currency" gorm:"size:10;not null"`
	ToCurrency   string          `json:"to_currency" gorm:"size:10;not null"`
	FromAmount   decimal.Decimal `json:"from_amount" gorm:"type:decimal(20,8);not null"`
	ToAmount     decimal.Decimal `json:"to_amount" gorm:"type:decimal(20,8);not null"`
	Rate         decimal.Decimal `json:"rate" gorm:"type:decimal(20,8);not null"`
	Fee          decimal.Decimal `json:"fee" gorm:"type:decimal(20,8);default:0"`
	Status       int8            `json:"status" gorm:"default:1"`
	CreatedAt    time.Time       `json:"created_at"`
}

// TableName 指定表名
func (Exchange) TableName() string {
	return "exchanges"
}
