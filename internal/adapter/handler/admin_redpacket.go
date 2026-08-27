package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminRedPacketHandler 红包管理处理器
type AdminRedPacketHandler struct {
	redPacketRepo      port.RedPacketRepository
	redPacketClaimRepo port.RedPacketClaimRepository
	userRepo           port.UserRepository
	db                 *gorm.DB
	logger             *zap.Logger
}

// NewAdminRedPacketHandler 创建红包管理处理器
func NewAdminRedPacketHandler(
	redPacketRepo port.RedPacketRepository,
	redPacketClaimRepo port.RedPacketClaimRepository,
	userRepo port.UserRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *AdminRedPacketHandler {
	return &AdminRedPacketHandler{
		redPacketRepo:      redPacketRepo,
		redPacketClaimRepo: redPacketClaimRepo,
		userRepo:           userRepo,
		db:                 db,
		logger:             logger,
	}
}

// redPacketListItem 红包列表响应项 (附带发送者信息)
type redPacketListItem struct {
	*entity.RedPacket
	SenderUsername string `json:"sender_username"`
	SenderName     string `json:"sender_name"`
}

// List 获取红包列表 (支持 status 筛选)
func (h *AdminRedPacketHandler) List(c *gin.Context) {
	page, size := parsePagination(c)
	offset := (page - 1) * size

	db := h.db.WithContext(c.Request.Context()).Model(&entity.RedPacket{})

	// 状态筛选
	if statusStr := c.Query("status"); statusStr != "" {
		db = db.Where("status = ?", statusStr)
	}

	// 查询总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		h.logger.Error("查询红包总数失败", zap.Error(err))
		response.InternalError(c, "获取红包列表失败")
		return
	}

	// 查询列表
	var packets []*entity.RedPacket
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&packets).Error; err != nil {
		h.logger.Error("查询红包列表失败", zap.Error(err))
		response.InternalError(c, "获取红包列表失败")
		return
	}

	// 批量获取发送者用户信息
	items := make([]*redPacketListItem, 0, len(packets))
	userCache := make(map[uint64]*entity.User)

	for _, p := range packets {
		item := &redPacketListItem{RedPacket: p}

		user, ok := userCache[p.UserID]
		if !ok {
			var err error
			user, err = h.userRepo.FindByID(c.Request.Context(), p.UserID)
			if err != nil {
				h.logger.Warn("查询红包发送者失败", zap.Uint64("user_id", p.UserID), zap.Error(err))
			}
			userCache[p.UserID] = user
		}
		if user != nil {
			item.SenderUsername = user.Username
			item.SenderName = user.DisplayName()
		}

		items = append(items, item)
	}

	response.SuccessWithPagination(c, items, total, page, size)
}

// redPacketDetailResp 红包详情响应 (含领取记录)
type redPacketDetailResp struct {
	*entity.RedPacket
	SenderUsername string              `json:"sender_username"`
	SenderName     string              `json:"sender_name"`
	Claims         []*redPacketClaimVO `json:"claims"`
}

// redPacketClaimVO 红包领取记录视图对象
type redPacketClaimVO struct {
	*entity.RedPacketClaim
	ClaimerUsername string `json:"claimer_username"`
	ClaimerName     string `json:"claimer_name"`
}

// Detail 获取红包详情 (含领取记录)
func (h *AdminRedPacketHandler) Detail(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的红包 ID")
		return
	}

	// 查询红包
	packet, err := h.redPacketRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询红包详情失败", zap.Uint64("id", id), zap.Error(err))
		response.InternalError(c, "获取红包详情失败")
		return
	}
	if packet == nil {
		response.NotFound(c, "红包不存在")
		return
	}

	resp := &redPacketDetailResp{RedPacket: packet}

	// 查询发送者信息
	sender, err := h.userRepo.FindByID(c.Request.Context(), packet.UserID)
	if err != nil {
		h.logger.Warn("查询红包发送者失败", zap.Uint64("user_id", packet.UserID), zap.Error(err))
	}
	if sender != nil {
		resp.SenderUsername = sender.Username
		resp.SenderName = sender.DisplayName()
	}

	// 查询领取记录
	claims, err := h.redPacketClaimRepo.ListByPacketID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("查询红包领取记录失败", zap.Uint64("packet_id", id), zap.Error(err))
		response.InternalError(c, "获取领取记录失败")
		return
	}

	// 填充领取者信息
	claimVOs := make([]*redPacketClaimVO, 0, len(claims))
	userCache := make(map[uint64]*entity.User)
	for _, claim := range claims {
		vo := &redPacketClaimVO{RedPacketClaim: claim}

		user, ok := userCache[claim.UserID]
		if !ok {
			user, _ = h.userRepo.FindByID(c.Request.Context(), claim.UserID)
			userCache[claim.UserID] = user
		}
		if user != nil {
			vo.ClaimerUsername = user.Username
			vo.ClaimerName = user.DisplayName()
		}

		claimVOs = append(claimVOs, vo)
	}
	resp.Claims = claimVOs

	response.Success(c, resp)
}

// redPacketStatsResp 红包统计响应
type redPacketStatsResp struct {
	TotalCount    int64           `json:"total_count"`    // 总发送数量
	TotalAmount   decimal.Decimal `json:"total_amount"`   // 总金额
	ClaimedCount  int64           `json:"claimed_count"`  // 已领完数量
	ExpiredCount  int64           `json:"expired_count"`  // 已过期数量
	PendingCount  int64           `json:"pending_count"`  // 待发送数量
	SentCount     int64           `json:"sent_count"`     // 已发送 (进行中) 数量
}

// Stats 红包统计
func (h *AdminRedPacketHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()
	stats := &redPacketStatsResp{}

	// 总红包数量
	if err := h.db.WithContext(ctx).Model(&entity.RedPacket{}).Count(&stats.TotalCount).Error; err != nil {
		h.logger.Error("查询红包总数失败", zap.Error(err))
		response.InternalError(c, "获取统计数据失败")
		return
	}

	// 总金额
	var totalAmount decimal.NullDecimal
	h.db.WithContext(ctx).Model(&entity.RedPacket{}).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalAmount)
	if totalAmount.Valid {
		stats.TotalAmount = totalAmount.Decimal
	}

	// 各状态数量
	type statusCount struct {
		Status int8
		Count  int64
	}
	var statusCounts []statusCount
	h.db.WithContext(ctx).Model(&entity.RedPacket{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts)

	for _, sc := range statusCounts {
		switch sc.Status {
		case entity.RedPacketStatusPending:
			stats.PendingCount = sc.Count
		case entity.RedPacketStatusSent:
			stats.SentCount = sc.Count
		case entity.RedPacketStatusClaimed:
			stats.ClaimedCount = sc.Count
		case entity.RedPacketStatusExpired:
			stats.ExpiredCount = sc.Count
		}
	}

	response.Success(c, stats)
}
