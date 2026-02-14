package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestProfileService 创建测试用的 ProfileService 及其所有 mock 依赖
func newTestProfileService() (*ProfileService, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockUserSettingsRepository, *mocks.MockTransactionRepository, *mocks.MockWithdrawalRepository) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	txRepo := new(mocks.MockTransactionRepository)
	withdrawalRepo := new(mocks.MockWithdrawalRepository)
	svc := NewProfileService(userRepo, walletRepo, settingsRepo, txRepo, withdrawalRepo)
	return svc, userRepo, walletRepo, settingsRepo, txRepo, withdrawalRepo
}

// ==================== GetProfileInfo 测试 ====================

func TestProfileService_GetProfileInfo_成功(t *testing.T) {
	svc, userRepo, walletRepo, _, _, _ := newTestProfileService()
	ctx := context.Background()

	user := &entity.User{ID: 1, TelegramID: 12345, Username: "testuser"}
	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Currency: "TRX", Balance: decimal.NewFromInt(500)},
		{ID: 3, UserID: 1, Currency: "CNY", Balance: decimal.NewFromFloat(688.88)},
	}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := svc.GetProfileInfo(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user, result.User)
	assert.True(t, decimal.NewFromInt(100).Equal(result.Balances["USDT"]))
	assert.True(t, decimal.NewFromInt(500).Equal(result.Balances["TRX"]))
	assert.True(t, decimal.NewFromFloat(688.88).Equal(result.Balances["CNY"]))
}

