package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service/mocks"
)

// 注意: TransferService.ExecuteTransfer 使用了 gorm.DB 事务和内部 txWalletRepo/txTransferRepo/txTransactionRepo，
// 无法直接通过 mock port 接口测试事务方法。
// 这里测试非事务方法: CreateReceiveRequest, RejectReceiveRequest, GetTransferByID, FindRecipient

// newTestTransferServiceDeps 创建 TransferService 非事务方法测试所需的 mock
// 注意: TransferService 依赖 *gorm.DB，但对于非事务方法我们传 nil (测试中不会调用事务)
func newTestTransferServiceDeps() (*TransferService, *mocks.MockWalletRepository, *mocks.MockTransferRepository, *mocks.MockTransactionRepository, *mocks.MockUserRepository) {
	walletRepo := new(mocks.MockWalletRepository)
	transferRepo := new(mocks.MockTransferRepository)
	txRepo := new(mocks.MockTransactionRepository)
	userRepo := new(mocks.MockUserRepository)

	svc := &TransferService{
		db:              nil, // 非事务方法不需要 db
		walletRepo:      walletRepo,
		transferRepo:    transferRepo,
		transactionRepo: txRepo,
		userRepo:        userRepo,
		logger:          nil, // 非事务方法日志不关键
	}
	return svc, walletRepo, transferRepo, txRepo, userRepo
}

// ==================== CreateReceiveRequest 测试 ====================

func TestTransferService_CreateReceiveRequest_成功(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transfer")).Return(nil)

	transfer, err := svc.CreateReceiveRequest(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.NoError(t, err)
	assert.NotNil(t, transfer)
	// fromUserID=1 (发起收款请求的人), toUserID=2 (被请求付款的人)
	// 实际代码: FromUserID = toUserID(付款人=2), ToUserID = fromUserID(收款人=1)
	assert.Equal(t, uint64(2), transfer.FromUserID) // 付款人
	assert.Equal(t, uint64(1), transfer.ToUserID)   // 收款人
	assert.Equal(t, int8(entity.TransferTypeRequest), transfer.Type)
	assert.Equal(t, int8(entity.TransferStatusPending), transfer.Status)
	assert.Equal(t, decimal.NewFromInt(100), transfer.Amount)
	assert.Equal(t, "USDT", transfer.Currency)
}

func TestTransferService_CreateReceiveRequest_创建失败(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transfer")).Return(errors.New("db error"))

	transfer, err := svc.CreateReceiveRequest(ctx, 1, 2, "USDT", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, transfer)
	assert.Contains(t, err.Error(), "创建收款请求失败")
}

func TestTransferService_CreateReceiveRequest_不同币种(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		amount   decimal.Decimal
	}{
		{"USDT转账", "USDT", decimal.NewFromInt(100)},
		{"TRX转账", "TRX", decimal.NewFromInt(500)},
		{"CNY转账", "CNY", decimal.NewFromFloat(88.88)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
			ctx := context.Background()

			transferRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transfer")).Return(nil)

			transfer, err := svc.CreateReceiveRequest(ctx, 1, 2, tt.currency, tt.amount)

			assert.NoError(t, err)
			assert.NotNil(t, transfer)
			assert.Equal(t, tt.currency, transfer.Currency)
			assert.True(t, tt.amount.Equal(transfer.Amount))
		})
	}
}

// ==================== RejectReceiveRequest 测试 ====================

func TestTransferService_RejectReceiveRequest_成功(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusPending}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)
	transferRepo.On("Update", ctx, mock.AnythingOfType("*entity.Transfer")).Return(nil)

	err := svc.RejectReceiveRequest(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.TransferStatusRejected), transfer.Status)
	assert.NotNil(t, transfer.CompletedAt)
}

func TestTransferService_RejectReceiveRequest_记录不存在(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.RejectReceiveRequest(ctx, 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestTransferService_RejectReceiveRequest_已处理(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusCompleted}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)

	err := svc.RejectReceiveRequest(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已处理")
}

func TestTransferService_RejectReceiveRequest_查询失败(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.RejectReceiveRequest(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询转账记录失败")
}

func TestTransferService_RejectReceiveRequest_更新失败(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transfer := &entity.Transfer{ID: 1, Status: entity.TransferStatusPending}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(transfer, nil)
	transferRepo.On("Update", ctx, mock.AnythingOfType("*entity.Transfer")).Return(errors.New("db error"))

	err := svc.RejectReceiveRequest(ctx, 1)

	assert.Error(t, err)
}

// ==================== GetTransferByID 测试 ====================

func TestTransferService_GetTransferByID_成功(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	expected := &entity.Transfer{ID: 1, FromUserID: 10, ToUserID: 20, Currency: "USDT", Amount: decimal.NewFromInt(100)}
	transferRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := svc.GetTransferByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTransferService_GetTransferByID_不存在(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.GetTransferByID(ctx, 999)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestTransferService_GetTransferByID_查询失败(t *testing.T) {
	svc, _, transferRepo, _, _ := newTestTransferServiceDeps()
	ctx := context.Background()

	transferRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetTransferByID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== FindRecipient 测试 ====================

func TestTransferService_FindRecipient(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		user       *entity.User
		findErr    error
		wantUser   bool
		wantErr    bool
	}{
		{
			name:       "通过用户名查找_成功",
			identifier: "testuser",
			user:       &entity.User{ID: 1, Username: "testuser"},
			wantUser:   true,
		},
		{
			name:       "通过@用户名查找_成功",
			identifier: "@testuser",
			user:       &entity.User{ID: 1, Username: "testuser"},
			wantUser:   true,
		},
		{
			name:       "用户不存在",
			identifier: "nobody",
			user:       nil,
			wantUser:   false,
		},
		{
			name:       "查询失败",
			identifier: "testuser",
			findErr:    errors.New("db error"),
			wantErr:    true,
		},
		{
			name:       "空标识符",
			identifier: "",
			wantUser:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, userRepo := newTestTransferServiceDeps()
			ctx := context.Background()

			// 解析用户名 (去除 @ 前缀)
			username := tt.identifier
			if len(username) > 0 && username[0] == '@' {
				username = username[1:]
			}
			if username != "" {
				userRepo.On("FindByUsername", ctx, username).Return(tt.user, tt.findErr)
			}

			result, err := svc.FindRecipient(ctx, tt.identifier)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantUser {
				assert.NotNil(t, result)
			} else if !tt.wantErr {
				assert.Nil(t, result)
			}
		})
	}
}

// ==================== NewTransferService 构造测试 ====================

func TestNewTransferService(t *testing.T) {
	walletRepo := new(mocks.MockWalletRepository)
	transferRepo := new(mocks.MockTransferRepository)
	txRepo := new(mocks.MockTransactionRepository)
	userRepo := new(mocks.MockUserRepository)
	logger := zap.NewNop()

	svc := NewTransferService(nil, walletRepo, transferRepo, txRepo, userRepo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, walletRepo, svc.walletRepo)
	assert.Equal(t, transferRepo, svc.transferRepo)
	assert.Equal(t, txRepo, svc.transactionRepo)
	assert.Equal(t, userRepo, svc.userRepo)
	assert.Equal(t, logger, svc.logger)
}
