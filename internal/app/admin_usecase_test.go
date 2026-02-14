package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestAdminUseCase 创建测试用的 AdminUseCase (仅测试非 gorm.DB 直接操作的方法)
func newTestAdminUseCase() (*AdminUseCase, *mocks.MockAdminUserRepository, *mocks.MockAdminLogRepository, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockWithdrawalRepository, *mocks.MockExchangeRateRepository, *mocks.MockSystemConfigRepository, *mocks.MockTransactionRepository) {
	adminUserRepo := new(mocks.MockAdminUserRepository)
	adminLogRepo := new(mocks.MockAdminLogRepository)
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	withdrawalRepo := new(mocks.MockWithdrawalRepository)
	exchangeRateRepo := new(mocks.MockExchangeRateRepository)
	configRepo := new(mocks.MockSystemConfigRepository)
	txRepo := new(mocks.MockTransactionRepository)

	adminSvc := service.NewAdminService(adminUserRepo, adminLogRepo, zap.NewNop())

	uc := NewAdminUseCase(adminSvc, userRepo, walletRepo, txRepo, nil, withdrawalRepo, exchangeRateRepo, configRepo, nil, zap.NewNop())
	return uc, adminUserRepo, adminLogRepo, userRepo, walletRepo, withdrawalRepo, exchangeRateRepo, configRepo, txRepo
}

// ==================== Login 测试 ====================

func TestAdminUseCase_Login_成功(t *testing.T) {
	uc, adminUserRepo, _, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := &entity.AdminUser{ID: 1, Username: "admin", PasswordHash: string(hash), Status: 1}
	adminUserRepo.On("FindByUsername", ctx, "admin").Return(admin, nil)
	adminUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.AdminUser")).Return(nil)

	result, err := uc.Login(ctx, "admin", "admin123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "admin", result.Username)
}

func TestAdminUseCase_Login_用户不存在(t *testing.T) {
	uc, adminUserRepo, _, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	adminUserRepo.On("FindByUsername", ctx, "nobody").Return(nil, nil)

	result, err := uc.Login(ctx, "nobody", "pass")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAdminUseCase_Login_密码错误(t *testing.T) {
	uc, adminUserRepo, _, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	admin := &entity.AdminUser{ID: 1, Username: "admin", PasswordHash: string(hash), Status: 1}
	adminUserRepo.On("FindByUsername", ctx, "admin").Return(admin, nil)

	result, err := uc.Login(ctx, "admin", "wrong")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetAdminByID 测试 ====================

func TestAdminUseCase_GetAdminByID_成功(t *testing.T) {
	uc, adminUserRepo, _, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	admin := &entity.AdminUser{ID: 1, Username: "admin"}
	adminUserRepo.On("FindByID", ctx, uint64(1)).Return(admin, nil)

	result, err := uc.GetAdminByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, admin, result)
}

func TestAdminUseCase_GetAdminByID_不存在(t *testing.T) {
	uc, adminUserRepo, _, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	adminUserRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := uc.GetAdminByID(ctx, 999)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ==================== GetUserDetail 测试 ====================

func TestAdminUseCase_GetUserDetail_成功(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, TelegramID: 12345, Username: "testuser"}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT"},
	}
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := uc.GetUserDetail(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user, result.User)
	assert.Len(t, result.Wallets, 1)
}

func TestAdminUseCase_GetUserDetail_用户不存在(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := uc.GetUserDetail(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_GetUserDetail_查询用户失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.GetUserDetail(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestAdminUseCase_GetUserDetail_查询钱包失败(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.GetUserDetail(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

// ==================== UpdateUserStatus 测试 ====================

func TestAdminUseCase_UpdateUserStatus_成功(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, Status: int8(entity.UserStatusActive)}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := uc.UpdateUserStatus(ctx, 1, int8(entity.UserStatusFrozen))

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.UserStatusFrozen), user.Status)
}

func TestAdminUseCase_UpdateUserStatus_用户不存在(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.UpdateUserStatus(ctx, 999, int8(entity.UserStatusFrozen))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_UpdateUserStatus_查询失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.UpdateUserStatus(ctx, 1, int8(entity.UserStatusFrozen))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户失败")
}

// ==================== ApproveWithdrawal 测试 ====================

func TestAdminUseCase_ApproveWithdrawal_成功(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusPending}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := uc.ApproveWithdrawal(ctx, 1, 100)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusProcessing), withdrawal.Status)
}

func TestAdminUseCase_ApproveWithdrawal_不存在(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.ApproveWithdrawal(ctx, 999, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_ApproveWithdrawal_非待审核状态(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusCompleted}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)

	err := uc.ApproveWithdrawal(ctx, 1, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "非待审核状态")
}

