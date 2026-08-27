package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
)

// RedPacketUseCase 红包用例层
type RedPacketUseCase struct {
	redPacketSvc *service.RedPacketService
	walletSvc    *service.WalletService
	userSvc      *service.UserService
	logger       *zap.Logger
}

// NewRedPacketUseCase 创建红包用例
func NewRedPacketUseCase(
	redPacketSvc *service.RedPacketService,
	walletSvc *service.WalletService,
	userSvc *service.UserService,
	logger *zap.Logger,
) *RedPacketUseCase {
	return &RedPacketUseCase{
		redPacketSvc: redPacketSvc,
		walletSvc:    walletSvc,
		userSvc:      userSvc,
		logger:       logger,
	}
}

// CreateRedPacket 创建红包
func (uc *RedPacketUseCase) CreateRedPacket(ctx context.Context, userID uint64, currency string, totalAmount decimal.Decimal, totalCount int, packetType int8, coverFileID string) (*entity.RedPacket, error) {
	return uc.redPacketSvc.CreateRedPacket(ctx, userID, currency, totalAmount, totalCount, packetType, coverFileID)
}

// ClaimRedPacket 领取红包
func (uc *RedPacketUseCase) ClaimRedPacket(ctx context.Context, packetID, claimerUserID uint64) (*service.ClaimResult, error) {
	return uc.redPacketSvc.ClaimRedPacket(ctx, packetID, claimerUserID)
}

// GetUserPendingPackets 获取用户待发送的红包列表
func (uc *RedPacketUseCase) GetUserPendingPackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	return uc.redPacketSvc.GetUserRedPackets(ctx, userID)
}

// GetUserActivePackets 获取用户进行中的红包 (待发送 + 已发送)
func (uc *RedPacketUseCase) GetUserActivePackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	return uc.redPacketSvc.GetUserActivePackets(ctx, userID)
}

// GetUserFinishedPackets 获取用户已结束的红包 (已领完 + 已过期 + 已关闭)
func (uc *RedPacketUseCase) GetUserFinishedPackets(ctx context.Context, userID uint64) ([]*entity.RedPacket, error) {
	return uc.redPacketSvc.GetUserFinishedPackets(ctx, userID)
}

// UpdateRedPacketFields 更新红包指定字段
func (uc *RedPacketUseCase) UpdateRedPacketFields(ctx context.Context, packetID uint64, fields map[string]interface{}) error {
	return uc.redPacketSvc.UpdateRedPacketFields(ctx, packetID, fields)
}

// CancelRedPacket 关闭红包并退款
func (uc *RedPacketUseCase) CancelRedPacket(ctx context.Context, packetID, userID uint64) error {
	return uc.redPacketSvc.CancelRedPacket(ctx, packetID, userID)
}

// MarkAsSent 标记红包已发送到群聊
func (uc *RedPacketUseCase) MarkAsSent(ctx context.Context, packetID uint64, chatID int64, messageID int) error {
	return uc.redPacketSvc.UpdateSentStatus(ctx, packetID, chatID, messageID)
}

// GetRedPacketByID 根据 ID 查询红包
func (uc *RedPacketUseCase) GetRedPacketByID(ctx context.Context, packetID uint64) (*entity.RedPacket, error) {
	return uc.redPacketSvc.GetRedPacketByID(ctx, packetID)
}

// GetClaimsByPacketID 获取红包的所有领取记录
func (uc *RedPacketUseCase) GetClaimsByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
	return uc.redPacketSvc.GetClaimsByPacketID(ctx, packetID)
}

// GetCoversByUser 获取用户红包封面列表
func (uc *RedPacketUseCase) GetCoversByUser(ctx context.Context, userID uint64) ([]*entity.RedPacketCover, error) {
	return uc.redPacketSvc.GetCoversByUserID(ctx, userID)
}

// AddCover 添加红包封面
func (uc *RedPacketUseCase) AddCover(ctx context.Context, userID uint64, fileID, fileType string) error {
	return uc.redPacketSvc.AddCover(ctx, userID, fileID, fileType)
}

// GetBalance 获取用户指定币种余额
func (uc *RedPacketUseCase) GetBalance(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	wallet, err := uc.walletSvc.GetWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, err
	}
	return wallet.Balance, nil
}

// GetUserByID 根据 ID 查询用户
func (uc *RedPacketUseCase) GetUserByID(ctx context.Context, userID uint64) (*entity.User, error) {
	return uc.userSvc.GetByID(ctx, userID)
}

// GetUserByTelegramID 根据 Telegram ID 查询用户
func (uc *RedPacketUseCase) GetUserByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error) {
	return uc.userSvc.GetByTelegramID(ctx, telegramID)
}

// RedPacketTypeName 返回红包类型中文名
func RedPacketTypeName(t int8) string {
	switch t {
	case entity.RedPacketTypeEqual:
		return "均分红包"
	case entity.RedPacketTypeRandom:
		return "拼手气红包"
	default:
		return "未知类型"
	}
}
