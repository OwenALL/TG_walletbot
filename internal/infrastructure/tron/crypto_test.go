package tron

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCryptoService_加密解密往返 验证 scrypt 加密后解密能正确返回原始私钥
func TestCryptoService_加密解密往返(t *testing.T) {
	cs := NewCryptoService()

	tests := []struct {
		name          string
		privateKeyHex string
		encryptionKey string
	}{
		{
			name:          "标准 64 字符私钥",
			privateKeyHex: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			encryptionKey: "test-encryption-key-12345",
		},
		{
			name:          "全零私钥",
			privateKeyHex: "0000000000000000000000000000000000000000000000000000000000000001",
			encryptionKey: "another-key",
		},
		{
			name:          "空加密密钥",
			privateKeyHex: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			encryptionKey: "",
		},
		{
			name:          "长加密密钥",
			privateKeyHex: "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			encryptionKey: "this-is-a-very-long-encryption-key-that-exceeds-32-bytes-and-will-be-hashed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := cs.EncryptPrivateKey(tt.privateKeyHex, tt.encryptionKey)
			require.NoError(t, err, "加密不应返回错误")
			require.NotEmpty(t, encrypted, "加密结果不应为空")

			// 验证新格式有 scrypt 前缀
			assert.True(t, strings.HasPrefix(encrypted, "scrypt:"), "新加密应使用 scrypt 格式前缀")

			// 解密
			decrypted, err := cs.DecryptPrivateKey(encrypted, tt.encryptionKey)
			require.NoError(t, err, "解密不应返回错误")

			// 验证往返一致
			assert.Equal(t, tt.privateKeyHex, decrypted, "解密后应与原始私钥一致")
		})
	}
}

// TestCryptoService_不同密钥解密失败 验证使用不同的加密密钥解密应报错
func TestCryptoService_不同密钥解密失败(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	encryptKey := "correct-key"
	wrongKey := "wrong-key"

	// 用正确密钥加密
	encrypted, err := cs.EncryptPrivateKey(privateKeyHex, encryptKey)
	require.NoError(t, err, "加密不应返回错误")

	// 用错误密钥解密
	_, err = cs.DecryptPrivateKey(encrypted, wrongKey)
	assert.Error(t, err, "使用错误密钥解密应返回错误")
	assert.Contains(t, err.Error(), "解密失败", "错误信息应包含'解密失败'")
}

// TestCryptoService_密文是ScryptBase64格式 验证加密结果是 scrypt 前缀 + 有效的 base64 字符串
func TestCryptoService_密文是ScryptBase64格式(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	encryptKey := "test-key"

	encrypted, err := cs.EncryptPrivateKey(privateKeyHex, encryptKey)
	require.NoError(t, err, "加密不应返回错误")

	// 验证 scrypt 前缀
	assert.True(t, strings.HasPrefix(encrypted, "scrypt:"), "加密结果应有 scrypt: 前缀")

	// 去掉前缀后应为有效 base64
	b64Part := encrypted[len("scrypt:"):]
	decoded, err := base64.StdEncoding.DecodeString(b64Part)
	assert.NoError(t, err, "去掉前缀后应为有效的 base64 字符串")

	// 验证密文长度: salt(16) + nonce(12) + 明文(64) + GCM tag(16) = 108 字节
	assert.True(t, len(decoded) >= 16+12+16, "密文长度应至少包含 salt(16) + nonce(12) + GCM tag(16)，实际: %d", len(decoded))
}

// TestCryptoService_每次加密结果不同 验证由于随机 salt+nonce，相同输入每次加密结果不同
func TestCryptoService_每次加密结果不同(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	encryptKey := "test-key"

	// 对相同输入加密多次
	count := 5
	results := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		encrypted, err := cs.EncryptPrivateKey(privateKeyHex, encryptKey)
		require.NoError(t, err, "第 %d 次加密不应返回错误", i+1)

		_, exists := results[encrypted]
		assert.False(t, exists, "第 %d 次加密结果与之前重复 (随机 salt+nonce 应使每次结果不同)", i+1)
		results[encrypted] = struct{}{}

		// 验证每次加密的结果都能正确解密
		decrypted, err := cs.DecryptPrivateKey(encrypted, encryptKey)
		require.NoError(t, err, "第 %d 次解密不应返回错误", i+1)
		assert.Equal(t, privateKeyHex, decrypted, "第 %d 次解密结果应与原始私钥一致", i+1)
	}

	assert.Equal(t, count, len(results), "应产生 %d 个不同的密文", count)
}

