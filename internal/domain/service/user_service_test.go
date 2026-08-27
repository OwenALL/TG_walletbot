package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// newTestUserService 创建测试用的 UserService 及其所有 mock 依赖
func newTestUserService() (*UserService, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockUserSettingsRepository, *mocks.MockSystemConfigRepository) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	configRepo := new(mocks.MockSystemConfigRepository)
	svc := NewUserService(userRepo, walletRepo, settingsRepo, configRepo)
	return svc, userRepo, walletRepo, settingsRepo, configRepo
}

// hashPIN 辅助函数：生成 bcrypt hash
func hashPIN(pin string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	return string(hash)
}

// ==================== Register 测试 ====================

func TestUserService_Register_新用户注册成功(t *testing.T) {
	svc, userRepo, walletRepo, settingsRepo, _ := newTestUserService()
	ctx := context.Background()

	// 用户不存在
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	// 创建用户成功
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(1).(*entity.User)
		u.ID = 1 // 模拟自增 ID
	})
	// 创建三个钱包
	walletRepo.On("Create", ctx, mock.AnythingOfType("*entity.Wallet")).Return(nil).Times(3)
	// 创建配置
	settingsRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(12345), user.TelegramID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, int8(entity.UserStatusActive), user.Status)
	userRepo.AssertExpectations(t)
	walletRepo.AssertExpectations(t)
	settingsRepo.AssertExpectations(t)
}

func TestUserService_Register_已注册用户直接返回(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	existing := &entity.User{ID: 1, TelegramID: 12345, Username: "existinguser"}
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(existing, nil)

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uint64(1), user.ID)
	assert.Equal(t, "existinguser", user.Username)
	userRepo.AssertExpectations(t)
}

func TestUserService_Register_查询用户失败(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, errors.New("db error"))

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestUserService_Register_创建用户失败(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "创建用户失败")
}

func TestUserService_Register_创建钱包失败(t *testing.T) {
	svc, userRepo, walletRepo, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(1).(*entity.User)
		u.ID = 1
	})
	// 第一个钱包创建成功，第二个失败
	walletRepo.On("Create", ctx, mock.AnythingOfType("*entity.Wallet")).Return(nil).Once()
	walletRepo.On("Create", ctx, mock.AnythingOfType("*entity.Wallet")).Return(errors.New("db error")).Once()

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "钱包失败")
}

func TestUserService_Register_创建配置失败(t *testing.T) {
	svc, userRepo, walletRepo, settingsRepo, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(1).(*entity.User)
		u.ID = 1
	})
	walletRepo.On("Create", ctx, mock.AnythingOfType("*entity.Wallet")).Return(nil).Times(3)
	settingsRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(errors.New("db error"))

	user, err := svc.Register(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "创建用户配置失败")
}

// ==================== SetPIN 测试 ====================

func TestUserService_SetPIN_成功(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: "", PINFailCount: 3}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := svc.SetPIN(ctx, 1, "123456")

	assert.NoError(t, err)
	assert.NotEmpty(t, user.PINHash)
	assert.Equal(t, 0, user.PINFailCount)
	assert.Nil(t, user.PINLockedUntil)
}

func TestUserService_SetPIN_用户不存在(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.SetPIN(ctx, 999, "123456")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestUserService_SetPIN_查询失败(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.SetPIN(ctx, 1, "123456")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户失败")
}

// ==================== VerifyPIN 测试 ====================

func TestUserService_VerifyPIN_验证成功(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	pin := "123456"
	user := &entity.User{ID: 1, PINHash: hashPIN(pin), PINFailCount: 0}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, pin)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 5, remaining)
}

func TestUserService_VerifyPIN_验证成功_重置失败计数(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	pin := "123456"
	user := &entity.User{ID: 1, PINHash: hashPIN(pin), PINFailCount: 2}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, pin)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 5, remaining)
	assert.Equal(t, 0, user.PINFailCount)
	assert.Nil(t, user.PINLockedUntil)
}

