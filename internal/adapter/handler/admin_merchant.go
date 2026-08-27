package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminMerchantHandler 商户管理处理器
type AdminMerchantHandler struct {
	merchantRepo port.MerchantRepository
	userRepo     port.UserRepository
	adminLogRepo port.AdminLogRepository
	db           *gorm.DB
	logger       *zap.Logger
}

// NewAdminMerchantHandler 创建商户管理处理器
func NewAdminMerchantHandler(
	merchantRepo port.MerchantRepository,
	userRepo port.UserRepository,
	adminLogRepo port.AdminLogRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *AdminMerchantHandler {
	return &AdminMerchantHandler{
		merchantRepo: merchantRepo,
		userRepo:     userRepo,
		adminLogRepo: adminLogRepo,
		db:           db,
		logger:       logger,
	}
}

// merchantListItem 商户列表响应项 (附带用户信息)
type merchantListItem struct {
	*entity.Merchant
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// List 获取商户列表 (分页, 支持 status 筛选)
func (h *AdminMerchantHandler) List(c *gin.Context) {
	page, size := parsePagination(c)
	offset := (page - 1) * size

	db := h.db.WithContext(c.Request.Context()).Model(&entity.Merchant{})

	// 状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		if s, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
			db = db.Where("status = ?", s)
		}
	}

	// 查询总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		h.logger.Error("查询商户总数失败", zap.Error(err))
		response.InternalError(c, "获取商户列表失败")
		return
	}

	// 查询列表
	var merchants []*entity.Merchant
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&merchants).Error; err != nil {
		h.logger.Error("查询商户列表失败", zap.Error(err))
		response.InternalError(c, "获取商户列表失败")
		return
	}

	// 批量获取用户信息
	items := make([]*merchantListItem, 0, len(merchants))
	userCache := make(map[uint64]*entity.User)

	for _, m := range merchants {
		item := &merchantListItem{Merchant: m}

		user, ok := userCache[m.UserID]
		if !ok {
			var err error
			user, err = h.userRepo.FindByID(c.Request.Context(), m.UserID)
			if err != nil {
				h.logger.Warn("查询商户用户失败", zap.Uint64("user_id", m.UserID), zap.Error(err))
			}
			userCache[m.UserID] = user
		}
		if user != nil {
			item.Username = user.Username
			item.DisplayName = user.DisplayName()
		}

		items = append(items, item)
	}

	response.SuccessWithPagination(c, items, total, page, size)
}

// ToggleStatus 切换商户启用/禁用状态 (Active <-> Disabled)
func (h *AdminMerchantHandler) ToggleStatus(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的商户 ID")
		return
	}

	ctx := c.Request.Context()

	merchant, err := h.merchantRepo.FindByID(ctx, id)
	if err != nil {
		h.logger.Error("查询商户失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}
	if merchant == nil {
		response.NotFound(c, "商户不存在")
		return
	}

	// 切换状态: Active -> Disabled, 其他 -> Active
	var newStatus int8
	var actionDesc string
	if merchant.IsActive() {
		newStatus = entity.MerchantStatusDisabled
		actionDesc = "禁用"
	} else {
		newStatus = entity.MerchantStatusActive
		actionDesc = "启用"
	}

	merchant.Status = newStatus
	merchant.UpdatedAt = time.Now()

	if err := h.merchantRepo.Update(ctx, merchant); err != nil {
		h.logger.Error("切换商户状态失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}

	// 记录审计日志
	adminID := getAdminID(c)
	go func() {
		log := &entity.AdminLog{
			AdminID:    adminID,
			Action:     "toggle_merchant_status",
			TargetType: "merchant",
			TargetID:   &id,
			Detail:     fmt.Sprintf("%s商户 %s (ID:%d)，状态变更为: %s", actionDesc, merchant.BusinessName, id, merchant.StatusText()),
			IPAddress:  c.ClientIP(),
		}
		if err := h.adminLogRepo.Create(ctx, log); err != nil {
			h.logger.Error("记录审计日志失败", zap.Error(err))
		}
	}()

	h.logger.Info("切换商户状态",
		zap.Uint64("merchant_id", id),
		zap.String("business_name", merchant.BusinessName),
		zap.String("action", actionDesc),
		zap.Int8("new_status", newStatus),
		zap.Uint64("admin_id", adminID),
	)

	response.Success(c, gin.H{
		"id":     id,
		"status": newStatus,
	})
}

// updateFeeRateReq 修改商户费率请求体
type updateFeeRateReq struct {
	FeeRate float64 `json:"fee_rate" binding:"required,gte=0,lte=100"`
}

// UpdateFeeRate 修改商户费率 (仅超级管理员)
func (h *AdminMerchantHandler) UpdateFeeRate(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的商户 ID")
		return
	}

	var req updateFeeRateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "费率参数无效，需为 0-100 之间的数值")
		return
	}

	ctx := c.Request.Context()

	merchant, err := h.merchantRepo.FindByID(ctx, id)
	if err != nil {
		h.logger.Error("查询商户失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}
	if merchant == nil {
		response.NotFound(c, "商户不存在")
		return
	}

	oldFeeRate := merchant.FeeRate
	merchant.FeeRate = decimal.NewFromFloat(req.FeeRate)
	merchant.UpdatedAt = time.Now()

	if err := h.merchantRepo.Update(ctx, merchant); err != nil {
		h.logger.Error("修改商户费率失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "操作失败")
		return
	}

	// 记录审计日志
	adminID := getAdminID(c)
	go func() {
		log := &entity.AdminLog{
			AdminID:    adminID,
			Action:     "update_merchant_fee_rate",
			TargetType: "merchant",
			TargetID:   &id,
			Detail:     fmt.Sprintf("修改商户 %s (ID:%d) 费率: %s%% -> %s%%", merchant.BusinessName, id, oldFeeRate.StringFixed(2), merchant.FeeRate.StringFixed(2)),
			IPAddress:  c.ClientIP(),
		}
		if err := h.adminLogRepo.Create(ctx, log); err != nil {
			h.logger.Error("记录审计日志失败", zap.Error(err))
		}
	}()

	h.logger.Info("修改商户费率",
		zap.Uint64("merchant_id", id),
		zap.String("business_name", merchant.BusinessName),
		zap.String("old_fee_rate", oldFeeRate.StringFixed(2)),
		zap.String("new_fee_rate", merchant.FeeRate.StringFixed(2)),
		zap.Uint64("admin_id", adminID),
	)

	response.Success(c, gin.H{
		"id":       id,
		"fee_rate": merchant.FeeRate,
	})
}