// TestCryptoService_旧格式兼容解密 验证能正确解密旧版 SHA256 格式的密文
func TestCryptoService_旧格式兼容解密(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	encryptionKey := "test-encryption-key-12345"

	// 使用旧版 SHA256 方式加密 (模拟历史数据)
	legacyEncrypted := encryptLegacy(t, privateKeyHex, encryptionKey)

	// 验证旧格式没有 scrypt 前缀
	assert.False(t, strings.HasPrefix(legacyEncrypted, "scrypt:"), "旧格式不应有 scrypt 前缀")

	// 用新的 DecryptPrivateKey 解密旧格式
	decrypted, err := cs.DecryptPrivateKey(legacyEncrypted, encryptionKey)
	require.NoError(t, err, "解密旧格式不应返回错误")
	assert.Equal(t, privateKeyHex, decrypted, "解密旧格式后应与原始私钥一致")
}

// TestCryptoService_旧格式错误密钥失败 验证旧格式使用错误密钥解密失败
func TestCryptoService_旧格式错误密钥失败(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	encryptionKey := "correct-key"
	wrongKey := "wrong-key"

	// 使用旧版加密
	legacyEncrypted := encryptLegacy(t, privateKeyHex, encryptionKey)

	// 错误密钥解密旧格式应失败
	_, err := cs.DecryptPrivateKey(legacyEncrypted, wrongKey)
	assert.Error(t, err, "旧格式使用错误密钥解密应失败")
}

// TestCryptoService_新旧格式互操作 验证新加密的数据无法被旧方式解密（安全性提升）
func TestCryptoService_新旧格式互操作(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	encryptionKey := "shared-key"

	// 新格式加密
	newEncrypted, err := cs.EncryptPrivateKey(privateKeyHex, encryptionKey)
	require.NoError(t, err)

	// 新格式能正确解密
	decrypted, err := cs.DecryptPrivateKey(newEncrypted, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, privateKeyHex, decrypted)

	// 旧格式加密
	legacyEncrypted := encryptLegacy(t, privateKeyHex, encryptionKey)

	// 旧格式也能正确解密
	decrypted, err = cs.DecryptPrivateKey(legacyEncrypted, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, privateKeyHex, decrypted)
}

// TestCryptoService_截断密文解密失败 验证被篡改或截断的密文解密失败
func TestCryptoService_截断密文解密失败(t *testing.T) {
	cs := NewCryptoService()

	tests := []struct {
		name      string
		encrypted string
		wantErr   string
	}{
		{
			name:      "scrypt 前缀但无数据",
			encrypted: "scrypt:",
			wantErr:   "密文数据长度不足",
		},
		{
			name:      "scrypt 前缀但 base64 无效",
			encrypted: "scrypt:!!!invalid-base64!!!",
			wantErr:   "base64 解码失败",
		},
		{
			name:      "旧格式 base64 无效",
			encrypted: "!!!invalid-base64!!!",
			wantErr:   "base64 解码失败",
		},
		{
			name:      "旧格式数据太短",
			encrypted: base64.StdEncoding.EncodeToString([]byte("short")),
			wantErr:   "密文数据长度不足",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cs.DecryptPrivateKey(tt.encrypted, "any-key")
			assert.Error(t, err, "应返回错误")
			assert.Contains(t, err.Error(), tt.wantErr, "错误信息应包含期望的内容")
		})
	}
}

// --- 补充测试: 新格式 scrypt 加密解密 ---

