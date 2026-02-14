package entity

import "time"

// 备用账号状态枚举
const (
	SubAccountStatusPending   = 0 // 待确认
	SubAccountStatusConfirmed = 1 // 已确认
	SubAccountStatusRemoved   = 2 // 已解除
)

// SubAccount 备用账号关联实体
type SubAccount struct {
	ID           uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	MasterUserID uint64     `json:"master_user_id" gorm:"not null;index;uniqueIndex:uk_master_sub"`
	SubUserID    uint64     `json:"sub_user_id" gorm:"not null;index;uniqueIndex:uk_master_sub"`
	Status       int8       `json:"status" gorm:"default:0"`
	CreatedAt    time.Time  `json:"created_at"`
	ConfirmedAt  *time.Time `json:"confirmed_at" gorm:"default:null"`
}

// TableName 指定表名
func (SubAccount) TableName() string {
	return "sub_accounts"
}
