package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// AdminTransactionHandler 交易管理处理器
type AdminTransactionHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminTransactionHandler 创建交易管理处理器
func NewAdminTransactionHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminTransactionHandler {
	return &AdminTransactionHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取交易流水列表
func (h *AdminTransactionHandler) List(c *gin.Context) {
	page, size := parsePagination(c)

	filter := &app.TransactionListFilter{
		Type:     c.Query("type"),
		Currency: c.Query("currency"),
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			filter.UserID = &uid
		}
	}

	txs, total, err := h.adminUC.ListTransactions(c.Request.Context(), filter, page, size)
	if err != nil {
		h.logger.Error("获取交易列表失败", zap.Error(err))
		response.InternalError(c, "获取交易列表失败")
		return
	}

	response.SuccessWithPagination(c, txs, total, page, size)
}

// Deposits 获取充值记录列表
func (h *AdminTransactionHandler) Deposits(c *gin.Context) {
	page, size := parsePagination(c)

	deposits, total, err := h.adminUC.ListDeposits(c.Request.Context(), page, size)
	if err != nil {
		h.logger.Error("获取充值列表失败", zap.Error(err))
		response.InternalError(c, "获取充值列表失败")
		return
	}

	response.SuccessWithPagination(c, deposits, total, page, size)
}