// TestCryptoService_新格式加密解密_不同私钥长度 验证各种长度私钥的加密解密
func TestCryptoService_新格式加密解密_不同私钥长度(t *testing.T) {
	cs := NewCryptoService()

	tests := []struct {
		name          string
		privateKeyHex string
		encryptionKey string
	}{
		{
			name:          "最短私钥 (2 字符)",
			privateKeyHex: "ab",
			encryptionKey: "key-1",
		},
		{
			name:          "标准 secp256k1 私钥 (64 字符 hex = 32 字节)",
			privateKeyHex: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			encryptionKey: "key-2",
		},
		{
			name:          "极长私钥 (256 字符)",
			privateKeyHex: strings.Repeat("ff", 128),
			encryptionKey: "key-3",
		},
		{
			name:          "Unicode 加密密钥",
			privateKeyHex: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			encryptionKey: "密钥-中文-测试-🔐",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := cs.EncryptPrivateKey(tt.privateKeyHex, tt.encryptionKey)
			require.NoError(t, err, "加密不应失败")
			require.True(t, strings.HasPrefix(encrypted, scryptPrefix), "应使用 scrypt 格式")

			decrypted, err := cs.DecryptPrivateKey(encrypted, tt.encryptionKey)
			require.NoError(t, err, "解密不应失败")
			assert.Equal(t, tt.privateKeyHex, decrypted, "解密后应与原始值一致")
		})
	}
}

// TestCryptoService_新格式_空私钥 验证空私钥字符串的加密解密
func TestCryptoService_新格式_空私钥(t *testing.T) {
	cs := NewCryptoService()

	encrypted, err := cs.EncryptPrivateKey("", "some-key")
	require.NoError(t, err, "空私钥加密不应报错")
	require.True(t, strings.HasPrefix(encrypted, scryptPrefix))

	decrypted, err := cs.DecryptPrivateKey(encrypted, "some-key")
	require.NoError(t, err, "空私钥解密不应报错")
	assert.Equal(t, "", decrypted, "解密结果应为空字符串")
}

// --- 补充测试: 旧格式兼容性 ---

// TestCryptoService_旧格式兼容_多种密钥 验证各种加密密钥下旧格式均可解密
func TestCryptoService_旧格式兼容_多种密钥(t *testing.T) {
	cs := NewCryptoService()

	tests := []struct {
		name          string
		privateKeyHex string
		encryptionKey string
	}{
		{
			name:          "短密钥",
			privateKeyHex: "aabb",
			encryptionKey: "a",
		},
		{
			name:          "空密钥",
			privateKeyHex: "deadbeef",
			encryptionKey: "",
		},
		{
			name:          "64 字符私钥 + 长密钥",
			privateKeyHex: "a1a2a3a4a5a6a7a8a1a2a3a4a5a6a7a8a1a2a3a4a5a6a7a8a1a2a3a4a5a6a7a8",
			encryptionKey: "a-very-long-encryption-key-that-is-longer-than-32-bytes-for-sure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 用旧版方式加密
			legacyEncrypted := encryptLegacy(t, tt.privateKeyHex, tt.encryptionKey)
			assert.False(t, strings.HasPrefix(legacyEncrypted, scryptPrefix), "旧格式不应有 scrypt 前缀")

			// 用 DecryptPrivateKey 解密 (应自动走 legacy 路径)
			decrypted, err := cs.DecryptPrivateKey(legacyEncrypted, tt.encryptionKey)
			require.NoError(t, err, "旧格式解密不应失败")
			assert.Equal(t, tt.privateKeyHex, decrypted, "旧格式解密后应与原始值一致")
		})
	}
}

// TestCryptoService_旧格式不可被新密钥解密 验证旧格式密文的密钥绑定性
func TestCryptoService_旧格式不可被新密钥解密(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	legacyEncrypted := encryptLegacy(t, privateKeyHex, "original-key")

	// 用 5 个不同的错误密钥尝试解密
	wrongKeys := []string{"wrong-key-1", "wrong-key-2", "key-original", "Original-key", " original-key"}
	for _, wk := range wrongKeys {
		_, err := cs.DecryptPrivateKey(legacyEncrypted, wk)
		assert.Error(t, err, "旧格式用密钥 '%s' 解密应失败", wk)
	}
}

// --- 补充测试: 错误密钥解密失败 ---

