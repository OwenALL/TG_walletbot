package service

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// RedPacketService 红包领域服务
type RedPacketService struct {
	db               *gorm.DB
	redPacketRepo    port.RedPacketRepository
	claimRepo        port.RedPacketClaimRepository
	coverRepo        port.RedPacketCoverRepository
	walletRepo       port.WalletRepository
	transactionRepo  port.TransactionRepository
	cacheRepo        port.CacheRepository
	systemConfigRepo port.SystemConfigRepository
	logger           *zap.Logger
}

// NewRedPacketService 创建红包领域服务
func NewRedPacketService(
	db *gorm.DB,
	redPacketRepo port.RedPacketRepository,
	claimRepo port.RedPacketClaimRepository,
	coverRepo port.RedPacketCoverRepository,
	walletRepo port.WalletRepository,
	transactionRepo port.TransactionRepository,
	cacheRepo port.CacheRepository,
	systemConfigRepo port.SystemConfigRepository,
	logger *zap.Logger,
) *RedPacketService {
	return &RedPacketService{
		db:               db,
		redPacketRepo:    redPacketRepo,
		claimRepo:        claimRepo,
		coverRepo:        coverRepo,
		walletRepo:       walletRepo,
		transactionRepo:  transactionRepo,
		cacheRepo:        cacheRepo,
		systemConfigRepo: systemConfigRepo,
		logger:           logger,
	}
}

// ClaimResult 领取红包结果
type ClaimResult struct {
	Claim      *entity.RedPacketClaim
	Packet     *entity.RedPacket
	IsFinished bool // 红包是否已领完
}

