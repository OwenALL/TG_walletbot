package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 提币状态枚举
const (
	WithdrawalStatusPending    = 0 // 待审核
	WithdrawalStatusProcessing = 1 // 处理中
	WithdrawalStatusCompleted  = 2 // 已完成
	WithdrawalStatusRejected   = 3 // 已拒绝
	WithdrawalStatusFailed     = 4 // 失败
)

// Withdrawal 提币记录实体
type Withdrawal struct {
	ID           uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64          `json:"user_id" gorm:"not null;index"`
	Currency     string          `json:"currency" gorm:"size:10;not null"`
	Amount       decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	Fee          decimal.Decimal `json:"fee" gorm:"type:decimal(20,8);not null"`
	ActualAmount decimal.Decimal `json:"actual_amount" gorm:"type:decimal(20,8);not null"`
	ToAddress    string          `json:"to_address" gorm:"size:128;not null"`
	TxHash       string          `json:"tx_hash" gorm:"size:128;default:''"`
	Status       int8            `json:"status" gorm:"default:0;index"`
	ReviewerID   *uint64         `json:"reviewer_id" gorm:"default:null"`
	ReviewNote   string          `json:"review_note" gorm:"size:255;default:''"`
	CreatedAt    time.Time       `json:"created_at"`
	ReviewedAt   *time.Time      `json:"reviewed_at" gorm:"default:null"`
	CompletedAt  *time.Time      `json:"completed_at" gorm:"default:null"`
}

// TableName 指定表名
func (Withdrawal) TableName() string {
	return "withdrawals"
}

// IsPending 判断是否待审核
func (w *Withdrawal) IsPending() bool {
	return w.Status == WithdrawalStatusPending
}

// CNYWithdrawal CNY 提现记录实体
type CNYWithdrawal struct {
	ID          uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64          `json:"user_id" gorm:"not null;index"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(20,2);not null"`
	Fee         decimal.Decimal `json:"fee" gorm:"type:decimal(20,2);default:0"`
	Method      string          `json:"method" gorm:"size:20;not null"` // alipay / bank_card
	AccountName string          `json:"account_name" gorm:"size:64;not null"`
	AccountNo   string          `json:"account_no" gorm:"size:128;not null"`
	BankName    string          `json:"bank_name" gorm:"size:64;default:''"`
	Status      int8            `json:"status" gorm:"default:0;index"`
	ReviewerID  *uint64         `json:"reviewer_id" gorm:"default:null"`
	ReviewNote  string          `json:"review_note" gorm:"size:255;default:''"`
	CreatedAt   time.Time       `json:"created_at"`
	ReviewedAt  *time.Time      `json:"reviewed_at" gorm:"default:null"`
	CompletedAt *time.Time      `json:"completed_at" gorm:"default:null"`
}

// TableName 指定表名
func (CNYWithdrawal) TableName() string {
	return "cny_withdrawals"
}
