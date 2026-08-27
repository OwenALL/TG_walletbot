package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// MerchantService 商户领域服务
type MerchantService struct {
	merchantRepo port.MerchantRepository
	logger       *zap.Logger
}

// NewMerchantService 创建商户领域服务
func NewMerchantService(merchantRepo port.MerchantRepository, logger *zap.Logger) *MerchantService {
	return &MerchantService{
		merchantRepo: merchantRepo,
		logger:       logger,
	}
}

// Create 一键创建商户 (PIN 验证后调用)
// 自动生成商户名称: "{用户名}的商户" 或 "{用户名}的商户(N)"
func (s *MerchantService) Create(ctx context.Context, userID uint64, displayName string) (*entity.Merchant, error) {
	// 获取当前商户数量，用于生成默认名称
	count, err := s.merchantRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询商户数量失败")
	}

	// 生成默认商户名
	var businessName string
	if count == 0 {
		businessName = displayName + "的商户"
	} else {
		businessName = fmt.Sprintf("%s的商户(%d)", displayName, count+1)
	}

	// 生成 API Key (32字符随机hex)
	apiKey, err := generateRandomHex(16)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Key 失败")
	}

	// 生成 API Secret (64字符随机hex)
	apiSecret, err := generateRandomHex(32)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Secret 失败")
	}

	now := time.Now()
	merchant := &entity.Merchant{
		UserID:       userID,
		BusinessName: businessName,
		APIKey:       apiKey,
		APISecret:    apiSecret,
		Status:       entity.MerchantStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.merchantRepo.Create(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "创建商户记录失败")
	}

	s.logger.Info("商户创建成功",
		zap.Uint64("user_id", userID),
		zap.String("business_name", businessName),
		zap.Uint64("merchant_id", merchant.ID),
	)

	return merchant, nil
}

// ListByUserID 获取用户所有商户列表
func (s *MerchantService) ListByUserID(ctx context.Context, userID uint64) ([]*entity.Merchant, error) {
	merchants, err := s.merchantRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询商户列表失败")
	}
	return merchants, nil
}

// GetByID 根据商户 ID 获取商户信息
func (s *MerchantService) GetByID(ctx context.Context, merchantID uint64) (*entity.Merchant, error) {
	merchant, err := s.merchantRepo.FindByID(ctx, merchantID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询商户信息失败")
	}
	if merchant == nil {
		return nil, apperrors.NewNotFound("商户", merchantID)
	}
	return merchant, nil
}

// UpdateName 更新商户名称
func (s *MerchantService) UpdateName(ctx context.Context, merchantID uint64, userID uint64, name string) (*entity.Merchant, error) {
	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	merchant.BusinessName = name
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新商户名称失败")
	}

	s.logger.Info("商户名称已更新",
		zap.Uint64("merchant_id", merchantID),
		zap.String("new_name", name),
	)

	return merchant, nil
}

// UpdateWebhook 更新商户回调地址
func (s *MerchantService) UpdateWebhook(ctx context.Context, merchantID uint64, userID uint64, webhookURL string) (*entity.Merchant, error) {
	// 验证 URL 格式
	if webhookURL != "" {
		parsed, err := url.ParseRequestURI(webhookURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, apperrors.NewBadRequest("回调地址格式不正确，请输入有效的 HTTP/HTTPS 地址")
		}
	}

	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	merchant.WebhookURL = webhookURL
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新回调地址失败")
	}

	s.logger.Info("商户回调地址已更新",
		zap.Uint64("merchant_id", merchantID),
		zap.String("webhook_url", webhookURL),
	)

	return merchant, nil
}

// UpdateLogo 更新商户 LOGO
func (s *MerchantService) UpdateLogo(ctx context.Context, merchantID uint64, userID uint64, logoFileID string) (*entity.Merchant, error) {
	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	merchant.Logo = logoFileID
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新商户 LOGO 失败")
	}

	s.logger.Info("商户 LOGO 已更新",
		zap.Uint64("merchant_id", merchantID),
	)

	return merchant, nil
}

// RefreshSecret 刷新商户密钥
func (s *MerchantService) RefreshSecret(ctx context.Context, merchantID uint64, userID uint64) (*entity.Merchant, error) {
	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	// 生成新的 API Key 和 Secret
	newKey, err := generateRandomHex(16)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Key 失败")
	}
	newSecret, err := generateRandomHex(32)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Secret 失败")
	}

	merchant.APIKey = newKey
	merchant.APISecret = newSecret
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新商户密钥失败")
	}

	s.logger.Info("商户密钥已刷新",
		zap.Uint64("merchant_id", merchantID),
		zap.Uint64("user_id", userID),
	)

	return merchant, nil
}

// ToggleStatus 切换商户开关状态
func (s *MerchantService) ToggleStatus(ctx context.Context, merchantID uint64, userID uint64) (*entity.Merchant, error) {
	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	if merchant.IsActive() {
		merchant.Status = entity.MerchantStatusDisabled
	} else {
		merchant.Status = entity.MerchantStatusActive
	}
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新商户状态失败")
	}

	s.logger.Info("商户状态已切换",
		zap.Uint64("merchant_id", merchantID),
		zap.Int8("new_status", merchant.Status),
	)

	return merchant, nil
}

