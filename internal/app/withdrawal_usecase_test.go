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

// newTestWithdrawalUseCase 创建测试用的 WithdrawalUseCase 及其 mock 依赖
func newTestWithdrawalUseCase() (*WithdrawalUseCase, *mocks.MockWithdrawalRepository, *mocks.MockWalletRepository, *mocks.MockTransactionRepository, *mocks.MockSystemConfigRepository, *mocks.MockUserRepository, *mocks.MockUserSettingsRepository) {
	withdrawalRepo := new(mocks.MockWithdrawalRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	configRepo := new(mocks.MockSystemConfigRepository)
	userRepo := new(mocks.MockUserRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)

	withdrawSvc := service.NewWithdrawalService(withdrawalRepo, walletRepo, txRepo, configRepo)
	walletSvc := service.NewWalletService(walletRepo)
	userSvc := service.NewUserService(userRepo, walletRepo, settingsRepo, configRepo)

	uc := NewWithdrawalUseCase(withdrawSvc, walletSvc, userSvc, nil, zap.NewNop())
	return uc, withdrawalRepo, walletRepo, txRepo, configRepo, userRepo, settingsRepo
}

// setupUSDTConfig 设置 USDT 提币配置
func setupUSDTConfig(configRepo *mocks.MockSystemConfigRepository) {
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMin).Return("5", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMax).Return("50000", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTDailyMax).Return("100000", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTFee).Return("1", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTAutoThreshold).Return("1000", nil)
}

// ==================== GetBalance 测试 ====================

func TestWithdrawalUseCase_GetBalance_成功(t *testing.T) {
	uc, _, walletRepo, _, _, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(500), FrozenBalance: decimal.NewFromInt(100)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)

	balance, err := uc.GetBalance(ctx, 10, "USDT")

	assert.NoError(t, err)
	// AvailableBalance = Balance - FrozenBalance = 500 - 100 = 400
	assert.True(t, decimal.NewFromInt(400).Equal(balance))
}

func TestWithdrawalUseCase_GetBalance_查询失败(t *testing.T) {
	uc, _, walletRepo, _, _, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, errors.New("db error"))

	_, err := uc.GetBalance(ctx, 10, "USDT")

	assert.Error(t, err)
}

// ==================== ValidateAddress 测试 ====================

