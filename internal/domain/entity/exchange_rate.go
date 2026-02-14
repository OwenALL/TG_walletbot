package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// ExchangeRate 汇率配置实体
type ExchangeRate struct {
	ID           uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	FromCurrency string          `json:"from_currency" gorm:"size:10;not null;uniqueIndex:uk_pair"`
	ToCurrency   string          `json:"to_currency" gorm:"size:10;not null;uniqueIndex:uk_pair"`
	Rate         decimal.Decimal `json:"rate" gorm:"type:decimal(20,8);not null"`
	Spread       decimal.Decimal `json:"spread" gorm:"type:decimal(10,4);default:0"` // 加点比例 (%)
	MinAmount    decimal.Decimal `json:"min_amount" gorm:"type:decimal(20,8);default:0"`
	MaxAmount    decimal.Decimal `json:"max_amount" gorm:"type:decimal(20,8);default:0"`
	Enabled      bool            `json:"enabled" gorm:"default:true"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (ExchangeRate) TableName() string {
	return "exchange_rates"
}

// EffectiveRate 计算含加点后的有效汇率
func (e *ExchangeRate) EffectiveRate() decimal.Decimal {
	if e.Spread.IsZero() {
		return e.Rate
	}
	// 有效汇率 = 基准汇率 * (1 - 加点/100)
	spreadFactor := decimal.NewFromInt(1).Sub(e.Spread.Div(decimal.NewFromInt(100)))
	return e.Rate.Mul(spreadFactor)
}
