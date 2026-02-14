package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// AdminAuditLogHandler 审计日志处理器
type AdminAuditLogHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminAuditLogHandler 创建审计日志处理器
func NewAdminAuditLogHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminAuditLogHandler {
	return &AdminAuditLogHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取审计日志列表
func (h *AdminAuditLogHandler) List(c *gin.Context) {
	page, size := parsePagination(c)

	filter := &port.AdminLogFilter{
		Action:     c.Query("action"),
		TargetType: c.Query("target_type"),
		StartTime:  c.Query("start_time"),
		EndTime:    c.Query("end_time"),
	}
	if adminIDStr := c.Query("admin_id"); adminIDStr != "" {
		if aid, err := strconv.ParseUint(adminIDStr, 10, 64); err == nil {
			filter.AdminID = &aid
		}
	}

	logs, total, err := h.adminUC.ListAuditLogs(c.Request.Context(), filter, page, size)
	if err != nil {
		h.logger.Error("获取审计日志失败", zap.Error(err))
		response.InternalError(c, "获取审计日志失败")
		return
	}

	response.SuccessWithPagination(c, logs, total, page, size)
}
