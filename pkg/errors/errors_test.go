package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- 构造函数测试 ----------

func TestNewBadRequest(t *testing.T) {
	err := NewBadRequest("参数错误")
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "参数错误", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewUnauthorized(t *testing.T) {
	err := NewUnauthorized("未登录")
	assert.Equal(t, http.StatusUnauthorized, err.Code)
	assert.Equal(t, "未登录", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewForbidden(t *testing.T) {
	err := NewForbidden("权限不足")
	assert.Equal(t, http.StatusForbidden, err.Code)
	assert.Equal(t, "权限不足", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewNotFound(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		id       interface{}
		wantMsg  string
	}{
		{
			name:     "整数 ID",
			resource: "用户",
			id:       42,
			wantMsg:  "用户 (ID: 42) 不存在",
		},
		{
			name:     "字符串 ID",
			resource: "订单",
			id:       "abc-123",
			wantMsg:  "订单 (ID: abc-123) 不存在",
		},
		{
			name:     "零值 ID",
			resource: "钱包",
			id:       0,
			wantMsg:  "钱包 (ID: 0) 不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewNotFound(tt.resource, tt.id)
			assert.Equal(t, http.StatusNotFound, err.Code)
			assert.Equal(t, tt.wantMsg, err.Message)
			assert.Nil(t, err.Err)
		})
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict("资源已存在")
	assert.Equal(t, http.StatusConflict, err.Code)
	assert.Equal(t, "资源已存在", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewInternal(t *testing.T) {
	inner := fmt.Errorf("db connection refused")
	err := NewInternal("内部错误", inner)
	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Equal(t, "内部错误", err.Message)
	assert.Equal(t, inner, err.Err)
}

func TestNewInternal_NilErr(t *testing.T) {
	err := NewInternal("内部错误", nil)
	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Nil(t, err.Err)
}

// ---------- Wrap 测试 ----------

func TestWrap(t *testing.T) {
	inner := fmt.Errorf("io timeout")
	err := Wrap(inner, "操作失败")
	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Equal(t, "操作失败", err.Message)
	assert.Equal(t, inner, err.Err)
}

func TestWrap_NilErr(t *testing.T) {
	err := Wrap(nil, "无内部错误")
	assert.Equal(t, http.StatusInternalServerError, err.Code)
	assert.Nil(t, err.Err)
}

// ---------- Error() 方法测试 ----------

func TestAppError_Error_WithInnerErr(t *testing.T) {
	inner := fmt.Errorf("disk full")
	err := &AppError{Code: 500, Message: "写入失败", Err: inner}
	assert.Equal(t, "写入失败: disk full", err.Error())
}

func TestAppError_Error_WithoutInnerErr(t *testing.T) {
	err := &AppError{Code: 400, Message: "参数错误"}
	assert.Equal(t, "参数错误", err.Error())
}

// ---------- Unwrap() 方法测试 ----------

func TestAppError_Unwrap_WithErr(t *testing.T) {
	inner := fmt.Errorf("原始错误")
	err := &AppError{Code: 500, Message: "包装错误", Err: inner}
	assert.Equal(t, inner, err.Unwrap())
}

func TestAppError_Unwrap_WithoutErr(t *testing.T) {
	err := &AppError{Code: 400, Message: "无内部错误"}
	assert.Nil(t, err.Unwrap())
}

// ---------- errors.Is / errors.As 兼容性测试 ----------

func TestErrorsIs_WithWrappedSentinel(t *testing.T) {
	// 将标准哨兵错误包装后，errors.Is 应通过 Unwrap 链找到它
	sentinel := fmt.Errorf("sentinel")
	wrapped := &AppError{Code: 500, Message: "外层", Err: sentinel}
	assert.True(t, errors.Is(wrapped, sentinel))
}

func TestErrorsIs_SentinelDirect(t *testing.T) {
	// 哨兵错误与自身比较
	assert.True(t, errors.Is(ErrNotFound, ErrNotFound))
	assert.True(t, errors.Is(ErrConflict, ErrConflict))
	assert.True(t, errors.Is(ErrUnauthorized, ErrUnauthorized))
	assert.True(t, errors.Is(ErrForbidden, ErrForbidden))
	assert.True(t, errors.Is(ErrInsufficientBalance, ErrInsufficientBalance))
	assert.True(t, errors.Is(ErrPINInvalid, ErrPINInvalid))
	assert.True(t, errors.Is(ErrPINLocked, ErrPINLocked))
}

func TestErrorsIs_DifferentSentinels(t *testing.T) {
	// 不同哨兵之间不应相等
	assert.False(t, errors.Is(ErrNotFound, ErrConflict))
	assert.False(t, errors.Is(ErrUnauthorized, ErrForbidden))
}

func TestErrorsAs_AppError(t *testing.T) {
	inner := fmt.Errorf("底层错误")
	err := Wrap(inner, "包装")

	var appErr *AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, http.StatusInternalServerError, appErr.Code)
	assert.Equal(t, "包装", appErr.Message)
	assert.Equal(t, inner, appErr.Err)
}

func TestErrorsAs_NestedWrap(t *testing.T) {
	// 通过 fmt.Errorf %w 再次包装 AppError，errors.As 仍能提取
	appErr := NewBadRequest("验证失败")
	wrapped := fmt.Errorf("handler: %w", appErr)

	var target *AppError
	require.True(t, errors.As(wrapped, &target))
	assert.Equal(t, http.StatusBadRequest, target.Code)
	assert.Equal(t, "验证失败", target.Message)
}

// ---------- 钱包业务错误构造函数测试 ----------

func TestNewInsufficientBalance(t *testing.T) {
	err := NewInsufficientBalance("USDT")
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "USDT 余额不足", err.Message)
	assert.Nil(t, err.Err)
}

func TestNewPINInvalid(t *testing.T) {
	err := NewPINInvalid()
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "支付密码错误，请重试", err.Message)
}

func TestNewPINLocked(t *testing.T) {
	err := NewPINLocked(30)
	assert.Equal(t, http.StatusForbidden, err.Code)
	assert.Equal(t, "支付密码连续错误次数过多，账户已锁定 30 分钟", err.Message)
}

func TestNewPINNotSet(t *testing.T) {
	err := NewPINNotSet()
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "请先设置支付密码", err.Message)
}

func TestNewUserFrozen(t *testing.T) {
	err := NewUserFrozen()
	assert.Equal(t, http.StatusForbidden, err.Code)
	assert.Equal(t, "您的账户已被冻结，如有疑问请联系客服", err.Message)
}

func TestNewWithdrawBelowMin(t *testing.T) {
	err := NewWithdrawBelowMin("10", "USDT")
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "最低提币金额为 10 USDT", err.Message)
}

func TestNewWithdrawExceedMax(t *testing.T) {
	err := NewWithdrawExceedMax("50000", "TRX")
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "单笔提币上限为 50000 TRX", err.Message)
}