func TestProfileService_GetProfileInfo_用户不存在(t *testing.T) {
	svc, userRepo, _, _, _, _ := newTestProfileService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.GetProfileInfo(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestProfileService_GetProfileInfo_查询用户失败(t *testing.T) {
	svc, userRepo, _, _, _, _ := newTestProfileService()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetProfileInfo(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询用户失败")
}

func TestProfileService_GetProfileInfo_查询钱包失败(t *testing.T) {
	svc, userRepo, walletRepo, _, _, _ := newTestProfileService()
	ctx := context.Background()

	user := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(user, nil)
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetProfileInfo(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

// ==================== GetBills 测试 ====================

func TestProfileService_GetBills_成功(t *testing.T) {
	svc, _, _, _, txRepo, _ := newTestProfileService()
	ctx := context.Background()

	txs := []*entity.Transaction{
		{ID: 1, UserID: 1, Type: entity.TxTypeDeposit, Currency: "USDT", Amount: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Type: entity.TxTypeWithdraw, Currency: "USDT", Amount: decimal.NewFromInt(50)},
	}
	txRepo.On("ListByUserID", ctx, uint64(1), mock.AnythingOfType("*port.TransactionFilter"), 0, 10).Return(txs, int64(2), nil)

	result, err := svc.GetBills(ctx, 1, "", 1, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
	assert.Equal(t, 1, result.TotalPages)
}

func TestProfileService_GetBills_带币种筛选(t *testing.T) {
	svc, _, _, _, txRepo, _ := newTestProfileService()
	ctx := context.Background()

	txs := []*entity.Transaction{
		{ID: 1, UserID: 1, Type: entity.TxTypeDeposit, Currency: "USDT"},
	}
	txRepo.On("ListByUserID", ctx, uint64(1), mock.MatchedBy(func(f *port.TransactionFilter) bool {
		return f.Currency == "USDT"
	}), 0, 10).Return(txs, int64(1), nil)

	result, err := svc.GetBills(ctx, 1, "USDT", 1, 10)

	assert.NoError(t, err)
	assert.Len(t, result.Items, 1)
}

func TestProfileService_GetBills_ALL币种不筛选(t *testing.T) {
	svc, _, _, _, txRepo, _ := newTestProfileService()
	ctx := context.Background()

	txRepo.On("ListByUserID", ctx, uint64(1), mock.MatchedBy(func(f *port.TransactionFilter) bool {
		return f.Currency == "" // ALL 不筛选
	}), 0, 10).Return([]*entity.Transaction{}, int64(0), nil)

	result, err := svc.GetBills(ctx, 1, "ALL", 1, 10)

	assert.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestProfileService_GetBills_分页计算(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		total          int64
		wantPage       int
		wantPageSize   int
		wantTotalPages int
		wantOffset     int
	}{
		{"第一页", 1, 10, 25, 1, 10, 3, 0},
		{"第二页", 2, 10, 25, 2, 10, 3, 10},
		{"最后一页", 3, 10, 25, 3, 10, 3, 20},
		{"页码为0修正为1", 0, 10, 25, 1, 10, 3, 0},
		{"页码为负修正为1", -1, 10, 25, 1, 10, 3, 0},
		{"每页大小为0修正为10", 1, 0, 25, 1, 10, 3, 0},
		{"每页大小为负修正为10", 1, -5, 25, 1, 10, 3, 0},
		{"总数刚好整除", 1, 5, 10, 1, 5, 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, txRepo, _ := newTestProfileService()
			ctx := context.Background()

			txRepo.On("ListByUserID", ctx, uint64(1), mock.Anything, tt.wantOffset, tt.wantPageSize).Return([]*entity.Transaction{}, tt.total, nil)

			result, err := svc.GetBills(ctx, 1, "", tt.page, tt.pageSize)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPage, result.Page)
			assert.Equal(t, tt.wantPageSize, result.PageSize)
			assert.Equal(t, tt.wantTotalPages, result.TotalPages)
		})
	}
}

func TestProfileService_GetBills_查询失败(t *testing.T) {
	svc, _, _, _, txRepo, _ := newTestProfileService()
	ctx := context.Background()

	txRepo.On("ListByUserID", ctx, uint64(1), mock.Anything, 0, 10).Return([]*entity.Transaction{}, int64(0), errors.New("db error"))

	result, err := svc.GetBills(ctx, 1, "", 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询账单失败")
}

// ==================== GetWithdrawalHistory 测试 ====================

func TestProfileService_GetWithdrawalHistory_成功(t *testing.T) {
	svc, _, _, _, _, withdrawalRepo := newTestProfileService()
	ctx := context.Background()

	withdrawals := []*entity.Withdrawal{
		{ID: 1, UserID: 1, Currency: "USDT", Amount: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Currency: "USDT", Amount: decimal.NewFromInt(200)},
	}
	withdrawalRepo.On("ListByUserID", ctx, uint64(1), 0, 10).Return(withdrawals, int64(2), nil)

	result, err := svc.GetWithdrawalHistory(ctx, 1, 1, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, int64(2), result.Total)
	assert.Equal(t, 1, result.TotalPages)
}

func TestProfileService_GetWithdrawalHistory_分页修正(t *testing.T) {
	svc, _, _, _, _, withdrawalRepo := newTestProfileService()
	ctx := context.Background()

	withdrawalRepo.On("ListByUserID", ctx, uint64(1), 0, 10).Return([]*entity.Withdrawal{}, int64(0), nil)

	result, err := svc.GetWithdrawalHistory(ctx, 1, 0, 0) // page=0, pageSize=0

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.PageSize)
}

func TestProfileService_GetWithdrawalHistory_查询失败(t *testing.T) {
	svc, _, _, _, _, withdrawalRepo := newTestProfileService()
	ctx := context.Background()

	withdrawalRepo.On("ListByUserID", ctx, uint64(1), 0, 10).Return([]*entity.Withdrawal{}, int64(0), errors.New("db error"))

	result, err := svc.GetWithdrawalHistory(ctx, 1, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询提币历史失败")
}

// ==================== GetSmallFreeInfo 测试 ====================

func TestProfileService_GetSmallFreeInfo_有配置(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settings := &entity.UserSettings{
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(200),
		SmallFreeTRX:  decimal.NewFromInt(300),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)

	result, err := svc.GetSmallFreeInfo(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.NewFromInt(200).Equal(result.USDTLimit))
	assert.True(t, decimal.NewFromInt(300).Equal(result.TRXLimit))
}

func TestProfileService_GetSmallFreeInfo_无配置返回默认值(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, nil)

	result, err := svc.GetSmallFreeInfo(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, decimal.NewFromInt(100).Equal(result.USDTLimit))
	assert.True(t, decimal.NewFromInt(100).Equal(result.TRXLimit))
}

func TestProfileService_GetSmallFreeInfo_查询失败(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetSmallFreeInfo(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询用户配置失败")
}

// ==================== UpdateSmallFreeLimit 测试 ====================

func TestProfileService_UpdateSmallFreeLimit_更新USDT(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settings := &entity.UserSettings{
		ID:            1,
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(100),
		SmallFreeTRX:  decimal.NewFromInt(100),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)
	settingsRepo.On("Update", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	err := svc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(500))

	assert.NoError(t, err)
	assert.True(t, decimal.NewFromInt(500).Equal(settings.SmallFreeUSDT))
}

func TestProfileService_UpdateSmallFreeLimit_更新TRX(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settings := &entity.UserSettings{
		ID:            1,
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(100),
		SmallFreeTRX:  decimal.NewFromInt(100),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)
	settingsRepo.On("Update", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	err := svc.UpdateSmallFreeLimit(ctx, 1, "TRX", decimal.NewFromInt(300))

	assert.NoError(t, err)
	assert.True(t, decimal.NewFromInt(300).Equal(settings.SmallFreeTRX))
}

func TestProfileService_UpdateSmallFreeLimit_负数金额(t *testing.T) {
	svc, _, _, _, _, _ := newTestProfileService()
	ctx := context.Background()

	err := svc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(-10))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不能为负数")
}

func TestProfileService_UpdateSmallFreeLimit_不支持的币种(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settings := &entity.UserSettings{ID: 1, UserID: 1}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)

	err := svc.UpdateSmallFreeLimit(ctx, 1, "BTC", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持该币种")
}

func TestProfileService_UpdateSmallFreeLimit_配置不存在_创建新配置(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, nil)
	settingsRepo.On("Upsert", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	err := svc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(200))

	assert.NoError(t, err)
	settingsRepo.AssertCalled(t, "Upsert", ctx, mock.MatchedBy(func(s *entity.UserSettings) bool {
		return s.UserID == 1 && s.SmallFreeUSDT.Equal(decimal.NewFromInt(200))
	}))
}

func TestProfileService_UpdateSmallFreeLimit_查询配置失败(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.NewFromInt(200))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询用户配置失败")
}

func TestProfileService_UpdateSmallFreeLimit_设置为零_关闭免密(t *testing.T) {
	svc, _, _, settingsRepo, _, _ := newTestProfileService()
	ctx := context.Background()

	settings := &entity.UserSettings{
		ID:            1,
		UserID:        1,
		SmallFreeUSDT: decimal.NewFromInt(100),
	}
	settingsRepo.On("FindByUserID", ctx, uint64(1)).Return(settings, nil)
	settingsRepo.On("Update", ctx, mock.AnythingOfType("*entity.UserSettings")).Return(nil)

	err := svc.UpdateSmallFreeLimit(ctx, 1, "USDT", decimal.Zero)

	assert.NoError(t, err)
	assert.True(t, settings.SmallFreeUSDT.IsZero())
}
