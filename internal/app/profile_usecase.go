package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/service"
)

// ProfileUseCase 个人中心用例层
type ProfileUseCase struct {
	profileSvc *service.ProfileService
	userSvc    *service.UserService
	logger     *zap.Logger
}

// NewProfileUseCase 创建个人中心用例
func NewProfileUseCase(profileSvc *service.ProfileService, userSvc *service.UserService, logger *zap.Logger) *ProfileUseCase {
	return &ProfileUseCase{
		profileSvc: profileSvc,
		userSvc:    userSvc,
		logger:     logger,
	}
}

// GetProfileDashboard 获取个人中心主页数据
func (uc *ProfileUseCase) GetProfileDashboard(ctx context.Context, userID uint64) (*service.ProfileInfo, error) {
	return uc.profileSvc.GetProfileInfo(ctx, userID)
}

// GetBills 获取账单列表
func (uc *ProfileUseCase) GetBills(ctx context.Context, userID uint64, currency string, page int) (*service.BillsPage, error) {
	return uc.profileSvc.GetBills(ctx, userID, currency, page, 10)
}

// GetWithdrawalHistory 获取提币历史
func (uc *ProfileUseCase) GetWithdrawalHistory(ctx context.Context, userID uint64, page int) (*service.WithdrawalHistoryPage, error) {
	return uc.profileSvc.GetWithdrawalHistory(ctx, userID, page, 10)
}

// GetSmallFreeInfo 获取小额免密额度
func (uc *ProfileUseCase) GetSmallFreeInfo(ctx context.Context, userID uint64) (*service.SmallFreeInfo, error) {
	return uc.profileSvc.GetSmallFreeInfo(ctx, userID)
}

// UpdateLanguage 更新用户语言设置
func (uc *ProfileUseCase) UpdateLanguage(ctx context.Context, userID uint64, language string) error {
	err := uc.profileSvc.UpdateLanguage(ctx, userID, language)
	if err != nil {
		return err
	}
	uc.logger.Info("用户更新语言设置",
		zap.Uint64("user_id", userID),
		zap.String("language", language),
	)
	return nil
}

// UpdateSmallFreeLimit 更新小额免密额度 (需要先验证 PIN)
func (uc *ProfileUseCase) UpdateSmallFreeLimit(ctx context.Context, userID uint64, currency string, newLimit decimal.Decimal) error {
	err := uc.profileSvc.UpdateSmallFreeLimit(ctx, userID, currency, newLimit)
	if err != nil {
		return err
	}
	uc.logger.Info("用户更新小额免密额度",
		zap.Uint64("user_id", userID),
		zap.String("currency", currency),
		zap.String("new_limit", newLimit.String()),
	)
	return nil
}
