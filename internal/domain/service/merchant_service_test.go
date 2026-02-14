package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestMerchantService 创建测试用的 MerchantService 及其 mock 依赖
func newTestMerchantService() (*MerchantService, *mocks.MockMerchantRepository) {
	merchantRepo := new(mocks.MockMerchantRepository)
	logger := zap.NewNop()
	svc := NewMerchantService(merchantRepo, logger)
	return svc, merchantRepo
}

// ==================== Create 测试 ====================

func TestMerchantService_Create_成功_首个商户(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("CountByUserID", ctx, uint64(1)).Return(int64(0), nil)
	merchantRepo.On("Create", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	merchant, err := svc.Create(ctx, 1, "测试用户")

	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, uint64(1), merchant.UserID)
	assert.Equal(t, "测试用户的商户", merchant.BusinessName)
	assert.Equal(t, int8(entity.MerchantStatusActive), merchant.Status)
	assert.Len(t, merchant.APIKey, 32)
	assert.Len(t, merchant.APISecret, 64)
	assert.False(t, merchant.CreatedAt.IsZero())
	assert.False(t, merchant.UpdatedAt.IsZero())
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_Create_成功_第二个商户(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("CountByUserID", ctx, uint64(1)).Return(int64(1), nil)
	merchantRepo.On("Create", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	merchant, err := svc.Create(ctx, 1, "测试用户")

	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, "测试用户的商户(2)", merchant.BusinessName)
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_Create_查询数量失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("CountByUserID", ctx, uint64(1)).Return(int64(0), errors.New("db error"))

	merchant, err := svc.Create(ctx, 1, "测试用户")

	assert.Error(t, err)
	assert.Nil(t, merchant)
	assert.Contains(t, err.Error(), "查询商户数量失败")
}

func TestMerchantService_Create_创建失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("CountByUserID", ctx, uint64(1)).Return(int64(0), nil)
	merchantRepo.On("Create", ctx, mock.AnythingOfType("*entity.Merchant")).Return(errors.New("db error"))

	merchant, err := svc.Create(ctx, 1, "测试用户")

	assert.Error(t, err)
	assert.Nil(t, merchant)
	assert.Contains(t, err.Error(), "创建商户记录失败")
}

func TestMerchantService_Create_APIKey唯一性(t *testing.T) {
	svc1, merchantRepo1 := newTestMerchantService()
	svc2, merchantRepo2 := newTestMerchantService()
	ctx := context.Background()

	var key1, key2 string

	merchantRepo1.On("CountByUserID", ctx, uint64(1)).Return(int64(0), nil)
	merchantRepo1.On("Create", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil).Run(func(args mock.Arguments) {
		m := args.Get(1).(*entity.Merchant)
		key1 = m.APIKey
	})

	merchantRepo2.On("CountByUserID", ctx, uint64(2)).Return(int64(0), nil)
	merchantRepo2.On("Create", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil).Run(func(args mock.Arguments) {
		m := args.Get(1).(*entity.Merchant)
		key2 = m.APIKey
	})

	_, _ = svc1.Create(ctx, 1, "用户1")
	_, _ = svc2.Create(ctx, 2, "用户2")

	assert.NotEqual(t, key1, key2)
}

// ==================== ListByUserID 测试 ====================

func TestMerchantService_ListByUserID_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	expected := []*entity.Merchant{
		{ID: 1, UserID: 1, BusinessName: "商户1"},
		{ID: 2, UserID: 1, BusinessName: "商户2"},
	}
	merchantRepo.On("ListByUserID", ctx, uint64(1)).Return(expected, nil)

	result, err := svc.ListByUserID(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expected, result)
}

func TestMerchantService_ListByUserID_空列表(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("ListByUserID", ctx, uint64(999)).Return([]*entity.Merchant{}, nil)

	result, err := svc.ListByUserID(ctx, 999)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestMerchantService_ListByUserID_查询失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("ListByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.ListByUserID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询商户列表失败")
}

// ==================== GetByID 测试 ====================