func TestUserService_VerifyPIN_密码错误_增加失败计数(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: hashPIN("123456"), PINFailCount: 0}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, "wrongpin")

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, 4, remaining)
	assert.Equal(t, 1, user.PINFailCount)
}

func TestUserService_VerifyPIN_连续错误触发锁定(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: hashPIN("123456"), PINFailCount: 4}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, "wrongpin")

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, remaining)
	assert.NotNil(t, user.PINLockedUntil)
	assert.Contains(t, err.Error(), "锁定")
}

func TestUserService_VerifyPIN_账户已锁定(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	lockUntil := time.Now().Add(30 * time.Minute)
	user := &entity.User{ID: 1, PINHash: hashPIN("123456"), PINLockedUntil: &lockUntil}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, "123456")

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, remaining)
	assert.Contains(t, err.Error(), "锁定")
}

func TestUserService_VerifyPIN_锁定已过期_自动解锁(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	pin := "123456"
	lockUntil := time.Now().Add(-1 * time.Minute) // 锁定已过期
	user := &entity.User{ID: 1, PINHash: hashPIN(pin), PINFailCount: 5, PINLockedUntil: &lockUntil}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	ok, _, err := svc.VerifyPIN(ctx, 1, pin)

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestUserService_VerifyPIN_未设置PIN(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: ""}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	ok, _, err := svc.VerifyPIN(ctx, 1, "123456")

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "设置支付密码")
}

func TestUserService_VerifyPIN_用户不存在(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	ok, _, err := svc.VerifyPIN(ctx, 999, "123456")

	assert.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "不存在")
}

func TestUserService_VerifyPIN_配置读取失败使用默认值(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	pin := "123456"
	user := &entity.User{ID: 1, PINHash: hashPIN(pin), PINFailCount: 0}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("", errors.New("redis error"))
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("", errors.New("redis error"))

	ok, remaining, err := svc.VerifyPIN(ctx, 1, pin)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 5, remaining) // 默认 5
}

// ==================== ChangePIN 测试 ====================

func TestUserService_ChangePIN_成功(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	oldPIN := "123456"
	newPIN := "654321"
	user := &entity.User{ID: 1, PINHash: hashPIN(oldPIN), PINFailCount: 0}

	// VerifyPIN 的调用链
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	// SetPIN 的 Update 调用
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := svc.ChangePIN(ctx, 1, oldPIN, newPIN)

	assert.NoError(t, err)
	// 验证新 PIN 已被设置
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PINHash), []byte(newPIN)))
}

func TestUserService_ChangePIN_旧密码错误(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: hashPIN("123456"), PINFailCount: 0}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("30", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := svc.ChangePIN(ctx, 1, "wrongold", "654321")

	assert.Error(t, err)
}

// ==================== GetByTelegramID / GetByID 测试 ====================

func TestUserService_GetByTelegramID_成功(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	expected := &entity.User{ID: 1, TelegramID: 12345}
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(expected, nil)

	user, err := svc.GetByTelegramID(ctx, 12345)

	assert.NoError(t, err)
	assert.Equal(t, expected, user)
}

func TestUserService_GetByID_成功(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	expected := &entity.User{ID: 1, TelegramID: 12345}
	userRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	user, err := svc.GetByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, user)
}

// ==================== NeedPIN 测试 ====================

