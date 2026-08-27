package app

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// newTestUserUseCase 创建测试用的 UserUseCase 及其依赖的 mock
func newTestUserUseCase() (*UserUseCase, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockUserSettingsRepository, *mocks.MockSystemConfigRepository) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	configRepo := new(mocks.MockSystemConfigRepository)

	userSvc := service.NewUserService(userRepo, walletRepo, settingsRepo, configRepo)
	walletSvc := service.NewWalletService(walletRepo)
	logger := zap.NewNop()

	// TRON 相关参数传 nil (测试不涉及链上功能)
	uc := NewUserUseCase(userSvc, walletSvc, nil, nil, nil, nil, "", logger)
	return uc, userRepo, walletRepo, settingsRepo, configRepo
}

// ==================== RegisterOrGet 测试 ====================

func TestUserUseCase_RegisterOrGet_已有用户直接返回(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	existing := &entity.User{ID: 1, TelegramID: 12345, Username: "testuser", PINHash: "some_hash"}
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(existing, nil)

	result, err := uc.RegisterOrGet(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsNew)
	assert.False(t, result.NeedPIN) // 已设置 PIN
	assert.Equal(t, existing, result.User)
}

func TestUserUseCase_RegisterOrGet_已有用户未设PIN(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	existing := &entity.User{ID: 1, TelegramID: 12345, Username: "testuser", PINHash: ""}
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(existing, nil)

	result, err := uc.RegisterOrGet(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsNew)
	assert.True(t, result.NeedPIN) // 未设置 PIN
}

func TestUserUseCase_RegisterOrGet_新用户注册(t *testing.T) {
	uc, userRepo, walletRepo, settingsRepo, _ := newTestUserUseCase()
	ctx := context.Background()

	// 用户不存在
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	// 注册流程
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(1).(*entity.User)
		u.ID = 1
	})
	walletRepo.On("Create", ctx, mock.AnythingOfType("*entity.Wallet")).Return(nil).Times(3)
	settingsRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	result, err := uc.RegisterOrGet(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsNew)
	assert.True(t, result.NeedPIN) // 新用户默认需要 PIN
}

func TestUserUseCase_RegisterOrGet_查询失败(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, errors.New("db error"))

	result, err := uc.RegisterOrGet(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserUseCase_RegisterOrGet_注册失败(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(nil, nil)
	userRepo.On("Create", ctx, mock.AnythingOfType("*entity.User")).Return(errors.New("db error"))

	result, err := uc.RegisterOrGet(ctx, 12345, "testuser", "Test", "User", "zh-CN", false)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetWalletOverview 测试 ====================

func TestUserUseCase_GetWalletOverview_成功(t *testing.T) {
	uc, userRepo, walletRepo, _, _ := newTestUserUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, TelegramID: 12345}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Currency: "TRX", Balance: decimal.NewFromInt(500)},
	}
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := uc.GetWalletOverview(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user, result.User)
	assert.True(t, decimal.NewFromInt(100).Equal(result.Balances["USDT"]))
	assert.True(t, decimal.NewFromInt(500).Equal(result.Balances["TRX"]))
}

func TestUserUseCase_GetWalletOverview_用户不存在(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := uc.GetWalletOverview(ctx, 999)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestUserUseCase_GetWalletOverview_查询用户失败(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.GetWalletOverview(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserUseCase_GetWalletOverview_查询钱包失败(t *testing.T) {
	uc, userRepo, walletRepo, _, _ := newTestUserUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.GetWalletOverview(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== SetPIN / VerifyPIN / NeedPIN 测试 ====================

func TestUserUseCase_SetPIN_成功(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: ""}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := uc.SetPIN(ctx, 1, "123456")

	assert.NoError(t, err)
}

func TestUserUseCase_SetPIN_失败(t *testing.T) {
	uc, userRepo, _, _, _ := newTestUserUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.SetPIN(ctx, 999, "123456")

	assert.Error(t, err)
}

func TestUserUseCase_VerifyPIN_委托调用(t *testing.T) {
	uc, userRepo, _, _, configRepo := newTestUserUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: "$2a$10$test"} // 假 hash，验证会失败
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	configRepo.On("Get", ctx, mock.Anything).Return("5", nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	_, _, err := uc.VerifyPIN(ctx, 1, "wrongpin")

	// 验证失败但不影响调用链
	assert.Error(t, err)
}

func TestUserUseCase_NeedPIN_委托调用(t *testing.T) {
	uc, userRepo, _, settingsRepo, _ := newTestUserUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: "some_hash"}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(&entity.UserSettings{
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(100),
	}, nil)

	needPIN, err := uc.NeedPIN(ctx, 1, "USDT", decimal.NewFromInt(50))

	assert.NoError(t, err)
	assert.False(t, needPIN) // 50 < 100 免密额度
}

// ==================== 构造函数测试 ====================

func TestNewUserUseCase(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	configRepo := new(mocks.MockSystemConfigRepository)

	userSvc := service.NewUserService(userRepo, walletRepo, settingsRepo, configRepo)
	walletSvc := service.NewWalletService(walletRepo)
	logger := zap.NewNop()

	uc := NewUserUseCase(userSvc, walletSvc, nil, nil, nil, nil, "", logger)

	assert.NotNil(t, uc)
}
