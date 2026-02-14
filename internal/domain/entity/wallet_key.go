package entity

import "time"

// WalletKey 钱包私钥实体
// 存储用户 TRON 地址对应的加密私钥
type WalletKey struct {
	ID           uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64    `json:"user_id" gorm:"uniqueIndex;not null"`
	Address      string    `json:"address" gorm:"type:varchar(64);not null"`
	EncryptedKey string    `json:"-" gorm:"type:text;not null"` // json 序列化时隐藏
	KeyVersion   int       `json:"key_version" gorm:"default:1"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (WalletKey) TableName() string {
	return "wallet_keys"
}