// CreateRedPacket 创建红包并冻结发送者余额
func (s *RedPacketService) CreateRedPacket(ctx context.Context, userID uint64, currency string, totalAmount decimal.Decimal, totalCount int, packetType int8, coverFileID string) (*entity.RedPacket, error) {
	// 参数校验
	if totalAmount.LessThanOrEqual(decimal.Zero) {
		return nil, apperrors.NewBadRequest("红包金额必须大于 0")
	}
	if totalCount <= 0 {
		return nil, apperrors.NewBadRequest("红包个数必须大于 0")
	}
	if totalCount > 100 {
		return nil, apperrors.NewBadRequest("红包个数最多 100 个")
	}

	// 均分红包检查: 确保每个人至少能分到 0.01
	if packetType == entity.RedPacketTypeEqual {
		perAmount := totalAmount.Div(decimal.NewFromInt(int64(totalCount)))
		if perAmount.LessThan(decimal.NewFromFloat(0.01)) {
			return nil, apperrors.NewBadRequest("每个红包金额不能低于 0.01")
		}
	}

	// 拼手气红包检查: 确保金额足够分配
	if packetType == entity.RedPacketTypeRandom {
		minTotal := decimal.NewFromFloat(0.01).Mul(decimal.NewFromInt(int64(totalCount)))
		if totalAmount.LessThan(minTotal) {
			return nil, apperrors.NewBadRequest("红包金额不足以分配给所有人")
		}
	}

	// 获取过期时间配置
	expireHours := s.getExpireHours(ctx)

	var packet *entity.RedPacket

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)

		// 1. 查询并锁定发送者钱包
		wallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, currency)
		if err != nil {
			return apperrors.Wrap(err, "查询钱包失败")
		}
		if wallet == nil {
			return apperrors.NewNotFound("钱包", currency)
		}

		// 2. 验证余额
		if !wallet.HasSufficientBalance(totalAmount) {
			return apperrors.NewInsufficientBalance(currency)
		}

		// 3. 扣减余额 (冻结到红包)
		balanceBefore := wallet.Balance
		balanceAfter := balanceBefore.Sub(totalAmount)
		if err := walletRepo.UpdateBalance(ctx, wallet.ID, totalAmount.Neg()); err != nil {
			return apperrors.Wrap(err, "扣减余额失败")
		}

		// 4. 创建红包记录
		now := time.Now()
		packet = &entity.RedPacket{
			UserID:        userID,
			Currency:      currency,
			TotalAmount:   totalAmount,
			TotalCount:    totalCount,
			ClaimedCount:  0,
			ClaimedAmount: decimal.Zero,
			Type:          packetType,
			CoverFileID:   coverFileID,
			Status:        entity.RedPacketStatusPending,
			ExpireAt:      now.Add(time.Duration(expireHours) * time.Hour),
			CreatedAt:     now,
		}
		if err := tx.WithContext(ctx).Create(packet).Error; err != nil {
			return apperrors.Wrap(err, "创建红包记录失败")
		}

		// 5. 创建交易流水 (发红包)
		txRepo := newTxTransactionRepo(tx)
		transaction := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeRedpacketSend,
			Currency:      currency,
			Amount:        totalAmount,
			Fee:           decimal.Zero,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedID:     &packet.ID,
			RelatedType:   "red_packet",
			Memo:          fmt.Sprintf("发红包 #%d", packet.ID),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建交易流水失败")
		}

		return nil
	})

	if err != nil {
		s.logger.Error("创建红包失败",
			zap.Uint64("user_id", userID),
			zap.String("currency", currency),
			zap.String("amount", totalAmount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("红包创建成功",
		zap.Uint64("user_id", userID),
		zap.Uint64("packet_id", packet.ID),
		zap.String("currency", currency),
		zap.String("amount", totalAmount.String()),
		zap.Int("count", totalCount),
	)

	return packet, nil
}

// ClaimRedPacket 领取红包 (Redis 分布式锁保护)
func (s *RedPacketService) ClaimRedPacket(ctx context.Context, packetID, userID uint64) (*ClaimResult, error) {
	// 1. Redis SetNX 加锁，防止并发领取
	lockKey := fmt.Sprintf("redpacket:lock:%d", packetID)
	locked, err := s.cacheRepo.SetNX(ctx, lockKey, "1", 5*time.Second)
	if err != nil {
		s.logger.Error("红包加锁失败", zap.Uint64("packet_id", packetID), zap.Error(err))
		return nil, apperrors.NewInternal("系统繁忙，请稍后重试", err)
	}
	if !locked {
		return nil, apperrors.NewBadRequest("红包领取中，请稍后重试")
	}
	defer func() {
		_ = s.cacheRepo.Delete(ctx, lockKey)
	}()

	// 2. 查询红包
	packet, err := s.redPacketRepo.FindByID(ctx, packetID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询红包失败")
	}
	if packet == nil {
		return nil, apperrors.NewNotFound("红包", packetID)
	}

	// 3. 状态检查
	// 允许 Pending 和 Sent 状态领取:
	// - Pending: 通过 inline 模式发送到群聊时，状态可能尚未转为 Sent
	//   (callback 按钮只存在于已发到群聊的消息上，能触发 claim 说明红包已实际发送)
	// - Sent: 正常已发送状态
	if packet.Status != entity.RedPacketStatusSent && packet.Status != entity.RedPacketStatusPending {
		return nil, apperrors.NewBadRequest("红包不可领取")
	}
	if packet.IsExpired() {
		return nil, apperrors.NewBadRequest("红包已过期")
	}
	if packet.IsFullyClaimed() {
		return nil, apperrors.NewBadRequest("红包已领完")
	}

	// 4. 检查是否已领取过
	existingClaim, err := s.claimRepo.FindByPacketAndUser(ctx, packetID, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询领取记录失败")
	}
	if existingClaim != nil {
		return nil, apperrors.NewBadRequest("你已经领取过了")
	}

	// 5. 计算领取金额
	claimAmount := s.calculateClaimAmount(packet)

	// 6. 事务执行领取
	var result ClaimResult
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 查询领取者钱包并加锁
		claimerWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, packet.Currency)
		if err != nil {
			return apperrors.Wrap(err, "查询领取者钱包失败")
		}
		if claimerWallet == nil {
			return apperrors.NewNotFound("钱包", packet.Currency)
		}

		// 创建领取记录
		now := time.Now()
		claim := &entity.RedPacketClaim{
			RedPacketID: packetID,
			UserID:      userID,
			Amount:      claimAmount,
			CreatedAt:   now,
		}
		if err := tx.WithContext(ctx).Create(claim).Error; err != nil {
			return apperrors.Wrap(err, "创建领取记录失败")
		}

		// 更新红包: claimed_count + 1, claimed_amount + claimAmount
		newClaimedCount := packet.ClaimedCount + 1
		newClaimedAmount := packet.ClaimedAmount.Add(claimAmount)
		newStatus := packet.Status
		// Pending 状态被领取时自动升级为 Sent (inline 模式发送后首次领取)
		if newStatus == entity.RedPacketStatusPending {
			newStatus = entity.RedPacketStatusSent
		}
		isFinished := newClaimedCount >= packet.TotalCount
		if isFinished {
			newStatus = entity.RedPacketStatusClaimed
		}
		if err := tx.WithContext(ctx).Model(&entity.RedPacket{}).Where("id = ?", packetID).Updates(map[string]interface{}{
			"claimed_count":  newClaimedCount,
			"claimed_amount": newClaimedAmount,
			"status":         newStatus,
		}).Error; err != nil {
			return apperrors.Wrap(err, "更新红包状态失败")
		}

		// 给领取者钱包加余额
		claimerBalanceBefore := claimerWallet.Balance
		claimerBalanceAfter := claimerBalanceBefore.Add(claimAmount)
		if err := walletRepo.UpdateBalance(ctx, claimerWallet.ID, claimAmount); err != nil {
			return apperrors.Wrap(err, "增加领取者余额失败")
		}

		// 创建交易流水 (领红包)
		transaction := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeRedpacketRecv,
			Currency:      packet.Currency,
			Amount:        claimAmount,
			Fee:           decimal.Zero,
			BalanceBefore: claimerBalanceBefore,
			BalanceAfter:  claimerBalanceAfter,
			RelatedID:     &packetID,
			RelatedType:   "red_packet",
			Memo:          fmt.Sprintf("领红包 #%d", packetID),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建交易流水失败")
		}

		// 更新内存中的 packet 状态
		packet.ClaimedCount = newClaimedCount
		packet.ClaimedAmount = newClaimedAmount
		packet.Status = newStatus

		result.Claim = claim
		result.Packet = packet
		result.IsFinished = isFinished
		return nil
	})

	if err != nil {
		s.logger.Error("领取红包失败",
			zap.Uint64("packet_id", packetID),
			zap.Uint64("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("红包领取成功",
		zap.Uint64("packet_id", packetID),
		zap.Uint64("user_id", userID),
		zap.String("amount", claimAmount.String()),
		zap.Bool("finished", result.IsFinished),
	)

	return &result, nil
}

// ExpireRedPackets 过期红包批量退款
func (s *RedPacketService) ExpireRedPackets(ctx context.Context) error {
	expiredPackets, err := s.redPacketRepo.ListExpired(ctx)
	if err != nil {
		return apperrors.Wrap(err, "查询过期红包失败")
	}

	if len(expiredPackets) == 0 {
		return nil
	}

	s.logger.Info("发现过期红包", zap.Int("count", len(expiredPackets)))

	for _, packet := range expiredPackets {
		if err := s.refundExpiredPacket(ctx, packet); err != nil {
			s.logger.Error("退款过期红包失败",
				zap.Uint64("packet_id", packet.ID),
				zap.Error(err),
			)
			continue
		}
	}

	return nil
}

// refundExpiredPacket 退款单个过期红包
func (s *RedPacketService) refundExpiredPacket(ctx context.Context, packet *entity.RedPacket) error {
	refundAmount := packet.RemainingAmount()
	if refundAmount.LessThanOrEqual(decimal.Zero) {
		// 已全部领完，仅更新状态
		packet.Status = entity.RedPacketStatusExpired
		return s.redPacketRepo.Update(ctx, packet)
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 查询发送者钱包并加锁
		senderWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, packet.UserID, packet.Currency)
		if err != nil {
			return apperrors.Wrap(err, "查询发送者钱包失败")
		}
		if senderWallet == nil {
			return apperrors.NewNotFound("钱包", packet.Currency)
		}

		// 退款给发送者
		now := time.Now()
		senderBalanceBefore := senderWallet.Balance
		senderBalanceAfter := senderBalanceBefore.Add(refundAmount)
		if err := walletRepo.UpdateBalance(ctx, senderWallet.ID, refundAmount); err != nil {
			return apperrors.Wrap(err, "退款余额失败")
		}

		// 创建退款交易流水
		transaction := &entity.Transaction{
			UserID:        packet.UserID,
			Type:          entity.TxTypeRedpacketRefund,
			Currency:      packet.Currency,
			Amount:        refundAmount,
			Fee:           decimal.Zero,
			BalanceBefore: senderBalanceBefore,
			BalanceAfter:  senderBalanceAfter,
			RelatedID:     &packet.ID,
			RelatedType:   "red_packet",
			Memo:          fmt.Sprintf("红包过期退回 #%d", packet.ID),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建退款交易流水失败")
		}

		// 更新红包状态为已过期
		if err := tx.WithContext(ctx).Model(&entity.RedPacket{}).Where("id = ?", packet.ID).
			Update("status", entity.RedPacketStatusExpired).Error; err != nil {
			return apperrors.Wrap(err, "更新红包状态失败")
		}

		return nil
	})

	if err != nil {
		return apperrors.Wrap(err, "红包过期退款事务执行失败")
	}

	s.logger.Info("红包过期退款成功",
		zap.Uint64("packet_id", packet.ID),
		zap.Uint64("user_id", packet.UserID),
		zap.String("refund_amount", refundAmount.String()),
	)

	return nil
}