func TestNewWithdrawDailyExceeded(t *testing.T) {
	err := NewWithdrawDailyExceeded()
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "已超过今日提币限额", err.Message)
}

func TestNewInvalidAddress(t *testing.T) {
	err := NewInvalidAddress()
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "地址格式不正确，请检查后重试", err.Message)
}

func TestNewExchangeRateUnavailable(t *testing.T) {
	err := NewExchangeRateUnavailable("USDT/TRX")
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "当前 USDT/TRX 兑换暂不可用", err.Message)
}

func TestNewTransferToSelf(t *testing.T) {
	err := NewTransferToSelf()
	assert.Equal(t, http.StatusBadRequest, err.Code)
	assert.Equal(t, "不能向自己转账", err.Message)
}

func TestNewRecipientNotFound(t *testing.T) {
	err := NewRecipientNotFound()
	assert.Equal(t, http.StatusNotFound, err.Code)
	assert.Equal(t, "收款人未注册，请确认对方已使用本 Bot", err.Message)
}

func TestNewRateLimited(t *testing.T) {
	err := NewRateLimited()
	assert.Equal(t, http.StatusTooManyRequests, err.Code)
	assert.Equal(t, "操作太频繁，请稍后再试", err.Message)
}

func TestNewMaintenanceMode(t *testing.T) {
	err := NewMaintenanceMode()
	assert.Equal(t, http.StatusServiceUnavailable, err.Code)
	assert.Equal(t, "系统维护中，请稍后再试", err.Message)
}

// ---------- AppError 实现 error 接口验证 ----------

func TestAppError_ImplementsErrorInterface(t *testing.T) {
	var _ error = (*AppError)(nil)
}
