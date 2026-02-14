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

// newTestWithdrawalService 创建测试用的 WithdrawalService 及其所有 mock 依赖
func newTestWithdrawalService() (*WithdrawalService, *mocks.MockWithdrawalRepository, *mocks.MockWalletRepository, *mocks.MockTransactionRepository, *mocks.MockSystemConfigRepository) {
	withdrawalRepo := new(mocks.MockWithdrawalRepository)
	walletRepo := new(mocks.MockWalletRepository)
	txRepo := new(mocks.MockTransactionRepository)
	configRepo := new(mocks.MockSystemConfigRepository)
	svc := NewWithdrawalService(withdrawalRepo, walletRepo, txRepo, configRepo)
	return svc, withdrawalRepo, walletRepo, txRepo, configRepo
}

// setupDefaultUSDTConfig 设置默认 USDT 提币配置 mock
func setupDefaultUSDTConfig(configRepo *mocks.MockSystemConfigRepository) {
	ctx := mock.Anything
	configRepo.On("Get", ctx, entity.ConfigWithdrawUSDTMin).Return("5", nil)
	configRepo.On("Get", ctx, entity.ConfigWithdrawUSDTMax).Return("50000", nil)
	configRepo.On("Get", ctx, entity.ConfigWithdrawUSDTDailyMax).Return("100000", nil)
	configRepo.On("Get", ctx, entity.ConfigWithdrawUSDTFee).Return("1", nil)
	configRepo.On("Get", ctx, entity.ConfigWithdrawUSDTAutoThreshold).Return("1000", nil)
}

// ==================== ValidateAmount 测试 ====================