// GetUserRedPackets 获取用户待发送红包列表
func (s *RedPacketService) GetUserRedPackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	status := int8(entity.RedPacketStatusPending)
	list, _, err := s.redPacketRepo.ListByUserID(ctx, userID, &status, 0, 50)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户红包失败")
	}
	return list, nil
}

// GetUserActivePackets 获取用户进行中的红包 (待发送 + 已发送)
func (s *RedPacketService) GetUserActivePackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	statuses := []int8{entity.RedPacketStatusPending, entity.RedPacketStatusSent}
	list, _, err := s.redPacketRepo.ListByUserIDAndStatuses(ctx, userID, statuses, 0, 50)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询进行中红包失败")
	}
	return list, nil
}

// GetUserFinishedPackets 获取用户已结束的红包 (已领完 + 已过期 + 已关闭)
func (s *RedPacketService) GetUserFinishedPackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	statuses := []int8{entity.RedPacketStatusClaimed, entity.RedPacketStatusExpired, entity.RedPacketStatusCancelled}
	list, _, err := s.redPacketRepo.ListByUserIDAndStatuses(ctx, userID, statuses, 0, 50)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询已结束红包失败")
	}
	return list, nil
}

// UpdateRedPacketFields 更新红包指定字段
func (s *RedPacketService) UpdateRedPacketFields(ctx context.Context, packetID uint64, fields map[string]interface{}) error {
	packet, err := s.redPacketRepo.FindByID(ctx, packetID)
	if err != nil {
		return apperrors.Wrap(err, "查询红包失败")
	}
	if packet == nil {
		return apperrors.NewNotFound("红包", packetID)
	}
	if packet.Status != entity.RedPacketStatusPending {
		return apperrors.NewBadRequest("只能修改待发送的红包")
	}
	return s.db.WithContext(ctx).Model(&entity.RedPacket{}).Where("id = ?", packetID).Updates(fields).Error
}

