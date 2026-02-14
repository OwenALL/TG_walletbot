package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// UserSettings 用户配置实体
type UserSettings struct {
	ID                   uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID               uint64          `json:"user_id" gorm:"not null;uniqueIndex"`
	Language             string          `json:"language" gorm:"size:10;default:'zh-CN'"`
	SmallFreeUSDT        decimal.Decimal `json:"small_free_usdt" gorm:"type:decimal(20,8);default:100"`
	SmallFreeTRX         decimal.Decimal `json:"small_free_trx" gorm:"type:decimal(20,8);default:100"`
	DailySpentUSDT       decimal.Decimal `json:"daily_spent_usdt" gorm:"type:decimal(20,8);default:0"`
	DailySpentTRX        decimal.Decimal `json:"daily_spent_trx" gorm:"type:decimal(20,8);default:0"`
	DailySpentDate       *time.Time      `json:"daily_spent_date" gorm:"type:date;default:null"`
	NotificationEnabled  bool            `json:"notification_enabled" gorm:"default:true"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (UserSettings) TableName() string {
	return "user_settings"
}

// NeedPINForAmount 判断是否需要 PIN 验证 (小额免密检查)
func (s *UserSettings) NeedPINForAmount(currency string, amount decimal.Decimal) bool {
	today := time.Now().Format("2006-01-02")
	var dailySpent, limit decimal.Decimal

	switch currency {
	case "USDT":
		limit = s.SmallFreeUSDT
		if s.DailySpentDate != nil && s.DailySpentDate.Format("2006-01-02") == today {
			dailySpent = s.DailySpentUSDT
		}
	case "TRX":
		limit = s.SmallFreeTRX
		if s.DailySpentDate != nil && s.DailySpentDate.Format("2006-01-02") == today {
			dailySpent = s.DailySpentTRX
		}
	default:
		// CNY 等其他币种默认需要 PIN
		return true
	}

	// 如果额度为 0 表示关闭免密
	if limit.IsZero() {
		return true
	}

	// 今日累计消费 + 本次金额是否超过额度
	return dailySpent.Add(amount).GreaterThan(limit)
}