func TestAdminUseCase_ApproveWithdrawal_查询失败(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.ApproveWithdrawal(ctx, 1, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

// ==================== GetAllExchangeRates 测试 ====================

func TestAdminUseCase_GetAllExchangeRates_成功(t *testing.T) {
	uc, _, _, _, _, _, exchangeRateRepo, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	rates := []*entity.ExchangeRate{
		{ID: 1, FromCurrency: "USDT", ToCurrency: "CNY"},
	}
	exchangeRateRepo.On("FindAll", ctx).Return(rates, nil)

	result, err := uc.GetAllExchangeRates(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

// ==================== UpdateExchangeRate 测试 ====================

func TestAdminUseCase_UpdateExchangeRate_成功(t *testing.T) {
	uc, _, _, _, _, _, exchangeRateRepo, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	rate := &entity.ExchangeRate{ID: 1, FromCurrency: "USDT", ToCurrency: "CNY"}
	exchangeRateRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.ExchangeRate")).Return(nil)

	err := uc.UpdateExchangeRate(ctx, rate)

	assert.NoError(t, err)
}

// ==================== GetAllConfigs / UpdateConfig 测试 ====================

func TestAdminUseCase_GetAllConfigs_成功(t *testing.T) {
	uc, _, _, _, _, _, _, configRepo, _ := newTestAdminUseCase()
	ctx := context.Background()

	configs := []*entity.SystemConfig{
		{ConfigKey: "withdraw_usdt_min", ConfigValue: "5"},
	}
	configRepo.On("GetAll", ctx).Return(configs, nil)

	result, err := uc.GetAllConfigs(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestAdminUseCase_UpdateConfig_成功(t *testing.T) {
	uc, _, _, _, _, _, _, configRepo, _ := newTestAdminUseCase()
	ctx := context.Background()

	configRepo.On("Set", ctx, "withdraw_usdt_min", "10").Return(nil)

	err := uc.UpdateConfig(ctx, "withdraw_usdt_min", "10")

	assert.NoError(t, err)
}

// ==================== CreateAuditLog / ListAuditLogs 测试 ====================

func TestAdminUseCase_CreateAuditLog_成功(t *testing.T) {
	uc, _, adminLogRepo, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	adminLogRepo.On("Create", ctx, mock.AnythingOfType("*entity.AdminLog")).Return(nil)

	// CreateAuditLog 不返回错误，测试不 panic
	assert.NotPanics(t, func() {
		uc.CreateAuditLog(ctx, 1, "login", "admin", nil, "登录成功", "127.0.0.1")
	})
}

func TestAdminUseCase_ListAuditLogs_成功(t *testing.T) {
	uc, _, adminLogRepo, _, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	logs := []*entity.AdminLog{{ID: 1, AdminID: 1, Action: "login"}}
	adminLogRepo.On("List", ctx, mock.AnythingOfType("*port.AdminLogFilter"), 0, 20).Return(logs, int64(1), nil)

	result, total, err := uc.ListAuditLogs(ctx, &port.AdminLogFilter{}, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), total)
}

// ==================== GetUserTransactions 测试 ====================

func TestAdminUseCase_GetUserTransactions_成功(t *testing.T) {
	uc, _, _, _, _, _, _, _, txRepo := newTestAdminUseCase()
	ctx := context.Background()

	txs := []*entity.Transaction{
		{ID: 1, UserID: 10, Currency: "USDT"},
	}
	txRepo.On("ListByUserID", ctx, uint64(10), mock.AnythingOfType("*port.TransactionFilter"), 0, 20).Return(txs, int64(1), nil)

	result, total, err := uc.GetUserTransactions(ctx, 10, &port.TransactionFilter{}, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), total)
}

func TestAdminUseCase_GetUserTransactions_查询失败(t *testing.T) {
	uc, _, _, _, _, _, _, _, txRepo := newTestAdminUseCase()
	ctx := context.Background()

	txRepo.On("ListByUserID", ctx, uint64(10), mock.AnythingOfType("*port.TransactionFilter"), 0, 20).Return(([]*entity.Transaction)(nil), int64(0), errors.New("db error"))

	result, _, err := uc.GetUserTransactions(ctx, 10, &port.TransactionFilter{}, 1, 20)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetDashboardStats 测试 ====================

func TestAdminUseCase_GetDashboardStats_查询用户数失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("CountAll", ctx).Return(int64(0), errors.New("db error"))

	result, err := uc.GetDashboardStats(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询用户数失败")
}

// ==================== RejectWithdrawal 测试 (仅测试 gorm.DB 访问前的分支) ====================

func TestAdminUseCase_RejectWithdrawal_不存在(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.RejectWithdrawal(ctx, 999, 100, "测试拒绝")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_RejectWithdrawal_查询失败(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.RejectWithdrawal(ctx, 1, 100, "测试拒绝")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

func TestAdminUseCase_RejectWithdrawal_非待审核状态(t *testing.T) {
	uc, _, _, _, _, withdrawalRepo, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusCompleted}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)

	err := uc.RejectWithdrawal(ctx, 1, 100, "测试拒绝")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "非待审核状态")
}

// ==================== ResetUserPIN 测试 ====================

func TestAdminUseCase_ResetUserPIN_成功(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	lockedUntil := time.Now().Add(30 * time.Minute)
	user := &entity.User{ID: 1, PINHash: "somehash", PINFailCount: 3, PINLockedUntil: &lockedUntil}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(nil)

	err := uc.ResetUserPIN(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, "", user.PINHash)
	assert.Equal(t, 0, user.PINFailCount)
	assert.Nil(t, user.PINLockedUntil)
}

func TestAdminUseCase_ResetUserPIN_用户不存在(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.ResetUserPIN(ctx, 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_ResetUserPIN_查询失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.ResetUserPIN(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestAdminUseCase_ResetUserPIN_更新失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, PINHash: "somehash"}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	userRepo.On("Update", ctx, mock.AnythingOfType("*entity.User")).Return(errors.New("update error"))

	err := uc.ResetUserPIN(ctx, 1)

	assert.Error(t, err)
}

// ==================== AdjustUserBalance 测试 ====================

func TestAdminUseCase_AdjustUserBalance_增加余额成功(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, Status: entity.UserStatusActive}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallet := &entity.Wallet{ID: 10, UserID: 1, Currency: entity.CurrencyUSDT, Balance: decimal.NewFromFloat(100)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(wallet, nil)
	walletRepo.On("UpdateBalance", ctx, uint64(10), decimal.NewFromFloat(50)).Return(nil)

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(50), "测试增加")

	assert.NoError(t, err)
}

func TestAdminUseCase_AdjustUserBalance_扣减余额成功(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1, Status: entity.UserStatusActive}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallet := &entity.Wallet{ID: 10, UserID: 1, Currency: entity.CurrencyUSDT, Balance: decimal.NewFromFloat(100)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(wallet, nil)
	walletRepo.On("UpdateBalance", ctx, uint64(10), decimal.NewFromFloat(-30)).Return(nil)

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(-30), "测试扣减")

	assert.NoError(t, err)
}

func TestAdminUseCase_AdjustUserBalance_用户不存在(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := uc.AdjustUserBalance(ctx, 999, entity.CurrencyUSDT, decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_AdjustUserBalance_不支持的币种(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	err := uc.AdjustUserBalance(ctx, 1, "BTC", decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的币种")
}

func TestAdminUseCase_AdjustUserBalance_钱包不存在(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(nil, nil)

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestAdminUseCase_AdjustUserBalance_余额不足(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallet := &entity.Wallet{ID: 10, UserID: 1, Currency: entity.CurrencyUSDT, Balance: decimal.NewFromFloat(50)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(wallet, nil)

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(-100), "测试扣减")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "余额不足")
}

func TestAdminUseCase_AdjustUserBalance_查询用户失败(t *testing.T) {
	uc, _, _, userRepo, _, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestAdminUseCase_AdjustUserBalance_查询钱包失败(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(nil, errors.New("db error"))

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

func TestAdminUseCase_AdjustUserBalance_更新余额失败(t *testing.T) {
	uc, _, _, userRepo, walletRepo, _, _, _, _ := newTestAdminUseCase()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)

	wallet := &entity.Wallet{ID: 10, UserID: 1, Currency: entity.CurrencyUSDT, Balance: decimal.NewFromFloat(100)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), entity.CurrencyUSDT).Return(wallet, nil)
	walletRepo.On("UpdateBalance", ctx, uint64(10), decimal.NewFromFloat(10)).Return(errors.New("update error"))

	err := uc.AdjustUserBalance(ctx, 1, entity.CurrencyUSDT, decimal.NewFromFloat(10), "测试")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "更新余额失败")
}

// ==================== 构造函数测试 ====================

func TestNewAdminUseCase(t *testing.T) {
	adminSvc := service.NewAdminService(nil, nil, zap.NewNop())

	uc := NewAdminUseCase(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, zap.NewNop())
	assert.NotNil(t, uc)
}