// CancelRedPacket 关闭红包并退款
func (s *RedPacketService) CancelRedPacket(ctx context.Context, packetID, userID uint64) error {
	packet, err := s.redPacketRepo.FindByID(ctx, packetID)
	if err != nil {
		return apperrors.Wrap(err, "查询红包失败")
	}
	if packet == nil {
		return apperrors.NewNotFound("红包", packetID)
	}
	if packet.UserID != userID {
		return apperrors.NewBadRequest("无权操作此红包")
	}
	if packet.Status != entity.RedPacketStatusPending {
		return apperrors.NewBadRequest("只能关闭待发送的红包")
	}

	refundAmount := packet.RemainingAmount()

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		walletRepo := newTxWalletRepo(tx)
		txRepo := newTxTransactionRepo(tx)

		// 退款给发送者
		senderWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, userID, packet.Currency)
		if err != nil {
			return apperrors.Wrap(err, "查询钱包失败")
		}
		if senderWallet == nil {
			return apperrors.NewNotFound("钱包", packet.Currency)
		}

		now := time.Now()
		balanceBefore := senderWallet.Balance
		balanceAfter := balanceBefore.Add(refundAmount)
		if err := walletRepo.UpdateBalance(ctx, senderWallet.ID, refundAmount); err != nil {
			return apperrors.Wrap(err, "退款余额失败")
		}

		// 创建退款流水
		transaction := &entity.Transaction{
			UserID:        userID,
			Type:          entity.TxTypeRedpacketRefund,
			Currency:      packet.Currency,
			Amount:        refundAmount,
			Fee:           decimal.Zero,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceAfter,
			RelatedID:     &packetID,
			RelatedType:   "red_packet",
			Memo:          fmt.Sprintf("关闭红包退回 #%d", packetID),
			CreatedAt:     now,
		}
		if err := txRepo.Create(ctx, transaction); err != nil {
			return apperrors.Wrap(err, "创建退款流水失败")
		}

		// 更新红包状态为已关闭
		if err := tx.WithContext(ctx).Model(&entity.RedPacket{}).Where("id = ?", packetID).
			Update("status", entity.RedPacketStatusCancelled).Error; err != nil {
			return apperrors.Wrap(err, "更新红包状态失败")
		}

		return nil
	})

	if err != nil {
		s.logger.Error("关闭红包失败",
			zap.Uint64("packet_id", packetID),
			zap.Uint64("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("红包已关闭退款",
		zap.Uint64("packet_id", packetID),
		zap.Uint64("user_id", userID),
		zap.String("refund_amount", refundAmount.String()),
	)
	return nil
}

// GetRedPacketByID 根据 ID 查询红包
func (s *RedPacketService) GetRedPacketByID(ctx context.Context, packetID uint64) (*entity.RedPacket, error) {
	return s.redPacketRepo.FindByID(ctx, packetID)
}

// UpdateSentStatus 标记红包已发送到群聊
func (s *RedPacketService) UpdateSentStatus(ctx context.Context, packetID uint64, chatID int64, messageID int) error {
	packet, err := s.redPacketRepo.FindByID(ctx, packetID)
	if err != nil {
		return apperrors.Wrap(err, "查询红包失败")
	}
	if packet == nil {
		return apperrors.NewNotFound("红包", packetID)
	}
	if packet.Status != entity.RedPacketStatusPending {
		return apperrors.NewBadRequest("红包状态不正确")
	}

	chatIDVal := chatID
	msgIDVal := int64(messageID)
	packet.ChatID = &chatIDVal
	packet.MessageID = &msgIDVal
	packet.Status = entity.RedPacketStatusSent
	return s.redPacketRepo.Update(ctx, packet)
}

// GetClaimsByPacketID 获取红包的所有领取记录
func (s *RedPacketService) GetClaimsByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
	return s.claimRepo.ListByPacketID(ctx, packetID)
}

