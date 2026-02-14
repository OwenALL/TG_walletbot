package entity

import "time"

// 管理员角色枚举
const (
	AdminRoleSuperAdmin = "super_admin"
	AdminRoleAdmin      = "admin"
	AdminRoleViewer     = "viewer"
)

// AdminUser 管理员实体
type AdminUser struct {
	ID           uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TelegramID   *int64     `json:"telegram_id" gorm:"default:null"`
	Username     string     `json:"username" gorm:"size:64;not null;uniqueIndex"`
	PasswordHash string     `json:"-" gorm:"size:255;not null"`
	Role         string     `json:"role" gorm:"size:20;default:'admin'"`
	Status       int8       `json:"status" gorm:"default:1"`
	LastLoginAt  *time.Time `json:"last_login_at" gorm:"default:null"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (AdminUser) TableName() string {
	return "admin_users"
}

// IsSuperAdmin 判断是否为超级管理员
func (a *AdminUser) IsSuperAdmin() bool {
	return a.Role == AdminRoleSuperAdmin
}

// IsViewer 判断是否为只读角色
func (a *AdminUser) IsViewer() bool {
	return a.Role == AdminRoleViewer
}

// AdminLog 管理员操作日志
type AdminLog struct {
	ID            uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	AdminID       uint64    `json:"admin_id" gorm:"not null;index"`
	AdminUsername string    `json:"admin_username" gorm:"-"` // 非数据库字段，LEFT JOIN 填充
	Action        string    `json:"action" gorm:"size:64;not null;index"`
	TargetType    string    `json:"target_type" gorm:"size:32;default:''"`
	TargetID      *uint64   `json:"target_id" gorm:"default:null"`
	Detail        string    `json:"detail" gorm:"type:json;default:null"`
	IPAddress     string    `json:"ip_address" gorm:"size:45;default:''"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
}

// TableName 指定表名
func (AdminLog) TableName() string {
	return "admin_logs"
}
