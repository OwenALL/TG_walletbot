package tron

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	// scrypt KDF 参数 (OWASP 推荐)
	scryptN      = 32768 // CPU/内存开销因子
	scryptR      = 8     // 块大小
	scryptP      = 1     // 并行度
	scryptKeyLen = 32    // 派生密钥长度 (AES-256)
	scryptSaltLen = 16   // 盐长度

	// scrypt 加密格式前缀，用于区分新旧格式
	scryptPrefix = "scrypt:"
)

// CryptoService AES-256-GCM 加密/解密服务
// 用于加密保存在数据库中的私钥
type CryptoService struct{}

// NewCryptoService 创建加密服务
func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

// EncryptPrivateKey 使用 scrypt KDF + AES-256-GCM 加密私钥
// privateKeyHex: 原始私钥 hex 字符串
// encryptionKey: 加密密钥 (任意长度，通过 scrypt 派生为 32 字节)
// 返回: "scrypt:" + base64(salt + nonce + ciphertext)
func (s *CryptoService) EncryptPrivateKey(privateKeyHex string, encryptionKey string) (string, error) {
	// 生成随机 salt
	salt := make([]byte, scryptSaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("生成 salt 失败: %w", err)
	}

	// 使用 scrypt 派生加密密钥
	key, err := scrypt.Key([]byte(encryptionKey), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return "", fmt.Errorf("scrypt 密钥派生失败: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}

	// 加密
	plaintext := []byte(privateKeyHex)
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// 组装: salt + nonce + ciphertext
	result := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	// 返回带前缀的 base64 编码
	return scryptPrefix + base64.StdEncoding.EncodeToString(result), nil
}

// DecryptPrivateKey 解密私钥 (自动兼容新旧格式)
// encrypted: 密文字符串 (新格式 "scrypt:base64..." 或旧格式 "base64...")
// encryptionKey: 加密密钥
// 返回: 原始私钥 hex 字符串
func (s *CryptoService) DecryptPrivateKey(encrypted string, encryptionKey string) (string, error) {
	if strings.HasPrefix(encrypted, scryptPrefix) {
		return s.decryptScrypt(encrypted[len(scryptPrefix):], encryptionKey)
	}
	return s.decryptLegacy(encrypted, encryptionKey)
}

// decryptScrypt 使用 scrypt KDF 解密 (新格式)
func (s *CryptoService) decryptScrypt(encodedData string, encryptionKey string) (string, error) {
	// base64 解码
	data, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return "", fmt.Errorf("base64 解码失败: %w", err)
	}

	// 分离 salt (前 16 字节)
	if len(data) < scryptSaltLen {
		return "", fmt.Errorf("密文数据长度不足: 缺少 salt")
	}
	salt := data[:scryptSaltLen]
	remaining := data[scryptSaltLen:]

	// 使用 scrypt 派生解密密钥
	key, err := scrypt.Key([]byte(encryptionKey), salt, scryptN, scryptR, scryptP, scryptKeyLen)
	if err != nil {
		return "", fmt.Errorf("scrypt 密钥派生失败: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	// 分离 nonce 和密文
	nonceSize := gcm.NonceSize()
	if len(remaining) < nonceSize {
		return "", fmt.Errorf("密文数据长度不足: 缺少 nonce")
	}
	nonce, ciphertext := remaining[:nonceSize], remaining[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// decryptLegacy 使用旧版 SHA256 解密 (向后兼容)
func (s *CryptoService) decryptLegacy(encrypted string, encryptionKey string) (string, error) {
	key := deriveKeyLegacy(encryptionKey)

	// base64 解码
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("base64 解码失败: %w", err)
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建 GCM 失败: %w", err)
	}

	// 分离 nonce 和密文
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文数据长度不足")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// deriveKeyLegacy 旧版密钥派生: SHA256 (仅用于解密历史数据)
func deriveKeyLegacy(encryptionKey string) []byte {
	hash := sha256.Sum256([]byte(encryptionKey))
	return hash[:]
}
