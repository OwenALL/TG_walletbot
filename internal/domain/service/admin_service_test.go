package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestAdminService 创建测试用的 AdminService 及其 mock 依赖
func newTestAdminService() (*AdminService, *mocks.MockAdminUserRepository, *mocks.MockAdminLogRepository) {
	adminUserRepo := new(mocks.MockAdminUserRepository)
	adminLogRepo := new(mocks.MockAdminLogRepository)
	logger := zap.NewNop()
	svc := NewAdminService(adminUserRepo, adminLogRepo, logger)
	return svc, adminUserRepo, adminLogRepo
}

// hashPassword 辅助函数: 生成 bcrypt hash
func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

// ==================== Authenticate 测试 ====================

func TestAdminService_Authenticate_成功(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	password := "admin123"
	admin := &entity.AdminUser{
		ID:           1,
		Username:     "admin",
		PasswordHash: hashPassword(password),
		Role:         entity.AdminRoleSuperAdmin,
		Status:       1,
	}
	adminUserRepo.On("FindByUsername", ctx, "admin").Return(admin, nil)
	adminUserRepo.On("Update", ctx, mock.AnythingOfType("*entity.AdminUser")).Return(nil)

	result, err := svc.Authenticate(ctx, "admin", password)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(1), result.ID)
	assert.Equal(t, "admin", result.Username)
	assert.NotNil(t, result.LastLoginAt)
}

func TestAdminService_Authenticate_用户名不存在(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	adminUserRepo.On("FindByUsername", ctx, "nobody").Return(nil, nil)

	result, err := svc.Authenticate(ctx, "nobody", "password")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "用户名或密码错误")
}

func TestAdminService_Authenticate_密码错误(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	admin := &entity.AdminUser{
		ID:           1,
		Username:     "admin",
		PasswordHash: hashPassword("correct_password"),
		Status:       1,
	}
	adminUserRepo.On("FindByUsername", ctx, "admin").Return(admin, nil)

	result, err := svc.Authenticate(ctx, "admin", "wrong_password")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "用户名或密码错误")
}

func TestAdminService_Authenticate_账号被禁用(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	admin := &entity.AdminUser{
		ID:           1,
		Username:     "admin",
		PasswordHash: hashPassword("admin123"),
		Status:       0, // 禁用
	}
	adminUserRepo.On("FindByUsername", ctx, "admin").Return(admin, nil)

	result, err := svc.Authenticate(ctx, "admin", "admin123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "已被禁用")
}

func TestAdminService_Authenticate_查询失败(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	adminUserRepo.On("FindByUsername", ctx, "admin").Return(nil, errors.New("db error"))

	result, err := svc.Authenticate(ctx, "admin", "admin123")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询管理员失败")
}

// ==================== GetAdminByID 测试 ====================

func TestAdminService_GetAdminByID_成功(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	expected := &entity.AdminUser{ID: 1, Username: "admin"}
	adminUserRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := svc.GetAdminByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestAdminService_GetAdminByID_不存在(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	adminUserRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.GetAdminByID(ctx, 999)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAdminService_GetAdminByID_查询失败(t *testing.T) {
	svc, adminUserRepo, _ := newTestAdminService()
	ctx := context.Background()

	adminUserRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetAdminByID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== CreateAuditLog 测试 ====================

func TestAdminService_CreateAuditLog_成功(t *testing.T) {
	svc, _, adminLogRepo := newTestAdminService()
	ctx := context.Background()

	adminLogRepo.On("Create", ctx, mock.AnythingOfType("*entity.AdminLog")).Return(nil)
	targetID := uint64(42)

	svc.CreateAuditLog(ctx, 1, "login", "admin_user", &targetID, `{"ip":"127.0.0.1"}`, "127.0.0.1")

	adminLogRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*entity.AdminLog"))
}

func TestAdminService_CreateAuditLog_创建失败不panic(t *testing.T) {
	svc, _, adminLogRepo := newTestAdminService()
	ctx := context.Background()

	adminLogRepo.On("Create", ctx, mock.AnythingOfType("*entity.AdminLog")).Return(errors.New("db error"))

	// 不应 panic，即使创建失败
	assert.NotPanics(t, func() {
		svc.CreateAuditLog(ctx, 1, "login", "admin_user", nil, "", "127.0.0.1")
	})
}

// ==================== ListAuditLogs 测试 ====================

func TestAdminService_ListAuditLogs_成功(t *testing.T) {
	svc, _, adminLogRepo := newTestAdminService()
	ctx := context.Background()

	logs := []*entity.AdminLog{
		{ID: 1, AdminID: 1, Action: "login"},
		{ID: 2, AdminID: 1, Action: "approve_withdrawal"},
	}
	filter := &port.AdminLogFilter{Action: "login"}
	adminLogRepo.On("List", ctx, filter, 0, 10).Return(logs, int64(2), nil)

	result, total, err := svc.ListAuditLogs(ctx, filter, 0, 10)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(2), total)
}

func TestAdminService_ListAuditLogs_空列表(t *testing.T) {
	svc, _, adminLogRepo := newTestAdminService()
	ctx := context.Background()

	adminLogRepo.On("List", ctx, (*port.AdminLogFilter)(nil), 0, 10).Return([]*entity.AdminLog{}, int64(0), nil)

	result, total, err := svc.ListAuditLogs(ctx, nil, 0, 10)

	assert.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, int64(0), total)
}

// ==================== HashPassword 测试 ====================

func TestHashPassword_成功(t *testing.T) {
	hash, err := HashPassword("test_password")

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	// 验证可以成功比对
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("test_password")))
}

func TestHashPassword_不同密码不同哈希(t *testing.T) {
	hash1, _ := HashPassword("password1")
	hash2, _ := HashPassword("password2")

	assert.NotEqual(t, hash1, hash2)
}
