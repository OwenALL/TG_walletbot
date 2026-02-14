package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 货币类型枚举
const (
	CurrencyUSDT = "USDT"
	CurrencyTRX  = "TRX"
	CurrencyCNY  = "CNY"
)

// Wallet 钱包实体
type Wallet struct {
	ID             uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         uint64          `json:"user_id" gorm:"not null;index"`
	Currency       string          `json:"currency" gorm:"size:10;not null"`
	Balance        decimal.Decimal `json:"balance" gorm:"type:decimal(20,8);not null;default:0"`
	FrozenBalance  decimal.Decimal `json:"frozen_balance" gorm:"type:decimal(20,8);not null;default:0"`
	DepositAddress string          `json:"deposit_address" gorm:"size:128;default:''"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (Wallet) TableName() string {
	return "wallets"
}

// AvailableBalance 可用余额 (可用 = 总余额 - 冻结)
func (w *Wallet) AvailableBalance() decimal.Decimal {
	return w.Balance.Sub(w.FrozenBalance)
}

// HasSufficientBalance 判断可用余额是否充足
func (w *Wallet) HasSufficientBalance(amount decimal.Decimal) bool {
	return w.AvailableBalance().GreaterThanOrEqual(amount)
}

// validCurrencies 支持的币种集合
var validCurrencies = map[string]bool{
	CurrencyUSDT: true,
	CurrencyTRX:  true,
	CurrencyCNY:  true,
}

// IsValidCurrency 校验币种是否合法 (USDT/TRX/CNY)
func IsValidCurrency(currency string) bool {
	return validCurrencies[currency]
}