// GetCoversByUserID 获取用户的红包封面列表
func (s *RedPacketService) GetCoversByUserID(ctx context.Context, userID uint64) ([]*entity.RedPacketCover, error) {
	return s.coverRepo.ListByUserID(ctx, userID)
}

// AddCover 添加红包封面
func (s *RedPacketService) AddCover(ctx context.Context, userID uint64, fileID, fileType string) error {
	cover := &entity.RedPacketCover{
		UserID:    userID,
		FileID:    fileID,
		FileType:  fileType,
		Status:    1,
		CreatedAt: time.Now(),
	}
	return s.coverRepo.Create(ctx, cover)
}

// calculateClaimAmount 计算领取金额
func (s *RedPacketService) calculateClaimAmount(packet *entity.RedPacket) decimal.Decimal {
	remaining := packet.RemainingAmount()
	remainingCount := packet.TotalCount - packet.ClaimedCount

	// 最后一个人拿剩余所有
	if remainingCount <= 1 {
		return remaining
	}

	switch packet.Type {
	case entity.RedPacketTypeEqual:
		// 均分: 每人 = 总额 / 总数 (最后一人拿剩余)
		return packet.TotalAmount.Div(decimal.NewFromInt(int64(packet.TotalCount))).Truncate(8)
	case entity.RedPacketTypeRandom:
		// 二倍均值法
		return s.randomAmount(remaining, remainingCount)
	default:
		// 默认均分
		return remaining.Div(decimal.NewFromInt(int64(remainingCount))).Truncate(8)
	}
}

// randomAmount 二倍均值随机算法
// 范围: [0.01, 2 * remaining / remainingCount)
func (s *RedPacketService) randomAmount(remaining decimal.Decimal, remainingCount int) decimal.Decimal {
	minAmount := decimal.NewFromFloat(0.01)

	// 二倍均值上限
	avg := remaining.Div(decimal.NewFromInt(int64(remainingCount)))
	maxAmount := avg.Mul(decimal.NewFromInt(2))

	// 确保上限不超过 remaining - (remainingCount-1)*minAmount
	// 保证后续每人至少 0.01
	maxAllowed := remaining.Sub(minAmount.Mul(decimal.NewFromInt(int64(remainingCount - 1))))
	if maxAmount.GreaterThan(maxAllowed) {
		maxAmount = maxAllowed
	}
	if maxAmount.LessThan(minAmount) {
		maxAmount = minAmount
	}

	// 生成随机金额 [minAmount, maxAmount]
	// 转为分 (精度 0.01) 来随机
	minCents := minAmount.Mul(decimal.NewFromInt(100)).IntPart()
	maxCents := maxAmount.Mul(decimal.NewFromInt(100)).IntPart()
	if maxCents <= minCents {
		return minAmount
	}

	randCents := minCents + rand.Int63n(maxCents-minCents+1)
	return decimal.NewFromInt(randCents).Div(decimal.NewFromInt(100))
}

// getExpireHours 从系统配置获取红包过期时间 (小时)
func (s *RedPacketService) getExpireHours(ctx context.Context) int {
	val, err := s.systemConfigRepo.Get(ctx, entity.ConfigRedpacketExpireHours)
	if err != nil || val == "" {
		return 24 // 默认 24 小时
	}
	hours, err := strconv.Atoi(val)
	if err != nil || hours <= 0 {
		return 24
	}
	return hours
}
