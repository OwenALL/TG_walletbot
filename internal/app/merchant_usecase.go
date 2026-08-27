package app

import (
	"context"

	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
)

// MerchantUseCase 商户用例层
type MerchantUseCase struct {
	merchantSvc *service.MerchantService
	logger      *zap.Logger
}

// NewMerchantUseCase 创建商户用例
func NewMerchantUseCase(merchantSvc *service.MerchantService, logger *zap.Logger) *MerchantUseCase {
	return &MerchantUseCase{
		merchantSvc: merchantSvc,
		logger:      logger,
	}
}

// Create 一键创建商户
func (uc *MerchantUseCase) Create(ctx context.Context, userID uint64, displayName string) (*entity.Merchant, error) {
	return uc.merchantSvc.Create(ctx, userID, displayName)
}

// ListByUserID 获取用户的所有商户
func (uc *MerchantUseCase) ListByUserID(ctx context.Context, userID uint64) ([]*entity.Merchant, error) {
	return uc.merchantSvc.ListByUserID(ctx, userID)
}

// GetByID 根据 ID 获取商户详情
func (uc *MerchantUseCase) GetByID(ctx context.Context, merchantID uint64) (*entity.Merchant, error) {
	return uc.merchantSvc.GetByID(ctx, merchantID)
}

// UpdateName 更新商户名称
func (uc *MerchantUseCase) UpdateName(ctx context.Context, merchantID uint64, userID uint64, name string) (*entity.Merchant, error) {
	return uc.merchantSvc.UpdateName(ctx, merchantID, userID, name)
}

// UpdateWebhook 更新商户回调地址
func (uc *MerchantUseCase) UpdateWebhook(ctx context.Context, merchantID uint64, userID uint64, url string) (*entity.Merchant, error) {
	return uc.merchantSvc.UpdateWebhook(ctx, merchantID, userID, url)
}

// UpdateLogo 更新商户 LOGO
func (uc *MerchantUseCase) UpdateLogo(ctx context.Context, merchantID uint64, userID uint64, logoFileID string) (*entity.Merchant, error) {
	return uc.merchantSvc.UpdateLogo(ctx, merchantID, userID, logoFileID)
}

// RefreshSecret 刷新商户密钥
func (uc *MerchantUseCase) RefreshSecret(ctx context.Context, merchantID uint64, userID uint64) (*entity.Merchant, error) {
	return uc.merchantSvc.RefreshSecret(ctx, merchantID, userID)
}

// ToggleStatus 切换商户状态
func (uc *MerchantUseCase) ToggleStatus(ctx context.Context, merchantID uint64, userID uint64) (*entity.Merchant, error) {
	return uc.merchantSvc.ToggleStatus(ctx, merchantID, userID)
}

// AddIPWhitelist 添加 IP 到白名单
func (uc *MerchantUseCase) AddIPWhitelist(ctx context.Context, merchantID uint64, userID uint64, ip string) (*entity.Merchant, error) {
	return uc.merchantSvc.AddIPWhitelist(ctx, merchantID, userID, ip)
}

// RemoveIPWhitelist 从白名单移除 IP
func (uc *MerchantUseCase) RemoveIPWhitelist(ctx context.Context, merchantID uint64, userID uint64, ip string) (*entity.Merchant, error) {
	return uc.merchantSvc.RemoveIPWhitelist(ctx, merchantID, userID, ip)
}

// ApproveMerchant 审核通过商户 (管理员调用)
func (uc *MerchantUseCase) ApproveMerchant(ctx context.Context, merchantID uint64) error {
	return uc.merchantSvc.ApproveMerchant(ctx, merchantID)
}

// RejectMerchant 拒绝商户申请 (管理员调用)
func (uc *MerchantUseCase) RejectMerchant(ctx context.Context, merchantID uint64) error {
	return uc.merchantSvc.RejectMerchant(ctx, merchantID)
}

// ResetAPIKey 重置商户 API 密钥 (兼容旧接口)
func (uc *MerchantUseCase) ResetAPIKey(ctx context.Context, merchantID uint64) (*entity.Merchant, error) {
	return uc.merchantSvc.ResetAPIKey(ctx, merchantID)
}
