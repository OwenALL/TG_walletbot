package config

import (
	"strings"
	"testing"
)

// TestValidateJWTSecret_长度不足 验证短密钥被拒绝
func TestValidateJWTSecret_长度不足(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{"空字符串", ""},
		{"1个字符", "a"},
		{"16个字符", "abcdefghijklmnop"},
		{"31个字符", strings.Repeat("x", 31)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if err == nil {
				t.Errorf("期望报错，但 secret=%q (len=%d) 通过了验证", tt.secret, len(tt.secret))
			}
			if err != nil && !strings.Contains(err.Error(), "长度不足") {
				t.Errorf("期望包含 '长度不足' 错误信息，实际: %v", err)
			}
		})
	}
}

// TestValidateJWTSecret_弱密钥 验证常见弱密钥被拒绝
func TestValidateJWTSecret_弱密钥(t *testing.T) {
	// 这些弱密钥需要补到 >= 32 字符才能通过长度检查，所以先测 IsWeakJWTSecret
	tests := []struct {
		secret string
		isWeak bool
	}{
		{"secret", true},
		{"changeme", true},
		{"change_me_in_production", true},
		{"JWT-SECRET", true}, // 大小写不敏感
		{"Password", true},
		{"this-is-a-very-strong-random-key-abc123", false},
	}

	for _, tt := range tests {
		t.Run(tt.secret, func(t *testing.T) {
			result := IsWeakJWTSecret(tt.secret)
			if result != tt.isWeak {
				t.Errorf("IsWeakJWTSecret(%q) = %v, 期望 %v", tt.secret, result, tt.isWeak)
			}
		})
	}
}

// TestValidateJWTSecret_合格密钥 验证合格密钥通过验证
func TestValidateJWTSecret_合格密钥(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{"正好32字符", strings.Repeat("a", 32)},
		{"64字符hex", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
		{"长随机字符串", "this-is-a-very-strong-and-secure-jwt-secret-key-2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJWTSecret(tt.secret)
			if err != nil {
				t.Errorf("合格密钥 %q 验证失败: %v", tt.secret, err)
			}
		})
	}
}

// TestValidateForAPI_JWT检查 验证 ValidateForAPI 中的 JWT 检查
func TestValidateForAPI_JWT检查(t *testing.T) {
	// JWT Secret 太短
	cfg := &Config{
		JWT: JWTConfig{
			Secret:      "short",
			ExpireHours: 24,
		},
	}
	err := cfg.ValidateForAPI()
	if err == nil {
		t.Error("期望 ValidateForAPI 对短密钥报错")
	}

	// JWT Secret 合格
	cfg.JWT.Secret = strings.Repeat("x", 32)
	err = cfg.ValidateForAPI()
	if err != nil {
		t.Errorf("合格密钥不应报错: %v", err)
	}
}

// TestCommonWeakSecrets_不区分大小写 验证弱密钥检测不区分大小写
func TestCommonWeakSecrets_不区分大小写(t *testing.T) {
	tests := []string{
		"SECRET", "Secret", "sEcReT",
		"CHANGEME", "ChangEMe",
		"JWT-Secret", "jwt-SECRET",
	}

	for _, s := range tests {
		if !IsWeakJWTSecret(s) {
			t.Errorf("IsWeakJWTSecret(%q) 应返回 true (不区分大小写)", s)
		}
	}
}
