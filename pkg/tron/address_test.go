package tron

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateAddress_ValidAddresses 测试合法 TRON 地址
func TestValidateAddress_ValidAddresses(t *testing.T) {
	// 已知合法的 TRON 主网地址
	validAddresses := []string{
		"TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW", // TRON Foundation 地址
		"TNPeeaaFB7K9cmo4uQpcU32zGK8G1NYqeL", // 常见交易所地址
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", // 另一个合法地址
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // USDT-TRC20 合约地址
	}

	for _, addr := range validAddresses {
		t.Run(addr, func(t *testing.T) {
			assert.True(t, ValidateAddress(addr), "合法地址 %s 应验证通过", addr)
		})
	}
}

// TestValidateAddress_InvalidAddresses 测试非法 TRON 地址
func TestValidateAddress_InvalidAddresses(t *testing.T) {
	tests := []struct {
		name    string
		address string
		reason  string
	}{
		{
			name:    "空字符串",
			address: "",
			reason:  "空字符串不是合法地址",
		},
		{
			name:    "长度不足",
			address: "T123",
			reason:  "地址长度不足 34 字符",
		},
		{
			name:    "长度超出",
			address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMWW",
			reason:  "地址长度超过 34 字符",
		},
		{
			name:    "不以T开头",
			address: "1JCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			reason:  "TRON 地址必须以 T 开头",
		},
		{
			name:    "非Base58字符(含0)",
			address: "T0CnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			reason:  "包含非 Base58 字符 0",
		},
		{
			name:    "非Base58字符(含O)",
			address: "TOCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			reason:  "包含非 Base58 字符 O",
		},
		{
			name:    "非Base58字符(含l)",
			address: "TlCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			reason:  "包含非 Base58 字符 l",
		},
		{
			name:    "非Base58字符(含I)",
			address: "TICnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			reason:  "包含非 Base58 字符 I",
		},
		{
			name:    "校验和错误",
			address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMX",
			reason:  "最后一位被篡改，校验和不匹配",
		},
		{
			name:    "篡改中间字符",
			address: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxNW",
			reason:  "中间字符被篡改，校验和不匹配",
		},
		{
			name:    "ETH地址格式",
			address: "0x742d35Cc6634C0532925a3b844",
			reason:  "这是以太坊地址格式，不是 TRON",
		},
		{
			name:    "BTC地址格式",
			address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			reason:  "这是比特币地址格式，不是 TRON",
		},
		{
			name:    "全T字符",
			address: "TTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTTT",
			reason:  "34个T不是合法的 TRON 地址",
		},
		{
			name:    "含特殊字符",
			address: "TJCnKsPa7y5okkXvQAidZBzqx3Q!Q6sMW",
			reason:  "含有非 Base58 特殊字符",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, ValidateAddress(tt.address), "非法地址应验证失败: %s", tt.reason)
		})
	}
}

// TestFormatAddress 测试地址格式化显示
func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "标准34字符地址",
			address:  "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			expected: "TJCnKs...Q6sxMW",
		},
		{
			name:     "短字符串不截断-正好12字符",
			address:  "123456789012",
			expected: "123456789012",
		},
		{
			name:     "短字符串不截断-少于12字符",
			address:  "TShort",
			expected: "TShort",
		},
		{
			name:     "空字符串",
			address:  "",
			expected: "",
		},
		{
			name:     "13字符-刚好触发截断",
			address:  "1234567890123",
			expected: "123456...890123",
		},
		{
			name:     "另一个合法TRON地址",
			address:  "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			expected: "TR7NHq...gjLj6t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBase58Decode 测试 base58 解码
func TestBase58Decode(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		isNil  bool
		reason string
	}{
		{
			name:   "空字符串",
			input:  "",
			isNil:  false,
			reason: "空字符串解码为空字节数组",
		},
		{
			name:   "含非法字符0",
			input:  "T0invalid",
			isNil:  true,
			reason: "字符 0 不在 Base58 字符集中",
		},
		{
			name:   "含非法字符O",
			input:  "TOinvalid",
			isNil:  true,
			reason: "字符 O 不在 Base58 字符集中",
		},
		{
			name:   "含非法字符l",
			input:  "Tlinvalid",
			isNil:  true,
			reason: "字符 l 不在 Base58 字符集中",
		},
		{
			name:   "含非法字符I",
			input:  "TIinvalid",
			isNil:  true,
			reason: "字符 I 不在 Base58 字符集中",
		},
		{
			name:   "合法Base58字符串",
			input:  "2NEpo7TZRRrLZSi2U",
			isNil:  false,
			reason: "纯 Base58 字符应正常解码",
		},
		{
			name:   "前导1字符(表示前导零字节)",
			input:  "111",
			isNil:  false,
			reason: "连续的 1 表示前导零字节",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base58Decode(tt.input)
			if tt.isNil {
				assert.Nil(t, result, "期望解码失败返回 nil: %s", tt.reason)
			} else {
				assert.NotNil(t, result, "期望解码成功: %s", tt.reason)
			}
		})
	}
}

// TestBase58Decode_LeadingZeros 测试前导零字节处理
func TestBase58Decode_LeadingZeros(t *testing.T) {
	// 前导 '1' 字符在 Base58 中表示前导零字节
	result := base58Decode("111")
	assert.NotNil(t, result)
	// 3 个 '1' 应产生 3 个前导零字节
	assert.Len(t, result, 3)
	for i := 0; i < 3; i++ {
		assert.Equal(t, byte(0), result[i], "第 %d 个字节应为 0x00", i)
	}
}

// TestValidateAddress_KnownContractAddresses 测试已知合约地址
func TestValidateAddress_KnownContractAddresses(t *testing.T) {
	// USDT-TRC20 合约地址
	assert.True(t, ValidateAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"))
}
