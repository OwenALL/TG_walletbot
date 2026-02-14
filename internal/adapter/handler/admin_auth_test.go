package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

// --- 测试辅助 ---

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// parseJSONResponse 解析 JSON 响应体
func parseJSONResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err, "解析 JSON 响应失败")
	return result
}

// --- Auth Handler 测试 ---

func TestAuthHandler_Login_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	now := time.Now()
	mockUC.loginFunc = func(ctx context.Context, username, password string) (*entity.AdminUser, error) {
		return &entity.AdminUser{
			ID:          1,
			Username:    "admin",
			Role:        entity.AdminRoleSuperAdmin,
			Status:      1,
			LastLoginAt: &now,
		}, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(LoginReq{Username: "admin", Password: "password123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"])

	user := data["user"].(map[string]interface{})
	assert.Equal(t, float64(1), user["id"])
	assert.Equal(t, "admin", user["username"])
	assert.Equal(t, entity.AdminRoleSuperAdmin, user["role"])
}

func TestAuthHandler_Login_BadRequest_MissingFields(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.POST("/auth/login", handler.Login)

	// 空 body
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(http.StatusBadRequest), resp["code"])
}

func TestAuthHandler_Login_BadRequest_MissingPassword(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{"username": "admin"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_Unauthorized(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.loginFunc = func(ctx context.Context, username, password string) (*entity.AdminUser, error) {
		return nil, apperrors.NewUnauthorized("用户名或密码错误")
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(LoginReq{Username: "admin", Password: "wrongpwd"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Login_InternalError(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.loginFunc = func(ctx context.Context, username, password string) (*entity.AdminUser, error) {
		return nil, assert.AnError
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(LoginReq{Username: "admin", Password: "password123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	now := time.Now()
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return &entity.AdminUser{
			ID:          1,
			Username:    "admin",
			Role:        entity.AdminRoleSuperAdmin,
			Status:      1,
			LastLoginAt: &now,
			CreatedAt:   createdAt,
		}, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.GET("/me", func(c *gin.Context) {
		// 模拟 JWT 中间件注入的管理员信息
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.Me)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(1), data["id"])
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, entity.AdminRoleSuperAdmin, data["role"])
}

func TestAuthHandler_Me_AdminNotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return nil, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.GET("/me", func(c *gin.Context) {
		c.Set("admin_id", uint64(999))
		c.Next()
	}, handler.Me)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Me_NoAdminID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	// getAdminByID 将收到 id=0，返回 nil
	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return nil, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	// 不在 context 中注入 admin_id
	router.GET("/me", handler.Me)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- ChangePassword Handler 测试 ---

// TestAuthHandler_ChangePassword_Success 测试修改密码成功
func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	// 生成旧密码哈希 (对应 "oldpass123")
	oldHash, err := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.MinCost)
	require.NoError(t, err)

	var passwordUpdated bool
	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return &entity.AdminUser{
			ID:           1,
			Username:     "admin",
			PasswordHash: string(oldHash),
			Role:         entity.AdminRoleSuperAdmin,
			Status:       1,
		}, nil
	}
	mockUC.updateAdminFunc = func(ctx context.Context, admin *entity.AdminUser) error {
		passwordUpdated = true
		// 验证新密码哈希有效
		err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("newpass456"))
		assert.NoError(t, err, "新密码哈希应能通过验证")
		return nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
	assert.True(t, passwordUpdated, "密码应已更新")
}

// TestAuthHandler_ChangePassword_BadRequest_MissingFields 测试修改密码 - 缺少字段
func TestAuthHandler_ChangePassword_BadRequest_MissingFields(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ChangePassword)

	// 空请求体
	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAuthHandler_ChangePassword_BadRequest_NewPasswordTooShort 测试修改密码 - 新密码太短
func TestAuthHandler_ChangePassword_BadRequest_NewPasswordTooShort(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "oldpass123",
		NewPassword: "abc", // 少于 6 位
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAuthHandler_ChangePassword_WrongOldPassword 测试修改密码 - 旧密码不正确
func TestAuthHandler_ChangePassword_WrongOldPassword(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	// 旧密码对应 "correctpass"
	oldHash, err := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.MinCost)
	require.NoError(t, err)

	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return &entity.AdminUser{
			ID:           1,
			Username:     "admin",
			PasswordHash: string(oldHash),
			Role:         entity.AdminRoleSuperAdmin,
			Status:       1,
		}, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "wrongpass",  // 错误的旧密码
		NewPassword: "newpass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Contains(t, resp["message"], "旧密码不正确")
}

// TestAuthHandler_ChangePassword_NoAdminID 测试修改密码 - 未认证
func TestAuthHandler_ChangePassword_NoAdminID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	// 不注入 admin_id
	router.PUT("/auth/change-password", handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAuthHandler_ChangePassword_AdminNotFound 测试修改密码 - 管理员不存在
func TestAuthHandler_ChangePassword_AdminNotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return nil, nil
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(999))
		c.Next()
	}, handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAuthHandler_ChangePassword_UpdateError 测试修改密码 - 数据库更新失败
func TestAuthHandler_ChangePassword_UpdateError(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	oldHash, err := bcrypt.GenerateFromPassword([]byte("oldpass123"), bcrypt.MinCost)
	require.NoError(t, err)

	mockUC.getAdminByIDFunc = func(ctx context.Context, id uint64) (*entity.AdminUser, error) {
		return &entity.AdminUser{
			ID:           1,
			Username:     "admin",
			PasswordHash: string(oldHash),
			Role:         entity.AdminRoleSuperAdmin,
			Status:       1,
		}, nil
	}
	mockUC.updateAdminFunc = func(ctx context.Context, admin *entity.AdminUser) error {
		return assert.AnError
	}

	handler := NewAuthHandler(mockUC.toAdminUseCase(), "test-jwt-secret", 24, testLogger())
	router.PUT("/auth/change-password", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ChangePassword)

	body, _ := json.Marshal(ChangePasswordReq{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
