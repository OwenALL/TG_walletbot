package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/middleware"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// AuthHandler 管理员认证处理器
type AuthHandler struct {
	adminUC   *app.AdminUseCase
	jwtSecret string
	jwtExpire int
	logger    *zap.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(adminUC *app.AdminUseCase, jwtSecret string, jwtExpire int, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		adminUC:   adminUC,
		jwtSecret: jwtSecret,
		jwtExpire: jwtExpire,
		logger:    logger,
	}
}

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 管理员登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入用户名和密码")
		return
	}

	admin, err := h.adminUC.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		response.InternalError(c, "登录失败")
		return
	}

	// 生成 JWT Token
	token, err := middleware.GenerateToken(admin.ID, admin.Username, admin.Role, h.jwtSecret, h.jwtExpire)
	if err != nil {
		h.logger.Error("生成 Token 失败", zap.Error(err))
		response.InternalError(c, "登录失败")
		return
	}

	// 记录审计日志
	go h.adminUC.CreateAuditLog(c.Request.Context(), admin.ID, "login", "admin", &admin.ID, "", c.ClientIP())

	response.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       admin.ID,
			"username": admin.Username,
			"role":     admin.Role,
		},
	})
}

// Me 获取当前管理员信息
func (h *AuthHandler) Me(c *gin.Context) {
	adminID := getAdminID(c)
	admin, err := h.adminUC.GetAdminByID(c.Request.Context(), adminID)
	if err != nil || admin == nil {
		response.Unauthorized(c, "管理员不存在")
		return
	}

	response.Success(c, gin.H{
		"id":            admin.ID,
		"username":      admin.Username,
		"role":          admin.Role,
		"last_login_at": admin.LastLoginAt,
		"created_at":    admin.CreatedAt,
	})
}

// ChangePasswordReq 修改密码请求
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改当前管理员密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入旧密码和新密码（新密码至少 6 位）")
		return
	}

	adminID := getAdminID(c)
	if adminID == 0 {
		response.Unauthorized(c, "缺少认证信息")
		return
	}

	err := h.adminUC.ChangePassword(c.Request.Context(), adminID, req.OldPassword, req.NewPassword)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok {
			response.ErrorFromAppError(c, appErr)
			return
		}
		h.logger.Error("修改密码失败", zap.Uint64("admin_id", adminID), zap.Error(err))
		response.InternalError(c, "修改密码失败")
		return
	}

	// 记录审计日志
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "change_password", "admin", &adminID, "修改密码", c.ClientIP())

	response.Success(c, nil)
}

// getAdminID 从 context 获取管理员 ID
func getAdminID(c *gin.Context) uint64 {
	v, exists := c.Get("admin_id")
	if !exists {
		return 0
	}
	id, ok := v.(uint64)
	if !ok {
		return 0
	}
	return id
}

// getAdminRole 从 context 获取管理员角色
func getAdminRole(c *gin.Context) string {
	v, exists := c.Get("admin_role")
	if !exists {
		return ""
	}
	role, ok := v.(string)
	if !ok {
		return ""
	}
	return role
}