func TestMerchantService_GetByID_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	expected := &entity.Merchant{ID: 1, UserID: 1, BusinessName: "测试商户"}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := svc.GetByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestMerchantService_GetByID_不存在(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.GetByID(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== UpdateName 测试 ====================

func TestMerchantService_UpdateName_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, BusinessName: "旧名称"}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.UpdateName(ctx, 1, 1, "新名称")

	assert.NoError(t, err)
	assert.Equal(t, "新名称", result.BusinessName)
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_UpdateName_非所有者(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)

	result, err := svc.UpdateName(ctx, 1, 999, "新名称")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "无权操作该商户")
}

// ==================== UpdateWebhook 测试 ====================

func TestMerchantService_UpdateWebhook_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.UpdateWebhook(ctx, 1, 1, "https://example.com/webhook")

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/webhook", result.WebhookURL)
}

func TestMerchantService_UpdateWebhook_URL格式错误(t *testing.T) {
	svc, _ := newTestMerchantService()
	ctx := context.Background()

	result, err := svc.UpdateWebhook(ctx, 1, 1, "not-a-url")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "回调地址格式不正确")
}

// ==================== RefreshSecret 测试 ====================

func TestMerchantService_RefreshSecret_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	oldKey := "old_api_key_1234567890123456"
	oldSecret := "old_api_secret_12345678901234567890123456789012345678901234"
	merchant := &entity.Merchant{
		ID:        1,
		UserID:    1,
		APIKey:    oldKey,
		APISecret: oldSecret,
		Status:    entity.MerchantStatusActive,
	}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.RefreshSecret(ctx, 1, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, oldKey, result.APIKey)
	assert.NotEqual(t, oldSecret, result.APISecret)
	assert.Len(t, result.APIKey, 32)
	assert.Len(t, result.APISecret, 64)
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_RefreshSecret_非所有者(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusActive}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)

	result, err := svc.RefreshSecret(ctx, 1, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "无权操作该商户")
}

// ==================== ToggleStatus 测试 ====================

func TestMerchantService_ToggleStatus_关闭商户(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusActive}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.ToggleStatus(ctx, 1, 1)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.MerchantStatusDisabled), result.Status)
}

func TestMerchantService_ToggleStatus_开启商户(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusDisabled}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.ToggleStatus(ctx, 1, 1)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.MerchantStatusActive), result.Status)
}

// ==================== AddIPWhitelist 测试 ====================

func TestMerchantService_AddIPWhitelist_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.AddIPWhitelist(ctx, 1, 1, "192.168.1.1")

	assert.NoError(t, err)
	assert.Contains(t, result.GetIPList(), "192.168.1.1")
}

func TestMerchantService_AddIPWhitelist_IP格式错误(t *testing.T) {
	svc, _ := newTestMerchantService()
	ctx := context.Background()

	result, err := svc.AddIPWhitelist(ctx, 1, 1, "not-an-ip")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "IP 地址格式不正确")
}

// ==================== RemoveIPWhitelist 测试 ====================

func TestMerchantService_RemoveIPWhitelist_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1}
	merchant.SetIPList([]string{"192.168.1.1", "10.0.0.1"})
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.RemoveIPWhitelist(ctx, 1, 1, "192.168.1.1")

	assert.NoError(t, err)
	assert.NotContains(t, result.GetIPList(), "192.168.1.1")
	assert.Contains(t, result.GetIPList(), "10.0.0.1")
}

// ==================== ApproveMerchant 测试 ====================

func TestMerchantService_ApproveMerchant_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusPending, BusinessName: "测试商户"}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	err := svc.ApproveMerchant(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.MerchantStatusActive), merchant.Status)
	assert.False(t, merchant.UpdatedAt.IsZero())
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_ApproveMerchant_商户不存在(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.ApproveMerchant(ctx, 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "商户")
}

func TestMerchantService_ApproveMerchant_非待审核状态(t *testing.T) {
	tests := []struct {
		name   string
		status int8
	}{
		{"已通过", entity.MerchantStatusActive},
		{"已拒绝", entity.MerchantStatusRejected},
		{"已禁用", entity.MerchantStatusDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, merchantRepo := newTestMerchantService()
			ctx := context.Background()

			merchant := &entity.Merchant{ID: 1, UserID: 1, Status: tt.status}
			merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)

			err := svc.ApproveMerchant(ctx, 1)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "仅待审核的商户可以审批")
		})
	}
}

