package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// AdminUserHandler 用户管理处理器
type AdminUserHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminUserHandler 创建用户管理处理器
func NewAdminUserHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminUserHandler {
	return &AdminUserHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取用户列表
func (h *AdminUserHandler) List(c *gin.Context) {
	page, size := parsePagination(c)

	filter := &app.UserListFilter{
		Search: c.Query("keyword"),
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status, err := strconv.ParseInt(statusStr, 10, 8)
		if err == nil {
			s := int8(status)
			filter.Status = &s
		}
	}

	users, total, err := h.adminUC.ListUsers(c.Request.Context(), filter, page, size)
	if err != nil {
		h.logger.Error("获取用户列表失败", zap.Error(err))
		response.InternalError(c, "获取用户列表失败")
		return
	}

	response.SuccessWithPagination(c, users, total, page, size)
}

// Detail 获取用户详情
func (h *AdminUserHandler) Detail(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}

	detail, err := h.adminUC.GetUserDetail(c.Request.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("获取用户详情失败", zap.Error(err))
		response.InternalError(c, "获取用户详情失败")
		return
	}

	response.Success(c, detail)
}

// UpdateStatus 更新用户状态
type UpdateUserStatusReq struct {
	Status int8 `json:"status" binding:"required,oneof=1 2 3"`
}

// UpdateStatus 冻结/解冻用户
func (h *AdminUserHandler) UpdateStatus(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}

	var req UpdateUserStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请指定用户状态 (1=正常, 2=冻结, 3=封禁)")
		return
	}

	if err := h.adminUC.UpdateUserStatus(c.Request.Context(), userID, req.Status); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("更新用户状态失败", zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志
	adminID := getAdminID(c)
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "update_user_status", "user", &userID, "", c.ClientIP())

	response.Success(c, nil)
}

// Transactions 获取用户交易记录
func (h *AdminUserHandler) Transactions(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}

	page, size := parsePagination(c)

	filter := &port.TransactionFilter{
		Currency: c.Query("currency"),
		Type:     c.Query("type"),
	}

	txs, total, err := h.adminUC.GetUserTransactions(c.Request.Context(), userID, filter, page, size)
	if err != nil {
		h.logger.Error("获取用户交易记录失败", zap.Error(err))
		response.InternalError(c, "获取交易记录失败")
		return
	}

	response.SuccessWithPagination(c, txs, total, page, size)
}

// ResetPIN 重置用户支付密码
func (h *AdminUserHandler) ResetPIN(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}

	if err := h.adminUC.ResetUserPIN(c.Request.Context(), userID); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("重置用户 PIN 失败", zap.Uint64("user_id", userID), zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志
	adminID := getAdminID(c)
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "reset_user_pin", "user", &userID, "重置用户支付密码", c.ClientIP())

	response.Success(c, nil)
}

// AdjustBalanceReq 余额调整请求
type AdjustBalanceReq struct {
	Currency string          `json:"currency" binding:"required"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Reason   string          `json:"reason" binding:"required,min=1,max=200"`
}

// AdjustBalance 调整用户余额
func (h *AdminUserHandler) AdjustBalance(c *gin.Context) {
	userID, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户 ID")
		return
	}

	var req AdjustBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请提供币种、调整金额和原因")
		return
	}

	// 调整金额不能为 0
	if req.Amount.IsZero() {
		response.BadRequest(c, "调整金额不能为 0")
		return
	}

	if err := h.adminUC.AdjustUserBalance(c.Request.Context(), userID, req.Currency, req.Amount, req.Reason); err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("调整用户余额失败",
			zap.Uint64("user_id", userID),
			zap.String("currency", req.Currency),
			zap.String("amount", req.Amount.String()),
			zap.Error(err),
		)
		response.InternalError(c, "操作失败")
		return
	}

	// 审计日志 (含 currency, amount, reason)
	adminID := getAdminID(c)
	detail := fmt.Sprintf("调整用户余额: %s %s, 原因: %s", req.Amount.String(), req.Currency, req.Reason)
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "adjust_user_balance", "user", &userID, detail, c.ClientIP())

	response.Success(c, nil)
}

// --- 通用辅助函数 ---

// parsePagination 解析分页参数
func parsePagination(c *gin.Context) (page, size int) {
	page = 1
	size = 20

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(c.Query("size")); err == nil && s > 0 && s <= 100 {
		size = s
	}

	return page, size
}

// parseIDParam 解析 URL 路径中的 ID 参数
func parseIDParam(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(c.Param(name), 10, 64)
}