// TestCryptoService_新格式错误密钥_微小差异 验证加密密钥的敏感性 (微小差异即解密失败)
func TestCryptoService_新格式错误密钥_微小差异(t *testing.T) {
	cs := NewCryptoService()

	privateKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	encryptionKey := "my-secret-key-2024"

	encrypted, err := cs.EncryptPrivateKey(privateKeyHex, encryptionKey)
	require.NoError(t, err)

	tests := []struct {
		name     string
		wrongKey string
	}{
		{name: "追加空格", wrongKey: "my-secret-key-2024 "},
		{name: "前导空格", wrongKey: " my-secret-key-2024"},
		{name: "大小写差异", wrongKey: "My-Secret-Key-2024"},
		{name: "少一个字符", wrongKey: "my-secret-key-202"},
		{name: "多一个字符", wrongKey: "my-secret-key-2024x"},
		{name: "完全不同", wrongKey: "totally-different-key"},
		{name: "空密钥", wrongKey: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cs.DecryptPrivateKey(encrypted, tt.wrongKey)
			assert.Error(t, err, "使用错误密钥 '%s' 应解密失败", tt.wrongKey)
		})
	}
}

// --- 补充测试: 空输入/损坏密文异常处理 ---

// TestCryptoService_损坏密文_scrypt格式 验证各种损坏的 scrypt 格式密文
func TestCryptoService_损坏密文_scrypt格式(t *testing.T) {
	cs := NewCryptoService()

	// 先生成一个正常密文作为基础
	encrypted, err := cs.EncryptPrivateKey("deadbeef", "test-key")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(encrypted, scryptPrefix))

	b64Part := encrypted[len(scryptPrefix):]
	rawBytes, err := base64.StdEncoding.DecodeString(b64Part)
	require.NoError(t, err)

	tests := []struct {
		name      string
		encrypted string
		wantErr   string
	}{
		{
			name:      "scrypt 前缀 + 空 base64",
			encrypted: scryptPrefix,
			wantErr:   "密文数据长度不足",
		},
		{
			name:      "scrypt 前缀 + 非法 base64",
			encrypted: scryptPrefix + "@#$%^&*(",
			wantErr:   "base64 解码失败",
		},
		{
			name:      "scrypt 前缀 + 只有 salt (16 字节，缺少 nonce 和密文)",
			encrypted: scryptPrefix + base64.StdEncoding.EncodeToString(rawBytes[:16]),
			wantErr:   "密文数据长度不足",
		},
		{
			name: "scrypt 前缀 + salt + 不完整 nonce (16+6 字节)",
			encrypted: scryptPrefix + base64.StdEncoding.EncodeToString(rawBytes[:22]),
			wantErr: "密文数据长度不足",
		},
		{
			name: "scrypt 前缀 + 篡改密文 (翻转最后一个字节)",
			encrypted: func() string {
				tampered := make([]byte, len(rawBytes))
				copy(tampered, rawBytes)
				tampered[len(tampered)-1] ^= 0xff // 翻转最后一字节
				return scryptPrefix + base64.StdEncoding.EncodeToString(tampered)
			}(),
			wantErr: "解密失败",
		},
		{
			name: "scrypt 前缀 + 篡改 salt (翻转第一个字节)",
			encrypted: func() string {
				tampered := make([]byte, len(rawBytes))
				copy(tampered, rawBytes)
				tampered[0] ^= 0xff // 翻转 salt 第一字节 → 派生不同密钥 → 解密失败
				return scryptPrefix + base64.StdEncoding.EncodeToString(tampered)
			}(),
			wantErr: "解密失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cs.DecryptPrivateKey(tt.encrypted, "test-key")
			require.Error(t, err, "应返回错误")
			assert.Contains(t, err.Error(), tt.wantErr, "错误信息应包含: %s", tt.wantErr)
		})
	}
}

