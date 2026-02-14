package entity

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// 商户状态枚举
const (
	MerchantStatusActive   = 1 // 正常
	MerchantStatusDisabled = 0 // 已关闭

	// 以下状态保留用于后台管理兼容
	MerchantStatusPending  = 2 // 待审核 (旧流程)
	MerchantStatusRejected = 3 // 已拒绝 (旧流程)

	// MerchantStatusApproved 兼容别名，等同于 MerchantStatusActive
	MerchantStatusApproved = MerchantStatusActive
)

// Merchant 商户实体
type Merchant struct {
	ID           uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64          `json:"user_id" gorm:"not null;index"`
	BusinessName string          `json:"business_name" gorm:"size:128;not null;default:''"`
	APIKey       string          `json:"api_key" gorm:"size:64;not null;default:'';uniqueIndex"`
	APISecret    string          `json:"api_secret" gorm:"size:128;not null;default:''"` // 商户可在详情页看到密钥
	WebhookURL   string          `json:"webhook_url" gorm:"size:255;not null;default:''"`
	Logo         string          `json:"logo" gorm:"size:255;not null;default:''"`
	FeeRate      decimal.Decimal `json:"fee_rate" gorm:"type:decimal(5,2);default:0.00"`
	IPWhitelist  string          `json:"ip_whitelist" gorm:"type:text"`
	Status       int8            `json:"status" gorm:"default:1;index"`
	Description  string          `json:"description" gorm:"size:512;default:''"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (Merchant) TableName() string { return "merchants" }

// IsActive 判断商户是否为正常状态
func (m *Merchant) IsActive() bool { return m.Status == MerchantStatusActive }

// IsDisabled 判断商户是否被关闭
func (m *Merchant) IsDisabled() bool { return m.Status == MerchantStatusDisabled }

// IsApproved 兼容旧逻辑，Active 等同于 Approved
func (m *Merchant) IsApproved() bool { return m.Status == MerchantStatusActive }

// IsPending 判断商户是否处于待审核状态 (旧流程兼容)
func (m *Merchant) IsPending() bool { return m.Status == MerchantStatusPending }

// IsRejected 判断商户是否被拒绝 (旧流程兼容)
func (m *Merchant) IsRejected() bool { return m.Status == MerchantStatusRejected }

// StatusText 获取状态文本
func (m *Merchant) StatusText() string {
	switch m.Status {
	case MerchantStatusActive:
		return "正常"
	case MerchantStatusDisabled:
		return "已关闭"
	case MerchantStatusPending:
		return "待审核"
	case MerchantStatusRejected:
		return "已拒绝"
	default:
		return "未知"
	}
}

// StatusEmoji 获取状态 emoji 文本
func (m *Merchant) StatusEmoji() string {
	if m.IsActive() {
		return "✅ 正常"
	}
	return "❌ 已关闭"
}

// GetIPList 获取 IP 白名单列表
func (m *Merchant) GetIPList() []string {
	if m.IPWhitelist == "" {
		return nil
	}
	var ips []string
	if err := json.Unmarshal([]byte(m.IPWhitelist), &ips); err != nil {
		return nil
	}
	return ips
}

// SetIPList 设置 IP 白名单列表
func (m *Merchant) SetIPList(ips []string) {
	if len(ips) == 0 {
		m.IPWhitelist = ""
		return
	}
	data, err := json.Marshal(ips)
	if err != nil {
		m.IPWhitelist = ""
		return
	}
	m.IPWhitelist = string(data)
}

// AddIP 添加 IP 到白名单，返回是否新增成功
func (m *Merchant) AddIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	ips := m.GetIPList()
	for _, existing := range ips {
		if existing == ip {
			return false // 已存在
		}
	}
	ips = append(ips, ip)
	m.SetIPList(ips)
	return true
}

// RemoveIP 从白名单移除 IP，返回是否移除成功
func (m *Merchant) RemoveIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	ips := m.GetIPList()
	found := false
	var result []string
	for _, existing := range ips {
		if existing == ip {
			found = true
			continue
		}
		result = append(result, existing)
	}
	if found {
		m.SetIPList(result)
	}
	return found
}

// WebhookDisplay 获取 webhook 显示文本
func (m *Merchant) WebhookDisplay() string {
	if m.WebhookURL == "" {
		return "未设置"
	}
	return m.WebhookURL
}

// FeeRateDisplay 获取费率显示文本
func (m *Merchant) FeeRateDisplay() string {
	return m.FeeRate.StringFixed(0) + "%"
}
