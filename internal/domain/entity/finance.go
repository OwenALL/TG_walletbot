package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 余额宝投资状态枚举
const (
	FinanceStatusHolding   = 1 // 持有中
	FinanceStatusWithdrawn = 2 // 已取出
)

// FinanceInvestment 余额宝投资记录
type FinanceInvestment struct {
	ID            uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint64          `json:"user_id" gorm:"not null;index"`
	Amount        decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	AnnualRate    decimal.Decimal `json:"annual_rate" gorm:"type:decimal(10,4);not null"`
	Status        int8            `json:"status" gorm:"default:1;index"`
	InterestStart time.Time       `json:"interest_start" gorm:"not null"`
	TotalProfit   decimal.Decimal `json:"total_profit" gorm:"type:decimal(20,8);default:0"`
	CreatedAt     time.Time       `json:"created_at"`
	WithdrawnAt   *time.Time      `json:"withdrawn_at" gorm:"default:null"`
}

// TableName 指定表名
func (FinanceInvestment) TableName() string {
	return "finance_investments"
}

// DailyProfit 计算每日收益: 持仓金额 * 年化 / 365
func (f *FinanceInvestment) DailyProfit() decimal.Decimal {
	rate := f.AnnualRate.Div(decimal.NewFromInt(100))
	return f.Amount.Mul(rate).Div(decimal.NewFromInt(365))
}

// IsHolding 判断是否持有中
func (f *FinanceInvestment) IsHolding() bool {
	return f.Status == FinanceStatusHolding
}