func TestWithdrawalService_ValidateAmount(t *testing.T) {
	tests := []struct {
		name        string
		amount      decimal.Decimal
		wallet      *entity.Wallet
		walletErr   error
		todaySpent  decimal.Decimal
		todayErr    error
		wantErr     bool
		errContains string
	}{
		{
			name:   "验证通过",
			amount: decimal.NewFromInt(100),
			wallet: &entity.Wallet{
				ID: 1, UserID: 1, Currency: "USDT",
				Balance: decimal.NewFromInt(200), FrozenBalance: decimal.Zero,
			},
			todaySpent: decimal.Zero,
			wantErr:    false,
		},
		{
			name:        "低于最低限额",
			amount:      decimal.NewFromInt(1), // min = 5
			wantErr:     true,
			errContains: "最低提币金额",
		},
		{
			name:        "超过单笔上限",
			amount:      decimal.NewFromInt(60000), // max = 50000
			wantErr:     true,
			errContains: "单笔提币上限",
		},
		{
			name:   "余额不足_金额加手续费超出",
			amount: decimal.NewFromInt(100),
			wallet: &entity.Wallet{
				ID: 1, UserID: 1, Currency: "USDT",
				Balance: decimal.NewFromInt(100), FrozenBalance: decimal.Zero,
			}, // 需要 100 + 1(fee) = 101，余额只有 100
			wantErr:     true,
			errContains: "余额不足",
		},
		{
			name:   "超过日限额",
			amount: decimal.NewFromInt(100),
			wallet: &entity.Wallet{
				ID: 1, UserID: 1, Currency: "USDT",
				Balance: decimal.NewFromInt(200), FrozenBalance: decimal.Zero,
			},
			todaySpent:  decimal.NewFromInt(99950), // 99950 + 100 > 100000
			wantErr:     true,
			errContains: "今日提币限额",
		},
		{
			name:        "钱包不存在",
			amount:      decimal.NewFromInt(100),
			wallet:      nil,
			wantErr:     true,
			errContains: "不存在",
		},
		{
			name:        "查询钱包失败",
			amount:      decimal.NewFromInt(100),
			wallet:      nil,
			walletErr:   errors.New("db error"),
			wantErr:     true,
			errContains: "查询钱包失败",
		},
		{
			name:   "查询今日提币总额失败",
			amount: decimal.NewFromInt(100),
			wallet: &entity.Wallet{
				ID: 1, UserID: 1, Currency: "USDT",
				Balance: decimal.NewFromInt(200), FrozenBalance: decimal.Zero,
			},
			todayErr:    errors.New("db error"),
			wantErr:     true,
			errContains: "查询今日提币总额失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, withdrawalRepo, walletRepo, _, configRepo := newTestWithdrawalService()
			ctx := context.Background()
			setupDefaultUSDTConfig(configRepo)

			// 只在金额通过 min/max 验证后才需要钱包和日限额 mock
			if tt.amount.GreaterThanOrEqual(decimal.NewFromInt(5)) && tt.amount.LessThanOrEqual(decimal.NewFromInt(50000)) {
				walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(1), "USDT").Return(tt.wallet, tt.walletErr)
				if tt.wallet != nil && tt.walletErr == nil {
					totalNeeded := tt.amount.Add(decimal.NewFromInt(1)) // fee = 1
					if tt.wallet.HasSufficientBalance(totalNeeded) {
						withdrawalRepo.On("SumTodayByUserID", ctx, uint64(1), "USDT").Return(tt.todaySpent, tt.todayErr)
					}
				}
			}

			err := svc.ValidateAmount(ctx, 1, "USDT", tt.amount)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ==================== CreateWithdrawal 测试 ====================

func TestWithdrawalService_CreateWithdrawal_成功(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("FreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Create", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	result, err := svc.CreateWithdrawal(ctx, 10, "USDT", "TAddr_to_123", decimal.NewFromInt(100))

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(10), result.UserID)
	assert.Equal(t, decimal.NewFromInt(100), result.Amount)
	assert.Equal(t, decimal.NewFromInt(1), result.Fee)
	assert.Equal(t, decimal.NewFromInt(99), result.ActualAmount)
	assert.Equal(t, int8(entity.WithdrawalStatusPending), result.Status)
}

func TestWithdrawalService_CreateWithdrawal_钱包不存在(t *testing.T) {
	svc, _, walletRepo, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, nil)

	result, err := svc.CreateWithdrawal(ctx, 10, "USDT", "TAddr_to_123", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestWithdrawalService_CreateWithdrawal_冻结失败_余额不足(t *testing.T) {
	svc, _, walletRepo, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(50)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("FreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(errors.New("insufficient"))

	result, err := svc.CreateWithdrawal(ctx, 10, "USDT", "TAddr_to_123", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "余额不足")
}

func TestWithdrawalService_CreateWithdrawal_创建记录失败_自动解冻(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(1000)}
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("FreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Create", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(errors.New("db error"))
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)

	result, err := svc.CreateWithdrawal(ctx, 10, "USDT", "TAddr_to_123", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	walletRepo.AssertCalled(t, "UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100))
}

// ==================== ApproveWithdrawal 测试 ====================

func TestWithdrawalService_ApproveWithdrawal_成功(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusPending}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.ApproveWithdrawal(ctx, 1, 100)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusProcessing), withdrawal.Status)
	assert.NotNil(t, withdrawal.ReviewerID)
	assert.Equal(t, uint64(100), *withdrawal.ReviewerID)
	assert.NotNil(t, withdrawal.ReviewedAt)
}

func TestWithdrawalService_ApproveWithdrawal_记录不存在(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.ApproveWithdrawal(ctx, 999, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestWithdrawalService_ApproveWithdrawal_已处理(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusCompleted}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)

	err := svc.ApproveWithdrawal(ctx, 1, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已处理")
}

// ==================== CompleteWithdrawal 测试 ====================

func TestWithdrawalService_CompleteWithdrawal_成功(t *testing.T) {
	svc, withdrawalRepo, walletRepo, txRepo, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID:       1,
		UserID:   10,
		Currency: "USDT",
		Amount:   decimal.NewFromInt(100),
		Fee:      decimal.NewFromInt(1),
		Status:   entity.WithdrawalStatusProcessing,
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(500)}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(-100)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash_complete")

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusCompleted), withdrawal.Status)
	assert.Equal(t, "tx_hash_complete", withdrawal.TxHash)
	assert.NotNil(t, withdrawal.CompletedAt)
}

func TestWithdrawalService_CompleteWithdrawal_记录不存在(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.CompleteWithdrawal(ctx, 999, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestWithdrawalService_CompleteWithdrawal_钱包不存在(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100)}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, nil)

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// ==================== RejectWithdrawal 测试 ====================

func TestWithdrawalService_RejectWithdrawal_成功(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID:       1,
		UserID:   10,
		Currency: "USDT",
		Amount:   decimal.NewFromInt(100),
		Status:   entity.WithdrawalStatusPending,
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT"}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.RejectWithdrawal(ctx, 1, 100, "风险交易")

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusRejected), withdrawal.Status)
	assert.Equal(t, "风险交易", withdrawal.ReviewNote)
	assert.NotNil(t, withdrawal.ReviewerID)
	assert.NotNil(t, withdrawal.ReviewedAt)
	walletRepo.AssertCalled(t, "UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100))
}

func TestWithdrawalService_RejectWithdrawal_已处理(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, Status: entity.WithdrawalStatusCompleted}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)

	err := svc.RejectWithdrawal(ctx, 1, 100, "风险交易")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已处理")
}

