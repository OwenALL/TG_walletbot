// Package errors 提供统一的错误类型定义和错误处理工具
package errors

import (
	"fmt"
	"net/http"
)

// AppError 应用层统一错误类型
type AppError struct {
	Code    int    `json:"code"`    // HTTP 状态码
	Message string `json:"message"` // 用户可见的错误信息
	Err     error  `json:"-"`       // 内部错误 (不暴露给用户)
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 支持 errors.Is/As
func (e *AppError) Unwrap() error {
	return e.Err
}

// --- 通用错误构造函数 ---

// NewBadRequest 创建 400 错误
func NewBadRequest(message string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: message}
}

// NewUnauthorized 创建 401 错误
func NewUnauthorized(message string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: message}
}

// NewForbidden 创建 403 错误
func NewForbidden(message string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: message}
}

// NewNotFound 创建 404 错误
func NewNotFound(resource string, id interface{}) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("%s (ID: %v) 不存在", resource, id),
	}
}

// NewConflict 创建 409 错误
func NewConflict(message string) *AppError {
	return &AppError{Code: http.StatusConflict, Message: message}
}

// NewInternal 创建 500 错误
func NewInternal(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

// Wrap 包装已有错误为 AppError
func Wrap(err error, message string) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: message,
		Err:     err,
	}
}

// --- 钱包业务专用错误 ---

// NewInsufficientBalance 余额不足
func NewInsufficientBalance(currency string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("%s 余额不足", currency),
	}
}

// NewPINInvalid 支付密码错误
func NewPINInvalid() *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "支付密码错误，请重试",
	}
}

// NewPINLocked PIN 被锁定
func NewPINLocked(minutes int) *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: fmt.Sprintf("支付密码连续错误次数过多，账户已锁定 %d 分钟", minutes),
	}
}

// NewPINNotSet PIN 未设置
func NewPINNotSet() *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "请先设置支付密码",
	}
}

// NewUserFrozen 用户被冻结
func NewUserFrozen() *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: "您的账户已被冻结，如有疑问请联系客服",
	}
}

// NewWithdrawBelowMin 提币金额低于最低限制
func NewWithdrawBelowMin(min string, currency string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("最低提币金额为 %s %s", min, currency),
	}
}

// NewWithdrawExceedMax 提币金额超过上限
func NewWithdrawExceedMax(max string, currency string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("单笔提币上限为 %s %s", max, currency),
	}
}

// NewWithdrawDailyExceeded 提币超过日限额
func NewWithdrawDailyExceeded() *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "已超过今日提币限额",
	}
}

// NewInvalidAddress 无效地址
func NewInvalidAddress() *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "地址格式不正确，请检查后重试",
	}
}

// NewExchangeRateUnavailable 汇率不可用
func NewExchangeRateUnavailable(pair string) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: fmt.Sprintf("当前 %s 兑换暂不可用", pair),
	}
}

// NewTransferToSelf 不能转账给自己
func NewTransferToSelf() *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: "不能向自己转账",
	}
}

// NewRecipientNotFound 收款人不存在
func NewRecipientNotFound() *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: "收款人未注册，请确认对方已使用本 Bot",
	}
}

// NewRateLimited 操作太频繁
func NewRateLimited() *AppError {
	return &AppError{
		Code:    http.StatusTooManyRequests,
		Message: "操作太频繁，请稍后再试",
	}
}

// NewMaintenanceMode 维护模式
func NewMaintenanceMode() *AppError {
	return &AppError{
		Code:    http.StatusServiceUnavailable,
		Message: "系统维护中，请稍后再试",
	}
}

// --- 哨兵错误 (用于 errors.Is 比较) ---

var (
	ErrNotFound            = &AppError{Code: http.StatusNotFound, Message: "资源不存在"}
	ErrConflict            = &AppError{Code: http.StatusConflict, Message: "资源冲突"}
	ErrUnauthorized        = &AppError{Code: http.StatusUnauthorized, Message: "未授权"}
	ErrForbidden           = &AppError{Code: http.StatusForbidden, Message: "权限不足"}
	ErrInsufficientBalance = &AppError{Code: http.StatusBadRequest, Message: "余额不足"}
	ErrPINInvalid          = &AppError{Code: http.StatusBadRequest, Message: "支付密码错误"}
	ErrPINLocked           = &AppError{Code: http.StatusForbidden, Message: "账户已锁定"}
)
