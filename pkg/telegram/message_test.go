package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUserFriendlyErrors_AllCodesExist 测试所有已知错误码都有对应友好提示
func TestUserFriendlyErrors_AllCodesExist(t *testing.T) {
	expectedCodes := []string{
		"insufficient_balance",
		"invalid_address",
		"network_error",
		"rate_limit",
		"pin_invalid",
		"pin_locked",
		"user_frozen",
		"maintenance",
		"default",
	}

	for _, code := range expectedCodes {
		t.Run(code, func(t *testing.T) {
			msg, ok := userFriendlyErrors[code]
			assert.True(t, ok, "错误码 %s 应存在于映射中", code)
			assert.NotEmpty(t, msg, "错误码 %s 对应的提示消息不应为空", code)
		})
	}
}

// TestUserFriendlyErrors_DefaultExists 测试默认错误提示存在
func TestUserFriendlyErrors_DefaultExists(t *testing.T) {
	msg, ok := userFriendlyErrors["default"]
	assert.True(t, ok, "必须有 default 错误提示")
	assert.NotEmpty(t, msg)
}

// TestUserFriendlyErrors_UnknownCodeFallback 测试未知错误码回退到 default
func TestUserFriendlyErrors_UnknownCodeFallback(t *testing.T) {
	unknownCodes := []string{
		"unknown_error",
		"something_weird",
		"",
		"UPPERCASE_ERROR",
	}

	defaultMsg := userFriendlyErrors["default"]

	for _, code := range unknownCodes {
		t.Run("unknown:"+code, func(t *testing.T) {
			msg, ok := userFriendlyErrors[code]
			if !ok {
				// 未知错误码应使用 default
				msg = defaultMsg
			}
			assert.NotEmpty(t, msg, "即使未知错误码也应有默认提示")
		})
	}
}

// TestUserFriendlyErrors_NoRawErrors 测试提示消息中不含技术性内容
func TestUserFriendlyErrors_NoRawErrors(t *testing.T) {
	// 确保所有错误提示不含技术性词语，对用户友好
	technicalKeywords := []string{
		"error",
		"err",
		"nil",
		"panic",
		"exception",
		"stack",
		"trace",
		"sql",
		"redis",
		"mysql",
		"internal",
	}

	for code, msg := range userFriendlyErrors {
		t.Run(code, func(t *testing.T) {
			for _, kw := range technicalKeywords {
				assert.NotContains(t, msg, kw, "友好提示不应包含技术术语: %s", kw)
			}
		})
	}
}

// TestUserFriendlyErrors_MessagesNotTooLong 测试提示消息长度合理
func TestUserFriendlyErrors_MessagesNotTooLong(t *testing.T) {
	// Telegram 消息不宜过长，错误提示应简洁
	maxLen := 100

	for code, msg := range userFriendlyErrors {
		t.Run(code, func(t *testing.T) {
			assert.LessOrEqual(t, len([]rune(msg)), maxLen,
				"错误提示 %s 过长 (超过 %d 个字符)", code, maxLen)
		})
	}
}

// TestGetFriendlyError 模拟 SendError/EditError 的错误码查找逻辑
func TestGetFriendlyError(t *testing.T) {
	// 模拟 SendError/EditError 内部的查找逻辑
	getFriendlyMsg := func(errCode string) string {
		msg, ok := userFriendlyErrors[errCode]
		if !ok {
			msg = userFriendlyErrors["default"]
		}
		return msg
	}

	tests := []struct {
		errCode  string
		expected string
	}{
		{"insufficient_balance", "余额不足，请先充值"},
		{"invalid_address", "地址格式不正确，请检查后重试"},
		{"network_error", "网络繁忙，请稍后重试"},
		{"rate_limit", "操作太频繁，请稍后再试"},
		{"pin_invalid", "支付密码错误，请重试"},
		{"pin_locked", "支付密码连续错误次数过多，账户已锁定"},
		{"user_frozen", "您的账户已被冻结，如有疑问请联系客服"},
		{"maintenance", "系统维护中，请稍后再试"},
		{"default", "操作失败，请稍后重试"},
		// 未知错误码应使用 default
		{"unknown_xxx", "操作失败，请稍后重试"},
		{"", "操作失败，请稍后重试"},
	}

	for _, tt := range tests {
		t.Run(tt.errCode, func(t *testing.T) {
			msg := getFriendlyMsg(tt.errCode)
			assert.Equal(t, tt.expected, msg)
		})
	}
}
