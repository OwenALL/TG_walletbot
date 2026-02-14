// Package service 定义领域服务，封装核心业务规则
// 仅依赖 domain/port 接口，不依赖任何具体实现
package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// UserService 用户领域服务
type UserService struct {
	userRepo     port.UserRepository
	walletRepo   port.WalletRepository
	settingsRepo port.UserSettingsRepository
	configRepo   port.SystemConfigRepository
}

// NewUserService 创建用户领域服务
func NewUserService(
	userRepo port.UserRepository,
	walletRepo port.WalletRepository,
	settingsRepo port.UserSettingsRepository,
	configRepo port.SystemConfigRepository,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		walletRepo:   walletRepo,
		settingsRepo: settingsRepo,
		configRepo:   configRepo,
	}
}

// Register 注册新用户
// 创建用户 + 三个钱包 (USDT/TRX/CNY) + 用户配置
func (s *UserService) Register(ctx context.Context, telegramID int64, username, firstName, lastName, languageCode string, isPremium bool) (*entity.User, error) {
	// 检查是否已注册
	existing, err := s.userRepo.FindByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户失败")
	}
	if existing != nil {
		return existing, nil
	}

	// 创建用户
	user := &entity.User{
		TelegramID:   telegramID,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		LanguageCode: languageCode,
		IsPremium:    isPremium,
		Status:       entity.UserStatusActive,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Wrap(err, "创建用户失败")
	}

	// 创建三个钱包
	currencies := []string{entity.CurrencyUSDT, entity.CurrencyTRX, entity.CurrencyCNY}
	for _, currency := range currencies {
		wallet := &entity.Wallet{
			UserID:   user.ID,
			Currency: currency,
			Balance:  decimal.Zero,
		}
		if err := s.walletRepo.Create(ctx, wallet); err != nil {
			return nil, apperrors.Wrap(err, fmt.Sprintf("创建 %s 钱包失败", currency))
		}
	}

	// 创建用户配置 (默认值)
	settings := &entity.UserSettings{
		UserID:              user.ID,
		Language:            languageCode,
		SmallFreeUSDT:       decimal.NewFromInt(100),
		SmallFreeTRX:        decimal.NewFromInt(100),
		DailySpentUSDT:      decimal.Zero,
		DailySpentTRX:       decimal.Zero,
		NotificationEnabled: true,
	}
	if err := s.settingsRepo.Upsert(ctx, settings); err != nil {
		return nil, apperrors.Wrap(err, "创建用户配置失败")
	}

	return user, nil
}

// SetPIN 设置支付密码
func (s *UserService) SetPIN(ctx context.Context, userID uint64, pin string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return apperrors.NewNotFound("用户", userID)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.Wrap(err, "加密 PIN 失败")
	}

	user.PINHash = string(hash)
	user.PINFailCount = 0
	user.PINLockedUntil = nil
	return s.userRepo.Update(ctx, user)
}

// VerifyPIN 验证支付密码
// 返回: 验证是否通过, 剩余重试次数, 错误
func (s *UserService) VerifyPIN(ctx context.Context, userID uint64, pin string) (bool, int, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return false, 0, apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return false, 0, apperrors.NewNotFound("用户", userID)
	}

	if !user.HasPIN() {
		return false, 0, apperrors.NewPINNotSet()
	}

	// 检查是否锁定
	if user.IsPINLocked() {
		return false, 0, apperrors.NewPINLocked(30)
	}

	// 获取最大失败次数配置
	maxFail := 5
	if val, err := s.configRepo.Get(ctx, entity.ConfigPINMaxFail); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxFail = n
		}
	}

	// 获取锁定时长配置
	lockMinutes := 30
	if val, err := s.configRepo.Get(ctx, entity.ConfigPINLockMinutes); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			lockMinutes = n
		}
	}

	// 验证 PIN
	if err := bcrypt.CompareHashAndPassword([]byte(user.PINHash), []byte(pin)); err != nil {
		// PIN 错误，增加失败计数
		user.PINFailCount++
		remaining := maxFail - user.PINFailCount

		if user.PINFailCount >= maxFail {
			// 锁定账户
			lockUntil := time.Now().Add(time.Duration(lockMinutes) * time.Minute)
			user.PINLockedUntil = &lockUntil
			_ = s.userRepo.Update(ctx, user)
			return false, 0, apperrors.NewPINLocked(lockMinutes)
		}

		_ = s.userRepo.Update(ctx, user)
		return false, remaining, apperrors.NewPINInvalid()
	}

	// 验证通过，重置失败计数
	if user.PINFailCount > 0 {
		user.PINFailCount = 0
		user.PINLockedUntil = nil
		_ = s.userRepo.Update(ctx, user)
	}

	return true, maxFail, nil
}

// ChangePIN 修改支付密码 (需要旧 PIN 验证)
func (s *UserService) ChangePIN(ctx context.Context, userID uint64, oldPIN, newPIN string) error {
	ok, _, err := s.VerifyPIN(ctx, userID, oldPIN)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.NewPINInvalid()
	}
	return s.SetPIN(ctx, userID, newPIN)
}

// GetByTelegramID 根据 Telegram ID 获取用户
func (s *UserService) GetByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error) {
	return s.userRepo.FindByTelegramID(ctx, telegramID)
}

// GetByID 根据 ID 获取用户
func (s *UserService) GetByID(ctx context.Context, id uint64) (*entity.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// NeedPIN 判断指定金额操作是否需要 PIN 验证
func (s *UserService) NeedPIN(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) (bool, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return true, err
	}
	if user == nil || !user.HasPIN() {
		return true, nil
	}

	settings, err := s.settingsRepo.FindByUserID(ctx, userID)
	if err != nil || settings == nil {
		return true, nil
	}

	return settings.NeedPINForAmount(currency, amount), nil
}
