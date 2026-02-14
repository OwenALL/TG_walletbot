package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/pkg/response"
)

// AdminFinanceHandler 余额宝管理处理器
type AdminFinanceHandler struct {
	financeRepo      port.FinanceInvestmentRepository
	systemConfigRepo port.SystemConfigRepository
	userRepo         port.UserRepository
	adminLogRepo     port.AdminLogRepository
	db               *gorm.DB
	logger           *zap.Logger
}

// NewAdminFinanceHandler 创建余额宝管理处理器
func NewAdminFinanceHandler(
	financeRepo port.FinanceInvestmentRepository,
	systemConfigRepo port.SystemConfigRepository,
	userRepo port.UserRepository,
	adminLogRepo port.AdminLogRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *AdminFinanceHandler {
	return &AdminFinanceHandler{
		financeRepo:      financeRepo,
		systemConfigRepo: systemConfigRepo,
		userRepo:         userRepo,
		adminLogRepo:     adminLogRepo,
		db:               db,
		logger:           logger,
	}
}

// financeInvestmentListItem 投资列表响应项 (附带用户信息)
type financeInvestmentListItem struct {
	*entity.FinanceInvestment
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// ListInvestments 获取投资列表 (分页, 支持 status 筛选)
func (h *AdminFinanceHandler) ListInvestments(c *gin.Context) {
	page, size := parsePagination(c)
	offset := (page - 1) * size

	db := h.db.WithContext(c.Request.Context()).Model(&entity.FinanceInvestment{})

	// 状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		switch statusStr {
		case "holding":
			db = db.Where("status = ?", entity.FinanceStatusHolding)
		case "withdrawn":
			db = db.Where("status = ?", entity.FinanceStatusWithdrawn)
		default:
			// 支持直接传数字
			if s, err := strconv.ParseInt(statusStr, 10, 8); err == nil {
				db = db.Where("status = ?", s)
			}
		}
	}

	// 查询总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		h.logger.Error("查询投资总数失败", zap.Error(err))
		response.InternalError(c, "获取投资列表失败")
		return
	}

	// 查询列表
	var investments []*entity.FinanceInvestment
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&investments).Error; err != nil {
		h.logger.Error("查询投资列表失败", zap.Error(err))
		response.InternalError(c, "获取投资列表失败")
		return
	}

	// 批量获取用户信息
	items := make([]*financeInvestmentListItem, 0, len(investments))
	userCache := make(map[uint64]*entity.User)

	for _, inv := range investments {
		item := &financeInvestmentListItem{FinanceInvestment: inv}

		user, ok := userCache[inv.UserID]
		if !ok {
			var err error
			user, err = h.userRepo.FindByID(c.Request.Context(), inv.UserID)
			if err != nil {
				h.logger.Warn("查询投资用户失败", zap.Uint64("user_id", inv.UserID), zap.Error(err))
			}
			userCache[inv.UserID] = user
		}
		if user != nil {
			item.Username = user.Username
			item.DisplayName = user.DisplayName()
		}

		items = append(items, item)
	}

	response.SuccessWithPagination(c, items, total, page, size)
}

// updateRateReq 调整利率请求体
type updateRateReq struct {
	AnnualRate float64 `json:"annual_rate" binding:"required,gt=0,lte=100"`
}

// UpdateRate 调整年化利率 (SuperAdmin)
func (h *AdminFinanceHandler) UpdateRate(c *gin.Context) {
	var req updateRateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入有效的年化利率 (0-100)")
		return
	}

	ctx := c.Request.Context()

	// 获取旧利率用于审计日志
	oldRate, _ := h.systemConfigRepo.Get(ctx, "finance_annual_rate")

	// 更新系统配置
	newRate := fmt.Sprintf("%.4f", req.AnnualRate)
	if err := h.systemConfigRepo.Set(ctx, "finance_annual_rate", newRate); err != nil {
		h.logger.Error("更新年化利率失败", zap.Error(err))
		response.InternalError(c, "更新利率失败")
		return
	}

	// 记录审计日志
	adminID := getAdminID(c)
	detail := fmt.Sprintf("年化利率: %s -> %s", oldRate, newRate)
	go func() {
		log := &entity.AdminLog{
			AdminID:    adminID,
			Action:     "update_finance_rate",
			TargetType: "system_config",
			Detail:     detail,
			IPAddress:  c.ClientIP(),
		}
		if err := h.adminLogRepo.Create(ctx, log); err != nil {
			h.logger.Error("记录审计日志失败", zap.Error(err))
		}
	}()

	h.logger.Info("年化利率已更新",
		zap.String("old_rate", oldRate),
		zap.String("new_rate", newRate),
		zap.Uint64("admin_id", adminID),
	)

	response.Success(c, gin.H{"annual_rate": newRate})
}

// financeStatsResp 投资统计响应
type financeStatsResp struct {
	TotalInvestAmount  decimal.Decimal `json:"total_invest_amount"`  // 总投资金额 (持有中)
	ActiveInvestCount  int64           `json:"active_invest_count"`  // 活跃投资数
	TotalProfitPaid    decimal.Decimal `json:"total_profit_paid"`    // 累计派息金额
	CurrentAnnualRate  string          `json:"current_annual_rate"`  // 当前年化利率
}

// Stats 投资统计
func (h *AdminFinanceHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()
	stats := &financeStatsResp{}

	// 活跃投资数 (持有中)
	if err := h.db.WithContext(ctx).Model(&entity.FinanceInvestment{}).
		Where("status = ?", entity.FinanceStatusHolding).
		Count(&stats.ActiveInvestCount).Error; err != nil {
		h.logger.Error("查询活跃投资数失败", zap.Error(err))
		response.InternalError(c, "获取统计数据失败")
		return
	}

	// 总投资金额 (持有中)
	var totalAmount decimal.NullDecimal
	h.db.WithContext(ctx).Model(&entity.FinanceInvestment{}).
		Where("status = ?", entity.FinanceStatusHolding).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalAmount)
	if totalAmount.Valid {
		stats.TotalInvestAmount = totalAmount.Decimal
	}

	// 累计派息金额 (所有投资记录的 total_profit 之和)
	var totalProfit decimal.NullDecimal
	h.db.WithContext(ctx).Model(&entity.FinanceInvestment{}).
		Select("COALESCE(SUM(total_profit), 0)").
		Scan(&totalProfit)
	if totalProfit.Valid {
		stats.TotalProfitPaid = totalProfit.Decimal
	}

	// 当前年化利率
	rate, err := h.systemConfigRepo.Get(ctx, "finance_annual_rate")
	if err != nil {
		h.logger.Warn("获取年化利率配置失败", zap.Error(err))
		rate = "0"
	}
	stats.CurrentAnnualRate = rate

	response.Success(c, stats)
}