func TestWithdrawalService_RejectWithdrawal_记录不存在(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.RejectWithdrawal(ctx, 999, 100, "风险")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

// ==================== FailWithdrawal 测试 ====================

func TestWithdrawalService_FailWithdrawal_成功(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID:       1,
		UserID:   10,
		Currency: "USDT",
		Amount:   decimal.NewFromInt(100),
		Status:   entity.WithdrawalStatusProcessing,
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT"}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.FailWithdrawal(ctx, 1, "链上发送失败")

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusFailed), withdrawal.Status)
	assert.Equal(t, "链上发送失败", withdrawal.ReviewNote)
}

func TestWithdrawalService_FailWithdrawal_记录不存在(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.FailWithdrawal(ctx, 999, "失败")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestWithdrawalService_FailWithdrawal_钱包不存在_跳过解冻(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
		Status: entity.WithdrawalStatusProcessing,
	}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.FailWithdrawal(ctx, 1, "链上失败")

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusFailed), withdrawal.Status)
	walletRepo.AssertNotCalled(t, "UnfreezeBalance", mock.Anything, mock.Anything, mock.Anything)
}

// ==================== IsAutoApprove 测试 ====================

func TestWithdrawalService_IsAutoApprove(t *testing.T) {
	tests := []struct {
		name     string
		amount   decimal.Decimal
		currency string
		want     bool
	}{
		{
			name:     "低于阈值_自动审核",
			amount:   decimal.NewFromInt(500),
			currency: "USDT",
			want:     true,
		},
		{
			name:     "等于阈值_不自动审核",
			amount:   decimal.NewFromInt(1000),
			currency: "USDT",
			want:     false,
		},
		{
			name:     "高于阈值_不自动审核",
			amount:   decimal.NewFromInt(5000),
			currency: "USDT",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, configRepo := newTestWithdrawalService()
			ctx := context.Background()
			setupDefaultUSDTConfig(configRepo)

			result := svc.IsAutoApprove(ctx, tt.currency, tt.amount)

			assert.Equal(t, tt.want, result)
		})
	}
}

// ==================== GetWithdrawConfig 测试 ====================

func TestWithdrawalService_GetWithdrawConfig_USDT(t *testing.T) {
	svc, _, _, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	cfg := svc.GetWithdrawConfig(ctx, "USDT")

	assert.Equal(t, decimal.NewFromInt(5), cfg.MinAmount)
	assert.Equal(t, decimal.NewFromInt(50000), cfg.MaxAmount)
	assert.Equal(t, decimal.NewFromInt(100000), cfg.DailyMaxAmount)
	assert.Equal(t, decimal.NewFromInt(1), cfg.Fee)
	assert.Equal(t, decimal.NewFromInt(1000), cfg.AutoThreshold)
}

func TestWithdrawalService_GetWithdrawConfig_TRX(t *testing.T) {
	svc, _, _, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()

	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawTRXMin).Return("50", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawTRXMax).Return("500000", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawTRXFee).Return("1", nil)

	cfg := svc.GetWithdrawConfig(ctx, "TRX")

	assert.Equal(t, decimal.NewFromInt(50), cfg.MinAmount)
	assert.Equal(t, decimal.NewFromInt(500000), cfg.MaxAmount)
	assert.Equal(t, decimal.NewFromInt(1), cfg.Fee)
}

func TestWithdrawalService_GetWithdrawConfig_不支持的币种(t *testing.T) {
	svc, _, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	cfg := svc.GetWithdrawConfig(ctx, "BTC")

	assert.Equal(t, decimal.NewFromInt(999999), cfg.MinAmount)
	assert.Equal(t, decimal.Zero, cfg.MaxAmount)
}

func TestWithdrawalService_GetWithdrawConfig_配置值无效使用默认(t *testing.T) {
	svc, _, _, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()

	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMin).Return("invalid", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMax).Return("50000", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTDailyMax).Return("100000", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTFee).Return("1", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTAutoThreshold).Return("1000", nil)

	cfg := svc.GetWithdrawConfig(ctx, "USDT")

	assert.Equal(t, decimal.NewFromInt(5), cfg.MinAmount) // 使用默认值 "5"
}

// ==================== 额外边界测试 ====================