func TestWithdrawalUseCase_ValidateAddress(t *testing.T) {
	uc, _, _, _, _, _, _ := newTestWithdrawalUseCase()

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{"空地址", "", false},
		{"长度不足", "TShort", false},
		{"非T开头", "1234567890123456789012345678901234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.ValidateAddress(tt.address)
			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== ValidateAmount 测试 ====================

func TestWithdrawalUseCase_ValidateAmount_成功(t *testing.T) {
	uc, withdrawalRepo, walletRepo, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	withdrawalRepo.On("SumTodayByUserID", ctx, uint64(10), "USDT").Return(decimal.Zero, nil)

	err := uc.ValidateAmount(ctx, 10, "USDT", decimal.NewFromInt(100))

	assert.NoError(t, err)
}

func TestWithdrawalUseCase_ValidateAmount_低于最低限额(t *testing.T) {
	uc, _, _, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// 最低 5 USDT
	err := uc.ValidateAmount(ctx, 10, "USDT", decimal.NewFromInt(1))

	assert.Error(t, err)
}

func TestWithdrawalUseCase_ValidateAmount_余额不足(t *testing.T) {
	uc, withdrawalRepo, walletRepo, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// 余额 10，提 100
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(10), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	withdrawalRepo.On("SumTodayByUserID", ctx, uint64(10), "USDT").Return(decimal.Zero, nil)

	err := uc.ValidateAmount(ctx, 10, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
}

// ==================== GetWithdrawFee 测试 ====================

func TestWithdrawalUseCase_GetWithdrawFee(t *testing.T) {
	uc, _, _, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	fee := uc.GetWithdrawFee(ctx, "USDT")

	assert.True(t, decimal.NewFromInt(1).Equal(fee))
}

// ==================== BuildConfirmInfo 测试 ====================

func TestWithdrawalUseCase_BuildConfirmInfo_成功(t *testing.T) {
	uc, _, _, _, configRepo, userRepo, settingsRepo := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// NeedPIN 调用链
	user := &entity.User{ID: 10, PINHash: "some_hash"}
	userRepo.On("FindByID", ctx, uint64(10)).Return(user, nil)
	settingsRepo.On("FindByUserID", ctx, uint64(10)).Return(&entity.UserSettings{
		UserID:        10,
		SmallFreeUSDT: decimal.NewFromInt(50),
	}, nil)

	result, err := uc.BuildConfirmInfo(ctx, 10, "USDT", decimal.NewFromInt(100), "TAddr123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "USDT", result.Currency)
	assert.True(t, decimal.NewFromInt(100).Equal(result.Amount))
	assert.True(t, decimal.NewFromInt(1).Equal(result.Fee))
	assert.True(t, decimal.NewFromInt(99).Equal(result.ActualAmount))
	assert.Equal(t, "TAddr123", result.ToAddress)
	assert.True(t, result.NeedPIN) // 100 > 50 免密额度
}

func TestWithdrawalUseCase_BuildConfirmInfo_金额小于手续费(t *testing.T) {
	uc, _, _, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// 金额 0.5 小于手续费 1
	result, err := uc.BuildConfirmInfo(ctx, 10, "USDT", decimal.NewFromFloat(0.5), "TAddr123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "手续费")
}

// ==================== SubmitWithdrawal 测试 ====================

func TestWithdrawalUseCase_SubmitWithdrawal_验证金额失败(t *testing.T) {
	uc, _, _, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// 金额低于最低限额 (min = 5)
	result, err := uc.SubmitWithdrawal(ctx, 10, "USDT", "TAddr123", decimal.NewFromInt(1))

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWithdrawalUseCase_SubmitWithdrawal_地址无效(t *testing.T) {
	uc, withdrawalRepo, walletRepo, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// ValidateAmount 需要钱包查询和日限额查询的 mock
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	withdrawalRepo.On("SumTodayByUserID", ctx, uint64(10), "USDT").Return(decimal.Zero, nil)

	// 地址 "invalid" 不满足 TRON Base58Check 格式
	result, err := uc.SubmitWithdrawal(ctx, 10, "USDT", "invalid", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWithdrawalUseCase_SubmitWithdrawal_创建提币失败(t *testing.T) {
	uc, withdrawalRepo, walletRepo, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// ValidateAmount 需要的 mock
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	withdrawalRepo.On("SumTodayByUserID", ctx, uint64(10), "USDT").Return(decimal.Zero, nil)

	// CreateWithdrawal 内部: 冻结成功但创建记录失败
	walletRepo.On("FreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Create", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(errors.New("db error"))
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)

	// 使用真实的 TRON Base58Check 地址通过地址验证
	result, err := uc.SubmitWithdrawal(ctx, 10, "USDT", "TNPeeaaFB7K9cmo4uQpcU32zGK8G1NYqeL", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestWithdrawalUseCase_SubmitWithdrawal_成功(t *testing.T) {
	uc, withdrawalRepo, walletRepo, _, configRepo, _, _ := newTestWithdrawalUseCase()
	ctx := context.Background()
	setupUSDTConfig(configRepo)

	// ValidateAmount 需要的 mock
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	withdrawalRepo.On("SumTodayByUserID", ctx, uint64(10), "USDT").Return(decimal.Zero, nil)

	// CreateWithdrawal 内部: 冻结和创建都成功
	walletRepo.On("FreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Create", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	// 使用真实的 TRON Base58Check 地址通过地址验证
	result, err := uc.SubmitWithdrawal(ctx, 10, "USDT", "TNPeeaaFB7K9cmo4uQpcU32zGK8G1NYqeL", decimal.NewFromInt(100))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(10), result.UserID)
}

// ==================== 构造函数测试 ====================

func TestNewWithdrawalUseCase(t *testing.T) {
	withdrawSvc := service.NewWithdrawalService(nil, nil, nil, nil)
	walletSvc := service.NewWalletService(nil)
	userSvc := service.NewUserService(nil, nil, nil, nil)

	uc := NewWithdrawalUseCase(withdrawSvc, walletSvc, userSvc, nil, zap.NewNop())
	assert.NotNil(t, uc)
}
