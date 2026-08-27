package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/app"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminWithdrawalHandler 提币审核处理器
type AdminWithdrawalHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminWithdrawalHandler 创建提币审核处理器
func NewAdminWithdrawalHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminWithdrawalHandler {
	return &AdminWithdrawalHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取提币记录列表 (支持状态筛选)
func (h *AdminWithdrawalHandler) List(c *gin.Context) {
	page, size := parsePagination(c)

	filter := &app.WithdrawalListFilter{}
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			filter.Status = &s
		}
	}

	list, total, err := h.adminUC.ListWithdrawals(c.Request.Context(), filter, page, size)
	if err != nil {
		h.logger.Error("获取提币列表失败", zap.Error(err))
		response.InternalError(c, "获取提币列表失败")
		return
	}

	response.SuccessWithPagination(c, list, total, page, size)
}

// ApproveReq 批准提币请求体
type ApproveReq struct {
	// 无需额外参数，reviewer 从 JWT 获取
}

// Approve 批准提币
func (h *AdminWithdrawalHandler) Approve(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的提币记录 ID")
		return
	}

	adminID := getAdminID(c)

	if err := h.adminUC.ApproveWithdrawal(c.Request.Context(), withdrawalID, adminID); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("批准提币失败",
			zap.Uint64("withdrawal_id", withdrawalID),
			zap.Error(err),
		)
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "approve_withdrawal", "withdrawal", &withdrawalID, "", c.ClientIP())

	response.Success(c, nil)
}

// CompleteReq 手动完成提币请求体
type CompleteReq struct {
	TxHash string `json:"tx_hash" binding:"required,min=1,max=128"`
}

// Complete 手动完成提币 (管理员回填交易哈希)
func (h *AdminWithdrawalHandler) Complete(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的提币记录 ID")
		return
	}

	var req CompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写交易哈希")
		return
	}

	adminID := getAdminID(c)

	if err := h.adminUC.CompleteWithdrawalManually(c.Request.Context(), withdrawalID, req.TxHash, adminID); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("手动完成提币失败",
			zap.Uint64("withdrawal_id", withdrawalID),
			zap.Error(err),
		)
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "complete_withdrawal", "withdrawal", &withdrawalID, "txHash: "+req.TxHash, c.ClientIP())

	response.Success(c, nil)
}

// RejectReq 拒绝提币请求体
type RejectReq struct {
	Reason string `json:"reason" binding:"required,min=1,max=500"`
}

// Reject 拒绝提币 (退回余额)
func (h *AdminWithdrawalHandler) Reject(c *gin.Context) {
	withdrawalID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的提币记录 ID")
		return
	}

	var req RejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写拒绝原因")
		return
	}

	adminID := getAdminID(c)

	if err := h.adminUC.RejectWithdrawal(c.Request.Context(), withdrawalID, adminID, req.Reason); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("拒绝提币失败",
			zap.Uint64("withdrawal_id", withdrawalID),
			zap.Error(err),
		)
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "reject_withdrawal", "withdrawal", &withdrawalID, req.Reason, c.ClientIP())

	response.Success(c, nil)
}