func TestUserService_NeedPIN(t *testing.T) {
	tests := []struct {
		name      string
		user      *entity.User
		settings  *entity.UserSettings
		currency  string
		amount    decimal.Decimal
		wantNeed  bool
		setupUser func(repo *mocks.MockUserRepository)
	}{
		{
			name:     "用户未设置PIN_需要验证",
			user:     &entity.User{ID: 1, PINHash: ""},
			settings: nil,
			currency: "USDT",
			amount:   decimal.NewFromInt(10),
			wantNeed: true,
		},
		{
			name: "金额低于免密额度_不需要验证",
			user: &entity.User{ID: 1, PINHash: hashPIN("123456")},
			settings: &entity.UserSettings{
				UserID:        1,
				SmallFreeUSDT: decimal.NewFromInt(100),
				SmallFreeTRX:  decimal.NewFromInt(100),
			},
			currency: "USDT",
			amount:   decimal.NewFromInt(50),
			wantNeed: false,
		},
		{
			name: "金额超过免密额度_需要验证",
			user: &entity.User{ID: 1, PINHash: hashPIN("123456")},
			settings: &entity.UserSettings{
				UserID:        1,
				SmallFreeUSDT: decimal.NewFromInt(100),
				SmallFreeTRX:  decimal.NewFromInt(100),
			},
			currency: "USDT",
			amount:   decimal.NewFromInt(200),
			wantNeed: true,
		},
		{
			name: "CNY币种_始终需要验证",
			user: &entity.User{ID: 1, PINHash: hashPIN("123456")},
			settings: &entity.UserSettings{
				UserID:        1,
				SmallFreeUSDT: decimal.NewFromInt(100),
				SmallFreeTRX:  decimal.NewFromInt(100),
			},
			currency: "CNY",
			amount:   decimal.NewFromInt(1),
			wantNeed: true,
		},
		{
			name: "免密额度为0_关闭免密",
			user: &entity.User{ID: 1, PINHash: hashPIN("123456")},
			settings: &entity.UserSettings{
				UserID:        1,
				SmallFreeUSDT: decimal.Zero,
			},
			currency: "USDT",
			amount:   decimal.NewFromInt(1),
			wantNeed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, userRepo, _, settingsRepo, _ := newTestUserService()
			ctx := context.Background()

			userRepo.On("FindByID", ctx, uint64(1)).Return(tt.user, nil)
			if tt.user.HasPIN() {
				settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(tt.settings, nil)
			}

			needPIN, err := svc.NeedPIN(ctx, 1, tt.currency, tt.amount)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantNeed, needPIN)
		})
	}
}

func TestUserService_NeedPIN_用户不存在(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	needPIN, err := svc.NeedPIN(ctx, 999, "USDT", decimal.NewFromInt(10))

	assert.NoError(t, err)
	assert.True(t, needPIN) // 用户不存在时默认需要 PIN
}

func TestUserService_NeedPIN_查询失败(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	needPIN, err := svc.NeedPIN(ctx, 1, "USDT", decimal.NewFromInt(10))

	assert.Error(t, err)
	assert.True(t, needPIN)
}

func TestUserService_NeedPIN_查询配置失败_默认需要验证(t *testing.T) {
	svc, userRepo, _, settingsRepo, _ := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: hashPIN("123456")}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	needPIN, err := svc.NeedPIN(ctx, 1, "USDT", decimal.NewFromInt(10))

	assert.NoError(t, err)
	assert.True(t, needPIN) // 查询配置失败时默认需要验证
}

func TestUserService_ChangePIN_验证出错_错误传播(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	// VerifyPIN 查询用户失败，直接返回错误
	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.ChangePIN(ctx, 1, "123456", "654321")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestUserService_VerifyPIN_配置值无效使用默认(t *testing.T) {
	svc, userRepo, _, _, configRepo := newTestUserService()
	ctx := context.Background()

	pin := "123456"
	user := &entity.User{ID: 1, PINHash: hashPIN(pin), PINFailCount: 0}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	// 配置返回非数字字符串 (无error，但 Atoi 会失败)
	configRepo.On("Get", ctx, entity.ConfigPINMaxFail).Return("abc", nil)
	configRepo.On("Get", ctx, entity.ConfigPINLockMinutes).Return("xyz", nil)

	ok, remaining, err := svc.VerifyPIN(ctx, 1, pin)

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 5, remaining) // 使用默认值 5
}

func TestUserService_SetPIN_更新失败(t *testing.T) {
	svc, userRepo, _, _, _ := newTestUserService()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: ""}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	err := svc.SetPIN(ctx, 1, "123456")

	assert.Error(t, err)
}