// TestCryptoService_损坏密文_旧格式 验证各种损坏的旧格式密文
func TestCryptoService_损坏密文_旧格式(t *testing.T) {
	cs := NewCryptoService()

	// 生成正常旧格式密文
	legacyEncrypted := encryptLegacy(t, "deadbeef", "test-key")
	rawBytes, err := base64.StdEncoding.DecodeString(legacyEncrypted)
	require.NoError(t, err)

	tests := []struct {
		name      string
		encrypted string
		wantErr   string
	}{
		{
			name:      "空字符串",
			encrypted: "",
			wantErr:   "密文数据长度不足", // 空字符串 base64 解码为空字节 → 长度不足
		},
		{
			name:      "非法 base64",
			encrypted: "not!valid@base64#",
			wantErr:   "base64 解码失败",
		},
		{
			name:      "数据太短 (小于 nonce 大小)",
			encrypted: base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03}),
			wantErr:   "密文数据长度不足",
		},
		{
			name: "篡改密文尾部",
			encrypted: func() string {
				tampered := make([]byte, len(rawBytes))
				copy(tampered, rawBytes)
				tampered[len(tampered)-1] ^= 0xff
				return base64.StdEncoding.EncodeToString(tampered)
			}(),
			wantErr: "解密失败",
		},
		{
			name: "截断密文 (只保留 nonce)",
			encrypted: base64.StdEncoding.EncodeToString(rawBytes[:12]),
			// nonce 长度恰好 = 12，ciphertext 为空 → GCM Open 失败
			wantErr: "解密失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cs.DecryptPrivateKey(tt.encrypted, "test-key")
			require.Error(t, err, "应返回错误")
			assert.Contains(t, err.Error(), tt.wantErr, "错误信息应包含: %s", tt.wantErr)
		})
	}
}

// TestCryptoService_格式路由正确 验证 DecryptPrivateKey 根据前缀正确路由到 scrypt 或 legacy
func TestCryptoService_格式路由正确(t *testing.T) {
	cs := NewCryptoService()
	privateKey := "a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8a1b2c3d4e5f6a7b8"
	encKey := "route-test-key"

	// 新格式加密 → 应有 scrypt: 前缀
	newEnc, err := cs.EncryptPrivateKey(privateKey, encKey)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(newEnc, scryptPrefix), "EncryptPrivateKey 应生成 scrypt 前缀")

	// 旧格式加密 → 不应有 scrypt: 前缀
	legacyEnc := encryptLegacy(t, privateKey, encKey)
	assert.False(t, strings.HasPrefix(legacyEnc, scryptPrefix), "旧格式不应有 scrypt 前缀")

	// 两种格式都能正确解密
	d1, err := cs.DecryptPrivateKey(newEnc, encKey)
	require.NoError(t, err)
	assert.Equal(t, privateKey, d1, "新格式解密应正确")

	d2, err := cs.DecryptPrivateKey(legacyEnc, encKey)
	require.NoError(t, err)
	assert.Equal(t, privateKey, d2, "旧格式解密应正确")
}

// TestCryptoService_新格式密文结构 验证 scrypt 密文的内部数据结构
func TestCryptoService_新格式密文结构(t *testing.T) {
	cs := NewCryptoService()

	privateKey := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	encrypted, err := cs.EncryptPrivateKey(privateKey, "struct-test-key")
	require.NoError(t, err)

	// 去掉 scrypt: 前缀
	b64Part := encrypted[len(scryptPrefix):]
	raw, err := base64.StdEncoding.DecodeString(b64Part)
	require.NoError(t, err)

	// 验证最小长度: salt(16) + nonce(12) + 明文(64) + GCM tag(16) = 108
	// 实际 GCM overhead = 16 bytes (auth tag)
	expectedMinLen := scryptSaltLen + 12 + len(privateKey) + 16 // 16 + 12 + 64 + 16 = 108
	assert.Equal(t, expectedMinLen, len(raw), "密文原始数据长度应为 salt(%d) + nonce(12) + plaintext(%d) + tag(16) = %d",
		scryptSaltLen, len(privateKey), expectedMinLen)

	// 前 16 字节是 salt
	salt := raw[:scryptSaltLen]
	assert.Equal(t, scryptSaltLen, len(salt), "salt 长度应为 %d", scryptSaltLen)

	// 接下来 12 字节是 nonce
	nonce := raw[scryptSaltLen : scryptSaltLen+12]
	assert.Equal(t, 12, len(nonce), "nonce 长度应为 12 (GCM 标准)")
}

// encryptLegacy 使用旧版 SHA256 方式加密 (仅测试辅助)
func encryptLegacy(t *testing.T, privateKeyHex, encryptionKey string) string {
	t.Helper()

	hash := sha256.Sum256([]byte(encryptionKey))
	key := hash[:]

	block, err := aes.NewCipher(key)
	require.NoError(t, err)

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	require.NoError(t, err)

	ciphertext := gcm.Seal(nonce, nonce, []byte(privateKeyHex), nil)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
