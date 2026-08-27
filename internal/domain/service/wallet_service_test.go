package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// newTestWalletService 创建测试用的 WalletService 及其 mock 依赖
func newTestWalletService() (*WalletService, *mocks.MockWalletRepository) {
	walletRepo := new(mocks.MockWalletRepository)
	svc := NewWalletService(walletRepo)
	return svc, walletRepo
}

// ==================== GetWallets 测试 ====================

func TestWalletService_GetWallets_成功(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Currency: "TRX", Balance: decimal.NewFromInt(500)},
		{ID: 3, UserID: 1, Currency: "CNY", Balance: decimal.NewFromInt(0)},
	}
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := svc.GetWallets(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "USDT", result[0].Currency)
}

func TestWalletService_GetWallets_查询失败(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetWallets(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

// ==================== GetWallet 测试 ====================

func TestWalletService_GetWallet_成功(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	result, err := svc.GetWallet(ctx, 1, "USDT")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, decimal.NewFromInt(100), result.Balance)
}

func TestWalletService_GetWallet_钱包不存在(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "BTC").Return(nil, nil)

	result, err := svc.GetWallet(ctx, 1, "BTC")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestWalletService_GetWallet_查询失败(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(nil, errors.New("db error"))

	result, err := svc.GetWallet(ctx, 1, "USDT")

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetBalanceMap 测试 ====================

func TestWalletService_GetBalanceMap_成功(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	wallets := []*entity.Wallet{
		{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(100)},
		{ID: 2, UserID: 1, Currency: "TRX", Balance: decimal.NewFromInt(500)},
	}
	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(wallets, nil)

	result, err := svc.GetBalanceMap(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, decimal.NewFromInt(100), result["USDT"])
	assert.Equal(t, decimal.NewFromInt(500), result["TRX"])
	assert.Equal(t, decimal.Zero, result["CNY"]) // 不存在的币种应返回 0
}

func TestWalletService_GetBalanceMap_空钱包(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	walletRepo.On("FindByUserID", ctx, uint64(1)).Return([]*entity.Wallet{}, nil)

	result, err := svc.GetBalanceMap(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, decimal.Zero, result["USDT"])
	assert.Equal(t, decimal.Zero, result["TRX"])
	assert.Equal(t, decimal.Zero, result["CNY"])
}

func TestWalletService_GetBalanceMap_查询失败(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	walletRepo.On("FindByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetBalanceMap(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== UpdateBalance 测试 ====================

func TestWalletService_UpdateBalance_增加余额(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(50)
	walletRepo.On("UpdateBalance", ctx, uint64(1), amount).Return(nil)

	err := svc.UpdateBalance(ctx, 1, amount)

	assert.NoError(t, err)
	walletRepo.AssertExpectations(t)
}

func TestWalletService_UpdateBalance_扣减余额(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(-30)
	walletRepo.On("UpdateBalance", ctx, uint64(1), amount).Return(nil)

	err := svc.UpdateBalance(ctx, 1, amount)

	assert.NoError(t, err)
	walletRepo.AssertExpectations(t)
}

func TestWalletService_UpdateBalance_失败(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(50)
	walletRepo.On("UpdateBalance", ctx, uint64(1), amount).Return(errors.New("insufficient balance"))

	err := svc.UpdateBalance(ctx, 1, amount)

	assert.Error(t, err)
}

// ==================== FreezeBalance 测试 ====================

func TestWalletService_FreezeBalance_成功(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(50)
	walletRepo.On("FreezeBalance", ctx, uint64(1), amount).Return(nil)

	err := svc.FreezeBalance(ctx, 1, amount)

	assert.NoError(t, err)
}

func TestWalletService_FreezeBalance_余额不足(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(9999)
	walletRepo.On("FreezeBalance", ctx, uint64(1), amount).Return(errors.New("insufficient available balance"))

	err := svc.FreezeBalance(ctx, 1, amount)

	assert.Error(t, err)
}

// ==================== UnfreezeBalance 测试 ====================

func TestWalletService_UnfreezeBalance_成功(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(50)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), amount).Return(nil)

	err := svc.UnfreezeBalance(ctx, 1, amount)

	assert.NoError(t, err)
}

func TestWalletService_UnfreezeBalance_失败(t *testing.T) {
	svc, walletRepo := newTestWalletService()
	ctx := context.Background()

	amount := decimal.NewFromInt(9999)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), amount).Return(errors.New("frozen balance not enough"))

	err := svc.UnfreezeBalance(ctx, 1, amount)

	assert.Error(t, err)
}

// ==================== CheckBalance 测试 ====================

func TestWalletService_CheckBalance(t *testing.T) {
	tests := []struct {
		name       string
		wallet     *entity.Wallet
		findErr    error
		amount     decimal.Decimal
		wantErr    bool
		errContain string
	}{
		{
			name: "余额充足",
			wallet: &entity.Wallet{
				ID:            1,
				UserID:        1,
				Currency:      "USDT",
				Balance:       decimal.NewFromInt(100),
				FrozenBalance: decimal.Zero,
			},
			amount:  decimal.NewFromInt(50),
			wantErr: false,
		},
		{
			name: "余额刚好",
			wallet: &entity.Wallet{
				ID:            1,
				UserID:        1,
				Currency:      "USDT",
				Balance:       decimal.NewFromInt(50),
				FrozenBalance: decimal.Zero,
			},
			amount:  decimal.NewFromInt(50),
			wantErr: false,
		},
		{
			name: "余额不足",
			wallet: &entity.Wallet{
				ID:            1,
				UserID:        1,
				Currency:      "USDT",
				Balance:       decimal.NewFromInt(30),
				FrozenBalance: decimal.Zero,
			},
			amount:     decimal.NewFromInt(50),
			wantErr:    true,
			errContain: "余额不足",
		},
		{
			name: "可用余额不足_有冻结",
			wallet: &entity.Wallet{
				ID:            1,
				UserID:        1,
				Currency:      "USDT",
				Balance:       decimal.NewFromInt(100),
				FrozenBalance: decimal.NewFromInt(80),
			},
			amount:     decimal.NewFromInt(30),
			wantErr:    true,
			errContain: "余额不足",
		},
		{
			name:       "钱包不存在",
			wallet:     nil,
			amount:     decimal.NewFromInt(10),
			wantErr:    true,
			errContain: "不存在",
		},
		{
			name:       "查询失败",
			wallet:     nil,
			findErr:    errors.New("db error"),
			amount:     decimal.NewFromInt(10),
			wantErr:    true,
			errContain: "查询钱包失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, walletRepo := newTestWalletService()
			ctx := context.Background()

			walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(tt.wallet, tt.findErr)

			err := svc.CheckBalance(ctx, 1, "USDT", tt.amount)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
