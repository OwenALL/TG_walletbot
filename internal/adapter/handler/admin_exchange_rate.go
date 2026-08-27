package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/app"
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminExchangeRateHandler 汇率管理处理器
type AdminExchangeRateHandler struct {
	adminUC *app.AdminUseCase
	logger  *zap.Logger
}

// NewAdminExchangeRateHandler 创建汇率管理处理器
func NewAdminExchangeRateHandler(adminUC *app.AdminUseCase, logger *zap.Logger) *AdminExchangeRateHandler {
	return &AdminExchangeRateHandler{
		adminUC: adminUC,
		logger:  logger,
	}
}

// List 获取所有汇率配置
func (h *AdminExchangeRateHandler) List(c *gin.Context) {
	rates, err := h.adminUC.GetAllExchangeRates(c.Request.Context())
	if err != nil {
		h.logger.Error("获取汇率列表失败", zap.Error(err))
		response.InternalError(c, "获取汇率列表失败")
		return
	}

	response.Success(c, rates)
}

// UpdateExchangeRateReq 更新汇率请求体
type UpdateExchangeRateReq struct {
	FromCurrency string `json:"from_currency" binding:"required,min=1,max=10"`
	ToCurrency   string `json:"to_currency" binding:"required,min=1,max=10"`
	Rate         string `json:"rate" binding:"required"`
	Spread       string `json:"spread"`
	MinAmount    string `json:"min_amount"`
	MaxAmount    string `json:"max_amount"`
	Enabled      *bool  `json:"enabled"`
}

// Update 更新汇率
func (h *AdminExchangeRateHandler) Update(c *gin.Context) {
	var req UpdateExchangeRateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请填写完整的汇率信息")
		return
	}

	rate, err := decimal.NewFromString(req.Rate)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "汇率必须为正数")
		return
	}

	exchangeRate := &entity.ExchangeRate{
		FromCurrency: req.FromCurrency,
		ToCurrency:   req.ToCurrency,
		Rate:         rate,
	}

	// 可选字段
	if req.Spread != "" {
		spread, err := decimal.NewFromString(req.Spread)
		if err == nil {
			exchangeRate.Spread = spread
		}
	}
	if req.MinAmount != "" {
		minAmt, err := decimal.NewFromString(req.MinAmount)
		if err == nil {
			exchangeRate.MinAmount = minAmt
		}
	}
	if req.MaxAmount != "" {
		maxAmt, err := decimal.NewFromString(req.MaxAmount)
		if err == nil {
			exchangeRate.MaxAmount = maxAmt
		}
	}
	if req.Enabled != nil {
		exchangeRate.Enabled = *req.Enabled
	} else {
		exchangeRate.Enabled = true
	}

	if err := h.adminUC.UpdateExchangeRate(c.Request.Context(), exchangeRate); err != nil {
		h.logger.Error("更新汇率失败",
			zap.String("from", req.FromCurrency),
			zap.String("to", req.ToCurrency),
			zap.Error(err),
		)
		response.InternalError(c, "更新汇率失败")
		return
	}

	// 审计日志
	adminID := getAdminID(c)
	detail := req.FromCurrency + "/" + req.ToCurrency + " -> " + req.Rate
	go h.adminUC.CreateAuditLog(c.Request.Context(), adminID, "update_exchange_rate", "exchange_rate", nil, detail, c.ClientIP())

	response.Success(c, nil)
}
