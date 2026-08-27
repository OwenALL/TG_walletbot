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

// newTestTransferUseCase 创建测试用的 TransferUseCase 及其 mock 依赖
func newTestTransferUseCase() (*TransferUseCase, *mocks.MockUserRepository, *mocks.MockWalletRepository, *mocks.MockTransferRepository, *mocks.MockTransactionRepository, *mocks.MockUserSettingsRepository, *mocks.MockSystemConfigRepository) {
	userRepo := new(mocks.MockUserRepository)
	walletRepo := new(mocks.MockWalletRepository)
	transferRepo := new(mocks.MockTransferRepository)
	txRepo := new(mocks.MockTransactionRepository)
	settingsRepo := new(mocks.MockUserSettingsRepository)
	configRepo := new(mocks.MockSystemConfigRepository)

	userSvc := service.NewUserService(userRepo, walletRepo, settingsRepo, configRepo)
	walletSvc := service.NewWalletService(walletRepo)
	transferSvc := service.NewTransferService(nil, walletRepo, transferRepo, txRepo, userRepo, zap.NewNop())

	uc := NewTransferUseCase(transferSvc, walletSvc, userSvc, zap.NewNop())
	return uc, userRepo, walletRepo, transferRepo, txRepo, settingsRepo, configRepo
}

// ==================== FindRecipientByUsername 测试 ====================