// AddIPWhitelist 添加 IP 到白名单
func (s *MerchantService) AddIPWhitelist(ctx context.Context, merchantID uint64, userID uint64, ip string) (*entity.Merchant, error) {
	// 验证 IP 格式
	if net.ParseIP(ip) == nil {
		return nil, apperrors.NewBadRequest("IP 地址格式不正确")
	}

	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	if !merchant.AddIP(ip) {
		return nil, apperrors.NewConflict("该 IP 已在白名单中")
	}

	merchant.UpdatedAt = time.Now()
	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新 IP 白名单失败")
	}

	s.logger.Info("商户 IP 白名单已添加",
		zap.Uint64("merchant_id", merchantID),
		zap.String("ip", ip),
	)

	return merchant, nil
}

// RemoveIPWhitelist 从白名单移除 IP
func (s *MerchantService) RemoveIPWhitelist(ctx context.Context, merchantID uint64, userID uint64, ip string) (*entity.Merchant, error) {
	merchant, err := s.getMerchantWithOwnerCheck(ctx, merchantID, userID)
	if err != nil {
		return nil, err
	}

	if !merchant.RemoveIP(ip) {
		return nil, apperrors.NewNotFound("IP", ip)
	}

	merchant.UpdatedAt = time.Now()
	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新 IP 白名单失败")
	}

	s.logger.Info("商户 IP 白名单已移除",
		zap.Uint64("merchant_id", merchantID),
		zap.String("ip", ip),
	)

	return merchant, nil
}

// --- 管理员操作 (后台管理兼容) ---

// ApproveMerchant 审核通过商户 (管理员调用)
func (s *MerchantService) ApproveMerchant(ctx context.Context, merchantID uint64) error {
	merchant, err := s.merchantRepo.FindByID(ctx, merchantID)
	if err != nil {
		return apperrors.Wrap(err, "查询商户信息失败")
	}
	if merchant == nil {
		return apperrors.NewNotFound("商户", merchantID)
	}
	if !merchant.IsPending() {
		return apperrors.NewBadRequest("仅待审核的商户可以审批")
	}

	merchant.Status = entity.MerchantStatusActive
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return apperrors.Wrap(err, "更新商户状态失败")
	}

	s.logger.Info("商户审核通过",
		zap.Uint64("merchant_id", merchantID),
		zap.Uint64("user_id", merchant.UserID),
		zap.String("business_name", merchant.BusinessName),
	)

	return nil
}

// RejectMerchant 拒绝商户申请 (管理员调用)
func (s *MerchantService) RejectMerchant(ctx context.Context, merchantID uint64) error {
	merchant, err := s.merchantRepo.FindByID(ctx, merchantID)
	if err != nil {
		return apperrors.Wrap(err, "查询商户信息失败")
	}
	if merchant == nil {
		return apperrors.NewNotFound("商户", merchantID)
	}
	if !merchant.IsPending() {
		return apperrors.NewBadRequest("仅待审核的商户可以拒绝")
	}

	merchant.Status = entity.MerchantStatusRejected
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return apperrors.Wrap(err, "更新商户状态失败")
	}

	s.logger.Info("商户申请已拒绝",
		zap.Uint64("merchant_id", merchantID),
		zap.Uint64("user_id", merchant.UserID),
	)

	return nil
}

// ResetAPIKey 重置商户 API 密钥 (管理员或用户调用，兼容旧接口)
func (s *MerchantService) ResetAPIKey(ctx context.Context, merchantID uint64) (*entity.Merchant, error) {
	merchant, err := s.merchantRepo.FindByID(ctx, merchantID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询商户信息失败")
	}
	if merchant == nil {
		return nil, apperrors.NewNotFound("商户", merchantID)
	}

	newKey, err := generateRandomHex(16)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Key 失败")
	}
	newSecret, err := generateRandomHex(32)
	if err != nil {
		return nil, apperrors.Wrap(err, "生成 API Secret 失败")
	}

	merchant.APIKey = newKey
	merchant.APISecret = newSecret
	merchant.UpdatedAt = time.Now()

	if err := s.merchantRepo.Update(ctx, merchant); err != nil {
		return nil, apperrors.Wrap(err, "更新商户密钥失败")
	}

	s.logger.Info("商户密钥已重置",
		zap.Uint64("merchant_id", merchantID),
		zap.Uint64("user_id", merchant.UserID),
	)

	return merchant, nil
}

// getMerchantWithOwnerCheck 获取商户并验证所有权
func (s *MerchantService) getMerchantWithOwnerCheck(ctx context.Context, merchantID uint64, userID uint64) (*entity.Merchant, error) {
	merchant, err := s.merchantRepo.FindByID(ctx, merchantID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询商户信息失败")
	}
	if merchant == nil {
		return nil, apperrors.NewNotFound("商户", merchantID)
	}
	if merchant.UserID != userID {
		return nil, apperrors.NewForbidden("无权操作该商户")
	}
	return merchant, nil
}

// generateRandomHex 生成指定字节数的随机 hex 字符串
// n 字节 => 2n 个十六进制字符
func generateRandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
