package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/app"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminConfigHandler 系统配置处理器
type AdminConfigHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminConfigHandler 创建系统配置处理器
func NewAdminConfigHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminConfigHandler {
	return &AdminConfigHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取所有系统配置
func (h *AdminConfigHandler) List(c *gin.Context) {
	configs, err := h.adminUC.GetAllConfigs(c.Request.Context())
	if err != nil {
		h.logger.Error("获取系统配置失败", zap.Error(err))
		response.InternalError(c, "获取系统配置失败")
		return
	}

	response.Success(c, configs)
}

// UpdateConfigReq 更新配置请求体
type UpdateConfigReq struct {
	Key   string `json:"key" binding:"required,min=1,max=64"`
	Value string `json:"value" binding:"required"`
}

// Update 更新系统配置
func (h *AdminConfigHandler) Update(c *gin.Context) {
	var req UpdateConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写配置项和值")
		return
	}

	if err := h.adminUC.UpdateConfig(c.Request.Context(), req.Key, req.Value); err != nil {
		h.logger.Error("更新系统配置失败",
			zap.String("key", req.Key),
			zap.Error(err),
		)
		response.InternalError(c, "更新配置失败")
		return
	}

	// 审计日志
	adminID := getAdminID(c)
	detail := req.Key + " = " + req.Value
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "update_config", "system_config", nil, detail, c.ClientIP())

	response.Success(c, nil)
}