func TestTransferUseCase_FindRecipientByUsername_成功(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	expected := &entity.User{ID: 2, Username: "recipient"}
	userRepo.On("FindByUsername", ctx, "recipient").Return(expected, nil)

	result, err := uc.FindRecipientByUsername(ctx, "recipient")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferUseCase_FindRecipientByUsername_不存在(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByUsername", ctx, "nobody").Return(nil, nil)

	result, err := uc.FindRecipientByUsername(ctx, "nobody")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ==================== FindRecipientByTelegramID 测试 ====================

func TestTransferUseCase_FindRecipientByTelegramID_成功(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	expected := &entity.User{ID: 2, TelegramID: 99999}
	userRepo.On("FindByTelegramID", ctx, int64(99999)).Return(expected, nil)

	result, err := uc.FindRecipientByTelegramID(ctx, 99999)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferUseCase_FindRecipientByTelegramID_查询失败(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByTelegramID", ctx, int64(99999)).Return(nil, errors.New("db error"))

	result, err := uc.FindRecipientByTelegramID(ctx, 99999)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== FindRecipientByID 测试 ====================

func TestTransferUseCase_FindRecipientByID_成功(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	expected := &entity.User{ID: 2, TelegramID: 12345}
	userRepo.On("FindByTelegramID", ctx, int64(12345)).Return(expected, nil)

	result, err := uc.FindRecipientByID(ctx, "12345")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferUseCase_FindRecipientByID_非数字(t *testing.T) {
	uc, _, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	result, err := uc.FindRecipientByID(ctx, "not_a_number")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ==================== ValidateTransferAmount 测试 ====================

func TestTransferUseCase_ValidateTransferAmount(t *testing.T) {
	uc, _, _, _, _, _, _ := newTestTransferUseCase()

	tests := []struct {
		name     string
		currency string
		amount   decimal.Decimal
		wantErr  bool
		errMsg   string
	}{
		{"USDT正常金额", "USDT", decimal.NewFromInt(10), false, ""},
		{"TRX正常金额", "TRX", decimal.NewFromInt(100), false, ""},
		{"CNY正常金额", "CNY", decimal.NewFromInt(50), false, ""},
		{"金额为0", "USDT", decimal.Zero, true, "大于 0"},
		{"金额为负", "USDT", decimal.NewFromInt(-1), true, "大于 0"},
		{"USDT低于最低", "USDT", decimal.NewFromFloat(0.001), true, "最低转账金额"},
		{"TRX低于最低", "TRX", decimal.NewFromFloat(0.5), true, "最低转账金额"},
		{"不支持的币种", "BTC", decimal.NewFromInt(1), true, "不支持的币种"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.ValidateTransferAmount(tt.currency, tt.amount)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== ExecuteTransfer 测试 ====================

func TestTransferUseCase_ExecuteTransfer_转给自己(t *testing.T) {
	uc, _, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	result, err := uc.ExecuteTransfer(ctx, 1, 1, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "自己")
}

func TestTransferUseCase_ExecuteTransfer_收款人不存在(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(2)).Return(nil, nil)

	result, err := uc.ExecuteTransfer(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "收款人")
}

func TestTransferUseCase_ExecuteTransfer_收款人被冻结(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	receiver := &entity.User{ID: 2, Status: int8(entity.UserStatusFrozen)}
	userRepo.On("FindByID", ctx, uint64(2)).Return(receiver, nil)

	result, err := uc.ExecuteTransfer(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "冻结")
}

func TestTransferUseCase_ExecuteTransfer_查询收款人失败(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(2)).Return(nil, errors.New("db error"))

	result, err := uc.ExecuteTransfer(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== CreateReceiveRequest 测试 ====================

func TestTransferUseCase_CreateReceiveRequest_成功(t *testing.T) {
	uc, userRepo, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	payer := &entity.User{ID: 2, Status: int8(entity.UserStatusActive)}
	userRepo.On("FindByID", ctx, uint64(2)).Return(payer, nil)
	transferRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transfer")).Return(nil)

	result, err := uc.CreateReceiveRequest(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTransferUseCase_CreateReceiveRequest_向自己收款(t *testing.T) {
	uc, _, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	result, err := uc.CreateReceiveRequest(ctx, 1, 1, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "自己")
}

func TestTransferUseCase_CreateReceiveRequest_付款人不存在(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(2)).Return(nil, nil)

	result, err := uc.CreateReceiveRequest(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "收款人")
}

func TestTransferUseCase_CreateReceiveRequest_查询付款人失败(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	userRepo.On("FindByID", ctx, uint64(2)).Return(nil, errors.New("db error"))

	result, err := uc.CreateReceiveRequest(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== ExecuteReceivePayment 测试 ====================

func TestTransferUseCase_ExecuteReceivePayment_记录不存在(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(1)).Return(nil, nil)

	result, err := uc.ExecuteReceivePayment(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestTransferUseCase_ExecuteReceivePayment_已处理(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusCompleted, Type: entity.TransferTypeRequest}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)

	result, err := uc.ExecuteReceivePayment(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "已处理")
}

func TestTransferUseCase_ExecuteReceivePayment_非收款请求(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusPending, Type: entity.TransferTypeDirect}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)

	result, err := uc.ExecuteReceivePayment(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "非收款请求")
}

func TestTransferUseCase_ExecuteReceivePayment_查询失败(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := uc.ExecuteReceivePayment(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetBalance 测试 ====================

func TestTransferUseCase_GetBalance_成功(t *testing.T) {
	uc, _, walletRepo, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(200)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	balance, err := uc.GetBalance(ctx, 1, "USDT")

	assert.NoError(t, err)
	assert.True(t, decimal.NewFromInt(200).Equal(balance))
}

func TestTransferUseCase_GetBalance_查询失败(t *testing.T) {
	uc, _, walletRepo, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(nil, errors.New("db error"))

	_, err := uc.GetBalance(ctx, 1, "USDT")

	assert.Error(t, err)
}

// ==================== 委托方法测试 ====================

func TestTransferUseCase_GetTransferByID_成功(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	expected := &entity.Transfer{ID: 1, Currency: "USDT"}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := uc.GetTransferByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferUseCase_GetUserByID_成功(t *testing.T) {
	uc, userRepo, _, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	expected := &entity.User{ID: 1}
	userRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := uc.GetUserByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferUseCase_RejectReceiveRequest_成功(t *testing.T) {
	uc, _, _, transferRepo, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusPending}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)
	transferRepo.On("Update", ctx, mock.AnythingOfType("*entity.Transfer")).Return(nil)

	err := uc.RejectReceiveRequest(ctx, 1)

	assert.NoError(t, err)
}

func TestTransferUseCase_CheckBalance_成功(t *testing.T) {
	uc, _, walletRepo, _, _, _, _ := newTestTransferUseCase()
	ctx := context.Background()

	wallet := &entity.Wallet{ID: 1, UserID: 1, Currency: "USDT", Balance: decimal.NewFromInt(200), FrozenBalance: decimal.Zero}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(wallet, nil)

	err := uc.CheckBalance(ctx, 1, "USDT", decimal.NewFromInt(100))

	assert.NoError(t, err)
}

// ==================== 构造函数测试 ====================

func TestNewTransferUseCase(t *testing.T) {
	transferSvc := service.NewTransferService(nil, nil, nil, nil, nil, zap.NewNop())
	walletSvc := service.NewWalletService(nil)
	userSvc := service.NewUserService(nil, nil, nil, nil)

	uc := NewTransferUseCase(transferSvc, walletSvc, userSvc, zap.NewNop())
	assert.NotNil(t, uc)
}
