// Package response 提供统一的 API 响应格式封装
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// Response API 统一响应结构
type Response struct {
	Code    int         `json:"code"`    // 业务状态码 (0=成功)
	Message string      `json:"message"` // 响应信息
	Data    interface{} `json:"data"`    // 响应数据
}

// PaginatedData 分页响应数据
type PaginatedData struct {
	List  interface{} `json:"list"`  // 数据列表
	Total int64       `json:"total"` // 总记录数
	Page  int         `json:"page"`  // 当前页码
	Size  int         `json:"size"`  // 每页数量
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Created 返回创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// SuccessWithPagination 返回带分页的成功响应
func SuccessWithPagination(c *gin.Context, list interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PaginatedData{
			List:  list,
			Total: total,
			Page:  page,
			Size:  size,
		},
	})
}

// Error 返回错误响应
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Code:    statusCode,
		Message: message,
		Data:    nil,
	})
}

// ErrorFromAppError 从 AppError 返回错误响应
func ErrorFromAppError(c *gin.Context, err *apperrors.AppError) {
	c.JSON(err.Code, Response{
		Code:    err.Code,
		Message: err.Message,
		Data:    nil,
	})
}

// BadRequest 返回 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 返回 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden 返回 403 错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 返回 404 错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalError 返回 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
