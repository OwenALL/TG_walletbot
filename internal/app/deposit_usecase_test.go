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

// newTestDepositUseCase 创建测试用的 DepositUseCase 及其 mock 依赖
func newTestDepositUseCase() (*DepositUseCase, *mocks.MockDepositRepository, *mocks.MockWalletRepository, *mocks.MockTransactionRepository, *mocks.MockSystemConfigRepository) {
	depositRepo := new(mocks.MockDepositRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	configRepo := new(mocks.MockSystemConfigRepository)

	depositSvc := service.NewDepositService(depositRepo, walletRepo, txRepo, configRepo)
	walletSvc := service.NewWalletService(walletRepo)

	uc := NewDepositUseCase(depositSvc, walletSvc, zap.NewNop())
	return uc, depositRepo, walletRepo, txRepo, configRepo
}

// ==================== GetDepositInfo 测试 ====================

func TestDepositUseCase_GetDepositInfo_成功(t *testing.T) {
	uc, _, walletRepo, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", DepositAddress: "TAddr_deposit_123"}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(wallet, nil)

	result, err := uc.GetDepositInfo(ctx, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TAddr_deposit_123", result.Address)
	assert.NotNil(t, result.QRCode) // QR Code 应被成功生成
}

func TestDepositUseCase_GetDepositInfo_无充值地址(t *testing.T) {
	uc, _, walletRepo, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", DepositAddress: ""}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(wallet, nil)

	result, err := uc.GetDepositInfo(ctx, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.Address)
}

func TestDepositUseCase_GetDepositInfo_查询失败(t *testing.T) {
	uc, _, walletRepo, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(nil, errors.New("db error"))

	result, err := uc.GetDepositInfo(ctx, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== ProcessChainDeposit 测试 ====================

func TestDepositUseCase_ProcessChainDeposit_新充值待确认(t *testing.T) {
	uc, depositRepo, walletRepo, _, configRepo := newTestDepositUseCase()
	ctx := context.Background()

	// 新交易，确认数不足
	depositRepo.On("FindByTxHash", ctx, "tx_001").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT"}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)

	result, err := uc.ProcessChainDeposit(ctx, "tx_001", "TFrom", "TAddr", "USDT", decimal.NewFromInt(50), 1)

	assert.NoError(t, err)
	assert.Nil(t, result) // 未入账，无通知
}

func TestDepositUseCase_ProcessChainDeposit_入账成功发送通知(t *testing.T) {
	uc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositUseCase()
	ctx := context.Background()

	// 新交易，确认数足够
	depositRepo.On("FindByTxHash", ctx, "tx_002").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	depositRepo.On("UpdateStatus", ctx, mock.AnythingOfType("uint64"), int8(entity.DepositStatusCredited)).Return(nil)

	// 获取入账后余额 (通知用)
	updatedWallets := []*entity.Wallet{
		{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(150)},
	}
	walletRepo.On("FindByUserID", ctx, uint64(10)).Return(updatedWallets, nil)

	result, err := uc.ProcessChainDeposit(ctx, "tx_002", "TFrom", "TAddr", "USDT", decimal.NewFromInt(50), 5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(10), result.UserID)
	assert.Equal(t, "USDT", result.Currency)
	assert.True(t, decimal.NewFromInt(50).Equal(result.Amount))
}

func TestDepositUseCase_ProcessChainDeposit_处理失败(t *testing.T) {
	uc, depositRepo, _, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_err").Return(nil, errors.New("db error"))

	result, err := uc.ProcessChainDeposit(ctx, "tx_err", "TFrom", "TAddr", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDepositUseCase_ProcessChainDeposit_已入账不通知(t *testing.T) {
	uc, depositRepo, _, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	// 已入账且已通知的充值
	existing := &entity.Deposit{
		ID:       1,
		Status:   entity.DepositStatusCredited,
		Notified: true,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_dup").Return(existing, nil)

	result, err := uc.ProcessChainDeposit(ctx, "tx_dup", "TFrom", "TAddr", "USDT", decimal.NewFromInt(50), 10)

	assert.NoError(t, err)
	assert.Nil(t, result) // 已通知过，不再发通知
}

func TestDepositUseCase_ProcessChainDeposit_入账后查询余额失败(t *testing.T) {
	uc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositUseCase()
	ctx := context.Background()

	// 新交易，确认数足够 -> 入账成功 -> 查询余额失败
	depositRepo.On("FindByTxHash", ctx, "tx_bal_err").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	depositRepo.On("UpdateStatus", ctx, mock.AnythingOfType("uint64"), int8(entity.DepositStatusCredited)).Return(nil)

	// 查询余额失败
	walletRepo.On("FindByUserID", ctx, uint64(10)).Return(nil, errors.New("balance error"))

	result, err := uc.ProcessChainDeposit(ctx, "tx_bal_err", "TFrom", "TAddr", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== ListPendingDeposits 测试 ====================

func TestDepositUseCase_ListPendingDeposits_成功(t *testing.T) {
	uc, depositRepo, _, _, _ := newTestDepositUseCase()
	ctx := context.Background()

	pending := []*entity.Deposit{
		{ID: 1, Status: entity.DepositStatusPending},
	}
	depositRepo.On("ListPending", ctx).Return(pending, nil)

	result, err := uc.ListPendingDeposits(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

// ==================== 构造函数测试 ====================

func TestNewDepositUseCase(t *testing.T) {
	depositSvc := service.NewDepositService(nil, nil, nil, nil)
	walletSvc := service.NewWalletService(nil)

	uc := NewDepositUseCase(depositSvc, walletSvc, zap.NewNop())
	assert.NotNil(t, uc)
}
