package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestDepositService 创建测试用的 DepositService 及其所有 mock 依赖
func newTestDepositService() (*DepositService, *mocks.MockDepositRepository, *mocks.MockWalletRepository, *mocks.MockTransactionRepository, *mocks.MockSystemConfigRepository) {
	depositRepo := new(mocks.MockDepositRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	configRepo := new(mocks.MockSystemConfigRepository)
	svc := NewDepositService(depositRepo, walletRepo, txRepo, configRepo)
	return svc, depositRepo, walletRepo, txRepo, configRepo
}

// ==================== GetRequiredConfirmations 测试 ====================

func TestDepositService_GetRequiredConfirmations_从配置读取(t *testing.T) {
	svc, _, _, _, configRepo := newTestDepositService()
	ctx := context.Background()

	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("6", nil)

	result := svc.GetRequiredConfirmations(ctx)

	assert.Equal(t, 6, result)
}

func TestDepositService_GetRequiredConfirmations_配置不存在使用默认(t *testing.T) {
	svc, _, _, _, configRepo := newTestDepositService()
	ctx := context.Background()

	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("", errors.New("not found"))

	result := svc.GetRequiredConfirmations(ctx)

	assert.Equal(t, 3, result) // 默认值
}

func TestDepositService_GetRequiredConfirmations_配置值无效使用默认(t *testing.T) {
	svc, _, _, _, configRepo := newTestDepositService()
	ctx := context.Background()

	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("abc", nil)

	result := svc.GetRequiredConfirmations(ctx)

	assert.Equal(t, 3, result)
}

// ==================== ProcessDeposit 测试 ====================

func TestDepositService_ProcessDeposit_新充值_确认数不足(t *testing.T) {
	svc, depositRepo, walletRepo, _, configRepo := newTestDepositService()
	ctx := context.Background()

	// 交易不存在
	depositRepo.On("FindByTxHash", ctx, "tx_hash_001").Return(nil, nil)
	// 查找充值地址对应的钱包
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	// 需要 3 次确认
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	// 创建充值记录 (确认数 1 < 3，状态为 Pending)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_001", "TFrom456", "TAddr123", "USDT", decimal.NewFromInt(50), 1)

	assert.NoError(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, deposit)
	assert.Equal(t, int8(entity.DepositStatusPending), deposit.Status)
	assert.Equal(t, 1, deposit.Confirmations)
}

func TestDepositService_ProcessDeposit_新充值_确认数达标直接入账(t *testing.T) {
	svc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_002").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)

	// 入账操作
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	depositRepo.On("UpdateStatus", ctx, mock.AnythingOfType("uint64"), int8(entity.DepositStatusCredited)).Return(nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_002", "TFrom456", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.NoError(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, deposit)
	assert.Equal(t, int8(entity.DepositStatusCredited), deposit.Status)
	walletRepo.AssertCalled(t, "UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50))
}

func TestDepositService_ProcessDeposit_重复交易哈希_已入账(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:     1,
		Status: entity.DepositStatusCredited,
		TxHash: "tx_hash_dup",
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_dup").Return(existing, nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_dup", "TFrom", "TTo", "USDT", decimal.NewFromInt(50), 10)

	assert.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, existing, deposit)
}

func TestDepositService_ProcessDeposit_重复交易哈希_更新确认数(t *testing.T) {
	svc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:            1,
		UserID:        10,
		Currency:      "USDT",
		Amount:        decimal.NewFromInt(50),
		ToAddress:     "TAddr123",
		TxHash:        "tx_hash_existing",
		Confirmations: 1,
		Status:        entity.DepositStatusPending,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_existing").Return(existing, nil)
	depositRepo.On("UpdateConfirmations", ctx, uint64(1), 5).Return(nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("UpdateStatus", ctx, uint64(1), int8(entity.DepositStatusConfirmed)).Return(nil)

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	depositRepo.On("UpdateStatus", ctx, uint64(1), int8(entity.DepositStatusCredited)).Return(nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_existing", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.NoError(t, err)
	assert.False(t, isNew)
	assert.NotNil(t, deposit)
	assert.Equal(t, 5, deposit.Confirmations)
}

func TestDepositService_ProcessDeposit_重复交易_确认数未增加(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:            1,
		Confirmations: 5,
		Status:        entity.DepositStatusPending,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_same").Return(existing, nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_same", "TFrom", "TTo", "USDT", decimal.NewFromInt(50), 3) // 3 <= 5

	assert.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, existing, deposit)
}

func TestDepositService_ProcessDeposit_未知充值地址(t *testing.T) {
	svc, depositRepo, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_unknown").Return(nil, nil)
	walletRepo.On("FindByDepositAddress", ctx, "unknown_addr").Return(nil, nil)

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_unknown", "TFrom", "unknown_addr", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.Nil(t, deposit)
	assert.Contains(t, err.Error(), "未知的充值地址")
}

func TestDepositService_ProcessDeposit_查询交易哈希失败(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_err").Return(nil, errors.New("db error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_err", "TFrom", "TTo", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.Nil(t, deposit)
	assert.Contains(t, err.Error(), "查询充值记录失败")
}

func TestDepositService_ProcessDeposit_创建记录失败(t *testing.T) {
	svc, depositRepo, walletRepo, _, configRepo := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_fail").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT"}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(errors.New("db create error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 1)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.Nil(t, deposit)
	assert.Contains(t, err.Error(), "创建充值记录失败")
}

// ==================== GetUserDepositAddress 测试 ====================

func TestDepositService_GetUserDepositAddress_成功(t *testing.T) {
	svc, _, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", DepositAddress: "TAddr_deposit_123"}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(wallet, nil)

	addr, err := svc.GetUserDepositAddress(ctx, 10)

	assert.NoError(t, err)
	assert.Equal(t, "TAddr_deposit_123", addr)
}

func TestDepositService_GetUserDepositAddress_钱包不存在(t *testing.T) {
	svc, _, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(nil, nil)

	addr, err := svc.GetUserDepositAddress(ctx, 10)

	assert.Error(t, err)
	assert.Empty(t, addr)
	assert.Contains(t, err.Error(), "不存在")
}

// ==================== ListPendingDeposits 测试 ====================

func TestDepositService_ListPendingDeposits_成功(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	pending := []*entity.Deposit{
		{ID: 1, Status: entity.DepositStatusPending},
		{ID: 2, Status: entity.DepositStatusPending},
	}
	depositRepo.On("ListPending", ctx).Return(pending, nil)

	result, err := svc.ListPendingDeposits(ctx)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestDepositService_ListPendingDeposits_空列表(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("ListPending", ctx).Return([]*entity.Deposit{}, nil)

	result, err := svc.ListPendingDeposits(ctx)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

// ==================== SetUserDepositAddress 测试 ====================

func TestDepositService_SetUserDepositAddress_成功(t *testing.T) {
	svc, _, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	walletRepo.On("UpdateDepositAddress", ctx, uint64(10), "TAddr_new_123").Return(nil)

	err := svc.SetUserDepositAddress(ctx, 10, "TAddr_new_123")

	assert.NoError(t, err)
	walletRepo.AssertCalled(t, "UpdateDepositAddress", ctx, uint64(10), "TAddr_new_123")
}

func TestDepositService_SetUserDepositAddress_更新失败(t *testing.T) {
	svc, _, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	walletRepo.On("UpdateDepositAddress", ctx, uint64(10), "TAddr_new_123").Return(errors.New("db error"))

	err := svc.SetUserDepositAddress(ctx, 10, "TAddr_new_123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "更新充值地址失败")
}

// ==================== 额外边界测试 ====================

func TestDepositService_ProcessDeposit_入账后更新余额失败(t *testing.T) {
	svc, depositRepo, walletRepo, _, configRepo := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_credit_fail").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)
	// 入账时更新余额失败
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(errors.New("balance update error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_credit_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, deposit) // deposit 已创建但入账失败
	assert.Contains(t, err.Error(), "充值入账失败")
}

func TestDepositService_ProcessDeposit_入账后创建流水失败(t *testing.T) {
	svc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_tx_fail").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(errors.New("tx create error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_tx_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, deposit)
	assert.Contains(t, err.Error(), "创建充值流水失败")
}

func TestDepositService_GetUserDepositAddress_查询失败(t *testing.T) {
	svc, _, walletRepo, _, _ := newTestDepositService()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), entity.CurrencyUSDT).Return(nil, errors.New("db error"))

	addr, err := svc.GetUserDepositAddress(ctx, 10)

	assert.Error(t, err)
	assert.Empty(t, addr)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

// ==================== updateConfirmations 边界测试 ====================

func TestDepositService_ProcessDeposit_重复交易_更新确认数失败(t *testing.T) {
	svc, depositRepo, _, _, _ := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:            1,
		UserID:        10,
		Currency:      "USDT",
		Amount:        decimal.NewFromInt(50),
		ToAddress:     "TAddr123",
		TxHash:        "tx_hash_upd_fail",
		Confirmations: 1,
		Status:        entity.DepositStatusPending,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_upd_fail").Return(existing, nil)
	depositRepo.On("UpdateConfirmations", ctx, uint64(1), 5).Return(errors.New("update conf error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_upd_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.NotNil(t, deposit)
	assert.Contains(t, err.Error(), "更新确认数失败")
}

func TestDepositService_ProcessDeposit_重复交易_更新状态失败(t *testing.T) {
	svc, depositRepo, _, _, configRepo := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:            1,
		UserID:        10,
		Currency:      "USDT",
		Amount:        decimal.NewFromInt(50),
		ToAddress:     "TAddr123",
		TxHash:        "tx_hash_status_fail",
		Confirmations: 1,
		Status:        entity.DepositStatusPending,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_status_fail").Return(existing, nil)
	depositRepo.On("UpdateConfirmations", ctx, uint64(1), 5).Return(nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	// 确认数达标后更新状态失败
	depositRepo.On("UpdateStatus", ctx, uint64(1), int8(entity.DepositStatusConfirmed)).Return(errors.New("status update error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_status_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.NotNil(t, deposit)
	assert.Contains(t, err.Error(), "更新充值状态失败")
}

func TestDepositService_ProcessDeposit_入账后更新状态为已到账失败(t *testing.T) {
	svc, depositRepo, walletRepo, txRepo, configRepo := newTestDepositService()
	ctx := context.Background()

	depositRepo.On("FindByTxHash", ctx, "tx_hash_status_credited_fail").Return(nil, nil)
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(100)}
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(wallet, nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("Create", ctx, mock.AnythingOfType("*entity.Deposit")).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(50)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	// 最后一步更新状态为 Credited 失败
	depositRepo.On("UpdateStatus", ctx, mock.AnythingOfType("uint64"), int8(entity.DepositStatusCredited)).Return(errors.New("status update error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_status_credited_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.True(t, isNew)
	assert.NotNil(t, deposit)
	assert.Contains(t, err.Error(), "更新充值状态失败")
}

func TestDepositService_ProcessDeposit_重复交易_入账时查询钱包失败(t *testing.T) {
	svc, depositRepo, walletRepo, _, configRepo := newTestDepositService()
	ctx := context.Background()

	existing := &entity.Deposit{
		ID:            1,
		UserID:        10,
		Currency:      "USDT",
		Amount:        decimal.NewFromInt(50),
		ToAddress:     "TAddr123",
		TxHash:        "tx_hash_wallet_fail",
		Confirmations: 1,
		Status:        entity.DepositStatusPending,
	}
	depositRepo.On("FindByTxHash", ctx, "tx_hash_wallet_fail").Return(existing, nil)
	depositRepo.On("UpdateConfirmations", ctx, uint64(1), 5).Return(nil)
	configRepo.On("Get", ctx, entity.ConfigDepositConfirmations).Return("3", nil)
	depositRepo.On("UpdateStatus", ctx, uint64(1), int8(entity.DepositStatusConfirmed)).Return(nil)
	// 入账时查询钱包失败
	walletRepo.On("FindByDepositAddress", ctx, "TAddr123").Return(nil, errors.New("wallet query error"))

	deposit, isNew, err := svc.ProcessDeposit(ctx, "tx_hash_wallet_fail", "TFrom", "TAddr123", "USDT", decimal.NewFromInt(50), 5)

	assert.Error(t, err)
	assert.False(t, isNew)
	assert.NotNil(t, deposit)
	assert.Contains(t, err.Error(), "查询钱包失败")
}
