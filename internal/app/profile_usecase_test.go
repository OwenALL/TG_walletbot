package app

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestProfileUseCase 创建测试用的 ProfileUseCase 及其 mock 依赖
func newTestProfileUseCase() (*ProfileUseCase, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockUserSettingsRepository, *mocks.MockTransactionRepository, *mocks.MockWithdrawalRepository, *mocks.MockSystemConfigRepository) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	txRepo := new(mocks.MockTransactionRepository)
	withdrawalRepo := new(mocks.MockWithdrawalRepository)
	configRepo := new(mocks.MockSystemConfigRepository)

	profileSvc := service.NewProfileService(userRepo, walletRepo, settingsRepo, txRepo, withdrawalRepo)
	userSvc := service.NewUserService(userRepo, walletRepo, settingsRepo, configRepo)

	uc := NewProfileUseCase(profileSvc, userSvc, zap.NewNop())
	return uc, userRepo, walletRepo, settingsRepo, txRepo, withdrawalRepo, configRepo
}

// ==================== GetProfileDashboard 测试 ====================

func TestProfileUseCase_GetProfileDashboard_成功(t *testing.T) {
	uc, userRepo, walletRepo, _, _, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, TelegramID: 12345, Username: "testuser"}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)},
	}
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := uc.GetProfileDashboard(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProfileUseCase_GetProfileDashboard_查询失败(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.GetProfileDashboard(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetBills 测试 ====================

func TestProfileUseCase_GetBills_成功(t *testing.T) {
	uc, _, _, _, txRepo, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	txs := []*entity.Transaction{
		{ID: 1, UserID: 1, Currency: "USDT", Amount: decimal.NewFromInt(50)},
	}
	txRepo.On("ListByUserID", ctx, uint64(1), mock.Anything, 0, 10).Return(txs, int64(1), nil)

	result, err := uc.GetBills(ctx, 1, "USDT", 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestProfileUseCase_GetBills_查询失败(t *testing.T) {
	uc, _, _, _, txRepo, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	txRepo.On("ListByUserID", ctx, uint64(1), mock.Anything, 0, 10).Return(([]*entity.Transaction)(nil), int64(0), errors.New("db error"))

	result, err := uc.GetBills(ctx, 1, "USDT", 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetWithdrawalHistory 测试 ====================

func TestProfileUseCase_GetWithdrawalHistory_成功(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _ := newTestProfileUseCase()
	ctx := context.Background()

	withdrawals := []*entity.Withdrawal{
		{ID: 1, UserID: 1, Currency: "USDT", Amount: decimal.NewFromInt(100)},
	}
	withdrawalRepo.On("ListByUserID", ctx, uint64(1), 0, 10).Return(withdrawals, int64(1), nil)

	result, err := uc.GetWithdrawalHistory(ctx, 1, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// ==================== GetSmallFreeInfo 测试 ====================

func TestProfileUseCase_GetSmallFreeInfo_成功(t *testing.T) {
	uc, _, _, settingsRepo, _, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	settings := &entity.UserSettings{
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(100),
		SmallFreeTRX:  decimal.NewFromInt(500),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)

	result, err := uc.GetSmallFreeInfo(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// ==================== UpdateSmallFreeLimit 测试 ====================

func TestProfileUseCase_UpdateSmallFreeLimit_成功(t *testing.T) {
	uc, _, _, settingsRepo, _, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	settings := &entity.UserSettings{
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(50),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)
	settingsRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	err := uc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(200))

	assert.NoError(t, err)
}

func TestProfileUseCase_UpdateSmallFreeLimit_失败(t *testing.T) {
	uc, _, _, settingsRepo, _, _, _ := newTestProfileUseCase()
	ctx := context.Background()

	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(200))

	assert.Error(t, err)
}

// ==================== 构造函数测试 ====================

func TestNewProfileUseCase(t *testing.T) {
	profileSvc := service.NewProfileService(nil, nil, nil, nil, nil)
	userSvc := service.NewUserService(nil, nil, nil, nil)

	uc := NewProfileUseCase(profileSvc, userSvc, zap.NewNop())
	assert.NotNil(t, uc)
}