func TestWithdrawalService_GetWithdrawConfig_配置为空字符串使用默认(t *testing.T) {
	svc, _, _, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()

	// 所有配置返回空字符串 (无错误)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMin).Return("", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTMax).Return("", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTDailyMax).Return("", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTFee).Return("", nil)
	configRepo.On("Get", mock.Anything, entity.ConfigWithdrawUSDTAutoThreshold).Return("", nil)

	cfg := svc.GetWithdrawConfig(ctx, "USDT")

	// 所有配置应使用默认值
	assert.Equal(t, decimal.NewFromInt(5), cfg.MinAmount)     // 默认 "5"
	assert.Equal(t, decimal.NewFromInt(50000), cfg.MaxAmount) // 默认 "50000"
	assert.Equal(t, decimal.NewFromInt(1), cfg.Fee)           // 默认 "1"
}

func TestWithdrawalService_CompleteWithdrawal_扣减余额失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100), Fee: decimal.NewFromInt(1),
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(500)}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(-100)).Return(errors.New("update error"))

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "扣减余额失败")
}

func TestWithdrawalService_CompleteWithdrawal_创建流水失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, txRepo, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100), Fee: decimal.NewFromInt(1),
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(500)}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	walletRepo.On("UpdateBalance", ctx, uint64(1), decimal.NewFromInt(-100)).Return(nil)
	txRepo.On("Create", ctx, mock.AnythingOfType("*entity.Transaction")).Return(errors.New("tx create error"))

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "创建提币流水失败")
}

func TestWithdrawalService_ApproveWithdrawal_查询失败(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.ApproveWithdrawal(ctx, 1, 100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

func TestWithdrawalService_RejectWithdrawal_查询失败(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.RejectWithdrawal(ctx, 1, 100, "原因")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

func TestWithdrawalService_RejectWithdrawal_查询钱包失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
		Status: entity.WithdrawalStatusPending,
	}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, errors.New("db error"))

	err := svc.RejectWithdrawal(ctx, 1, 100, "风险交易")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

func TestWithdrawalService_RejectWithdrawal_钱包不存在_跳过解冻(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
		Status: entity.WithdrawalStatusPending,
	}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	// 钱包不存在 (返回 nil, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(nil)

	err := svc.RejectWithdrawal(ctx, 1, 100, "风险交易")

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.WithdrawalStatusRejected), withdrawal.Status)
	// 确认 UnfreezeBalance 没有被调用 (钱包不存在时跳过)
	walletRepo.AssertNotCalled(t, "UnfreezeBalance", mock.Anything, mock.Anything, mock.Anything)
}

func TestWithdrawalService_RejectWithdrawal_更新失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
		Status: entity.WithdrawalStatusPending,
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT"}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(nil)
	withdrawalRepo.On("Update", ctx, mock.AnythingOfType("*entity.Withdrawal")).Return(errors.New("update error"))

	err := svc.RejectWithdrawal(ctx, 1, 100, "风险交易")

	assert.Error(t, err)
}

func TestWithdrawalService_FailWithdrawal_查询失败(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.FailWithdrawal(ctx, 1, "失败原因")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

func TestWithdrawalService_FailWithdrawal_查询钱包失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
	}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, errors.New("db error"))

	err := svc.FailWithdrawal(ctx, 1, "链上失败")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

func TestWithdrawalService_CreateWithdrawal_查询钱包失败(t *testing.T) {
	svc, _, walletRepo, _, configRepo := newTestWithdrawalService()
	ctx := context.Background()
	setupDefaultUSDTConfig(configRepo)

	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, errors.New("db error"))

	result, err := svc.CreateWithdrawal(ctx, 10, "USDT", "TAddr", decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

func TestWithdrawalService_CompleteWithdrawal_查询失败(t *testing.T) {
	svc, withdrawalRepo, _, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询提币记录失败")
}

func TestWithdrawalService_CompleteWithdrawal_查询钱包失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100)}
	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(nil, errors.New("db error"))

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询钱包失败")
}

func TestWithdrawalService_CompleteWithdrawal_解冻失败(t *testing.T) {
	svc, withdrawalRepo, walletRepo, _, _ := newTestWithdrawalService()
	ctx := context.Background()

	withdrawal := &entity.Withdrawal{
		ID: 1, UserID: 10, Currency: "USDT", Amount: decimal.NewFromInt(100),
	}
	wallet := &entity.Wallet{ID: 1, UserID: 10, Currency: "USDT", Balance: decimal.NewFromInt(500)}

	withdrawalRepo.On("FindByID", ctx, uint64(1)).Return(withdrawal, nil)
	walletRepo.On("FindByUserIDAndCurrency", ctx, uint64(10), "USDT").Return(wallet, nil)
	walletRepo.On("UnfreezeBalance", ctx, uint64(1), decimal.NewFromInt(100)).Return(errors.New("unfreeze error"))

	err := svc.CompleteWithdrawal(ctx, 1, "tx_hash")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "解冻余额失败")
}

