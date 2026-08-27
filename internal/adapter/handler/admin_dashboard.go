package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/app"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewDashboardHandler 创建仪表盘处理器
func NewDashboardHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// Stats 获取仪表盘统计数据
func (h *DashboardHandler) Stats(c *gin.Context) {
	stats, err := h.adminUC.GetDashboardStats(c.Request.Context())
	if err != nil {
		h.logger.Error("获取仪表盘统计失败", zap.Error(err))
		response.InternalError(c, "获取统计数据失败")
		return
	}

	response.Success(c, stats)
}

// Trend 获取交易趋势数据
func (h *DashboardHandler) Trend(c *gin.Context) {
	days := 7 // 默认 7 天
	if d, err := strconv.Atoi(c.Query("days")); err == nil && d > 0 && d <= 90 {
		days = d
	}

	trend, err := h.adminUC.GetDashboardTrend(c.Request.Context(), days)
	if err != nil {
		h.logger.Error("获取交易趋势失败", zap.Error(err), zap.Int("days", days))
		response.InternalError(c, "获取趋势数据失败")
		return
	}

	response.Success(c, trend)
}
