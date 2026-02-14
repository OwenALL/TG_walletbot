package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

// 红包类型枚举
const (
	RedPacketTypeEqual = 1 // 均分红包
	RedPacketTypeRandom = 2 // 拼手气红包
)

// 红包状态枚举
const (
	RedPacketStatusPending   = 0 // 待发送
	RedPacketStatusSent      = 1 // 已发送
	RedPacketStatusClaimed   = 2 // 已领完
	RedPacketStatusExpired   = 3 // 已过期
	RedPacketStatusCancelled = 4 // 已关闭(退款)
)

// RedPacket 红包实体
type RedPacket struct {
	ID            uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID        uint64          `json:"user_id" gorm:"not null;index"`
	Currency      string          `json:"currency" gorm:"size:10;not null"`
	TotalAmount   decimal.Decimal `json:"total_amount" gorm:"type:decimal(20,8);not null"`
	TotalCount    int             `json:"total_count" gorm:"not null"`
	ClaimedCount  int             `json:"claimed_count" gorm:"default:0"`
	ClaimedAmount decimal.Decimal `json:"claimed_amount" gorm:"type:decimal(20,8);default:0"`
	Type          int8            `json:"type" gorm:"default:1"`
	CoverFileID   string          `json:"cover_file_id" gorm:"size:255;default:''"`
	Remark        string          `json:"remark" gorm:"size:200;default:''"`          // 红包备注
	IsCaptcha     bool            `json:"is_captcha" gorm:"default:false"`            // 验证码红包
	IsPremiumOnly bool            `json:"is_premium_only" gorm:"default:false"`       // 会员专属红包
	IsPrivate     bool            `json:"is_private" gorm:"default:false"`            // 隐私红包(不显示领取人)
	Status        int8            `json:"status" gorm:"default:0;index"`
	MessageID     *int64          `json:"message_id" gorm:"default:null"`
	ChatID        *int64          `json:"chat_id" gorm:"default:null"`
	ExpireAt      time.Time       `json:"expire_at" gorm:"index"`
	CreatedAt     time.Time       `json:"created_at"`
}

// TableName 指定表名
func (RedPacket) TableName() string {
	return "red_packets"
}

// IsExpired 判断红包是否过期
func (r *RedPacket) IsExpired() bool {
	return time.Now().After(r.ExpireAt)
}

// IsFullyClaimed 判断是否已全部领取
func (r *RedPacket) IsFullyClaimed() bool {
	return r.ClaimedCount >= r.TotalCount
}

// RemainingAmount 剩余金额
func (r *RedPacket) RemainingAmount() decimal.Decimal {
	return r.TotalAmount.Sub(r.ClaimedAmount)
}

// RedPacketClaim 红包领取记录
type RedPacketClaim struct {
	ID          uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	RedPacketID uint64          `json:"red_packet_id" gorm:"not null;index;uniqueIndex:uk_packet_user"`
	UserID      uint64          `json:"user_id" gorm:"not null;index;uniqueIndex:uk_packet_user"`
	Amount      decimal.Decimal `json:"amount" gorm:"type:decimal(20,8);not null"`
	CreatedAt   time.Time       `json:"created_at"`
}

// TableName 指定表名
func (RedPacketClaim) TableName() string {
	return "red_packet_claims"
}

// RedPacketCover 红包封面
type RedPacketCover struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `json:"user_id" gorm:"not null;index"`
	FileID    string    `json:"file_id" gorm:"size:255;not null"`
	FileType  string    `json:"file_type" gorm:"size:20;not null"` // image / gif / video
	Status    int8      `json:"status" gorm:"default:1"`           // 1:正常 2:违规下架
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (RedPacketCover) TableName() string {
	return "red_packet_covers"
}