func TestMerchantService_ApproveMerchant_查询失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.ApproveMerchant(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询商户信息失败")
}

func TestMerchantService_ApproveMerchant_更新失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusPending}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(errors.New("db error"))

	err := svc.ApproveMerchant(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "更新商户状态失败")
}

// ==================== RejectMerchant 测试 ====================

func TestMerchantService_RejectMerchant_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusPending}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	err := svc.RejectMerchant(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.MerchantStatusRejected), merchant.Status)
	assert.False(t, merchant.UpdatedAt.IsZero())
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_RejectMerchant_商户不存在(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.RejectMerchant(ctx, 999)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "商户")
}

func TestMerchantService_RejectMerchant_非待审核状态(t *testing.T) {
	tests := []struct {
		name   string
		status int8
	}{
		{"已通过", entity.MerchantStatusActive},
		{"已拒绝", entity.MerchantStatusRejected},
		{"已禁用", entity.MerchantStatusDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, merchantRepo := newTestMerchantService()
			ctx := context.Background()

			merchant := &entity.Merchant{ID: 1, UserID: 1, Status: tt.status}
			merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)

			err := svc.RejectMerchant(ctx, 1)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "仅待审核的商户可以拒绝")
		})
	}
}

func TestMerchantService_RejectMerchant_查询失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.RejectMerchant(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询商户信息失败")
}

func TestMerchantService_RejectMerchant_更新失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusPending}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(errors.New("db error"))

	err := svc.RejectMerchant(ctx, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "更新商户状态失败")
}

// ==================== ResetAPIKey 测试 ====================

func TestMerchantService_ResetAPIKey_成功(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	oldKey := "old_api_key_1234567890123456"
	oldSecret := "old_api_secret_12345678901234567890123456789012345678901234"
	merchant := &entity.Merchant{
		ID:        1,
		UserID:    1,
		APIKey:    oldKey,
		APISecret: oldSecret,
		Status:    entity.MerchantStatusActive,
	}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	result, err := svc.ResetAPIKey(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, oldKey, result.APIKey)
	assert.NotEqual(t, oldSecret, result.APISecret)
	assert.Len(t, result.APIKey, 32)
	assert.Len(t, result.APISecret, 64)
	assert.False(t, result.UpdatedAt.IsZero())
	merchantRepo.AssertExpectations(t)
}

func TestMerchantService_ResetAPIKey_商户不存在(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.ResetAPIKey(ctx, 999)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "商户")
}

func TestMerchantService_ResetAPIKey_查询失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchantRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.ResetAPIKey(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询商户信息失败")
}

func TestMerchantService_ResetAPIKey_更新失败(t *testing.T) {
	svc, merchantRepo := newTestMerchantService()
	ctx := context.Background()

	merchant := &entity.Merchant{ID: 1, UserID: 1, Status: entity.MerchantStatusActive}
	merchantRepo.On("FindByID", ctx, uint64(1)).Return(merchant, nil)
	merchantRepo.On("Update", ctx, mock.AnythingOfType("*entity.Merchant")).Return(errors.New("db error"))

	result, err := svc.ResetAPIKey(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "更新商户密钥失败")
}

// ==================== NewMerchantService 构造测试 ====================

func TestNewMerchantService(t *testing.T) {
	merchantRepo := new(mocks.MockMerchantRepository)
	logger := zap.NewNop()

	svc := NewMerchantService(merchantRepo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, merchantRepo, svc.merchantRepo)
	assert.Equal(t, logger, svc.logger)
}

// ==================== generateRandomHex 测试 ====================

func TestGenerateRandomHex_长度正确(t *testing.T) {
	tests := []struct {
		name    string
		bytes   int
		wantLen int
	}{
		{"16字节", 16, 32},
		{"32字节", 32, 64},
		{"1字节", 1, 2},
		{"64字节", 64, 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomHex(tt.bytes)
			assert.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestGenerateRandomHex_两次结果不同(t *testing.T) {
	result1, err1 := generateRandomHex(16)
	result2, err2 := generateRandomHex(16)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, result1, result2)
}
