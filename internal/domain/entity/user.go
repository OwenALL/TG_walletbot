// Package entity 定义领域层业务实体，不依赖任何外部框架
package entity

import "time"

// 用户状态枚举
const (
	UserStatusActive = 1 // 正常
	UserStatusFrozen = 2 // 冻结
	UserStatusBanned = 3 // 封禁
)

// User 用户实体
type User struct {
	ID             uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramID     int64      `json:"telegram_id" gorm:"uniqueIndex;not null"`
	Username       string     `json:"username" gorm:"size:64;default:''"`
	FirstName      string     `json:"first_name" gorm:"size:128;default:''"`
	LastName       string     `json:"last_name" gorm:"size:128;default:''"`
	LanguageCode   string     `json:"language_code" gorm:"size:10;default:'zh-CN'"`
	IsPremium      bool       `json:"is_premium" gorm:"default:false"`
	PINHash        string     `json:"-" gorm:"size:255;default:''"`
	TOTPSecret     string     `json:"-" gorm:"size:64;default:''"`
	TOTPEnabled    bool       `json:"totp_enabled" gorm:"default:false"`
	HasPINSet      bool       `json:"has_pin" gorm:"-"` // 非数据库字段，由应用层计算
	Status         int8       `json:"status" gorm:"default:1"`
	PINFailCount   int        `json:"pin_fail_count" gorm:"default:0"`
	PINLockedUntil *time.Time `json:"pin_locked_until" gorm:"default:null"`
	ReferrerID     *uint64    `json:"referrer_id" gorm:"default:null"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// FullName 获取完整显示名称
func (u *User) FullName() string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	return name
}

// DisplayName 获取显示名称 (优先 FullName, 其次 Username)
func (u *User) DisplayName() string {
	name := u.FullName()
	if name == "" && u.Username != "" {
		name = "@" + u.Username
	}
	if name == "" {
		name = "未知用户"
	}
	return name
}

// IsActive 判断用户是否为正常状态
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsFrozen 判断用户是否被冻结
func (u *User) IsFrozen() bool {
	return u.Status == UserStatusFrozen
}

// HasPIN 判断是否已设置支付密码
func (u *User) HasPIN() bool {
	return u.PINHash != ""
}

// IsPINLocked 判断 PIN 是否被锁定
func (u *User) IsPINLocked() bool {
	if u.PINLockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.PINLockedUntil)
}
