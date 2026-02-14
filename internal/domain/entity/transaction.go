package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 交易类型枚举
const (
	TxTypeDeposit         = "deposit"          // 充值
	TxTypeWithdraw        = "withdraw"         // 提币
	TxTypeTransferIn      = "transfer_in"      // 收到转账
	TxTypeTransferOut     = "transfer_out"     // 发起转账
	TxTypeExchangeIn      = "exchange_in"      // 闪兑兑入
	TxTypeExchangeOut     = "exchange_out"     // 闪兑兑出
	TxTypeRedpacketSend   = "redpacket_send"   // 发红包
	TxTypeRedpacketRecv   = "redpacket_receive" // 领红包
	TxTypeRedpacketRefund = "redpacket_refund" // 红包退回
	TxTypeFinanceIn       = "finance_in"       // 余额宝买入
	TxTypeFinanceOut      = "finance_out"      // 余额宝取出
	TxTypeFinanceProfit   = "finance_profit"   // 余额宝收益
)

// Transaction 交易流水实体
type Transaction struct {
	ID            uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint64          `json:"user_id" gorm:"not null;index"`
	Type          string          `json:"type" gorm:"size:20;not null;index"`
	Currency      string          `json:"currency" gorm:"size:10;not null"`
	Amount        decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	Fee           decimal.Decimal `json:"fee" gorm:"type:decimal(20,8);default:0"`
	BalanceBefore decimal.Decimal `json:"balance_before" gorm:"type:decimal(20,8);not null"`
	BalanceAfter  decimal.Decimal `json:"balance_after" gorm:"type:decimal(20,8);not null"`
	RelatedID     *uint64         `json:"related_id" gorm:"default:null"`
	RelatedType   string          `json:"related_type" gorm:"size:20;default:''"`
	Memo          string          `json:"memo" gorm:"size:255;default:''"`
	CreatedAt     time.Time       `json:"created_at" gorm:"index"`
}

// TableName 指定表名
func (Transaction) TableName() string {
	return "transactions"
}

// IsIncome 判断是否为收入类型交易
func (t *Transaction) IsIncome() bool {
	switch t.Type {
	case TxTypeDeposit, TxTypeTransferIn, TxTypeExchangeIn,
		TxTypeRedpacketRecv, TxTypeRedpacketRefund,
		TxTypeFinanceOut, TxTypeFinanceProfit:
		return true
	default:
		return false
	}
}
