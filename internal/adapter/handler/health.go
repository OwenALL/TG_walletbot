// Package handler 提供 Admin API HTTP 处理器
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check 健康检查接口
func (h *HealthHandler) Check(c *gin.Context) {
	response.Success(c, gin.H{
		"status":  "ok",
		"service": "walletbot-api",
	})
}
