package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// --- AdminUserHandler 测试 ---

// TestAdminUserHandler_Detail_Success 测试获取用户详情成功
func TestAdminUserHandler_Detail_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{
			ID:         1,
			TelegramID: 123456,
			Username:   "testuser",
			FirstName:  "Test",
			LastName:   "User",
			Status:     entity.UserStatusActive,
		}, nil
	}
	mockUC.findWalletsByUserIDFunc = func(ctx context.Context, userID uint64) ([]*entity.Wallet, error) {
		return []*entity.Wallet{
			{ID: 1, UserID: 1, Currency: entity.CurrencyUSDT, Balance: decimal.NewFromFloat(100.50)},
			{ID: 2, UserID: 1, Currency: entity.CurrencyTRX, Balance: decimal.NewFromFloat(200)},
		}, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/users/:id", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	// UserDetail 使用匿名嵌入，JSON 为扁平结构
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "testuser", data["username"])
	assert.Equal(t, false, data["has_pin"])

	wallets := data["wallets"].([]interface{})
	assert.Len(t, wallets, 2)
}

// TestAdminUserHandler_Detail_InvalidID 测试无效用户 ID
func TestAdminUserHandler_Detail_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/users/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_Detail_NotFound 测试用户不存在
func TestAdminUserHandler_Detail_NotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/users/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users/999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAdminUserHandler_UpdateStatus_Success 测试更新用户状态成功
func TestAdminUserHandler_UpdateStatus_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, Status: entity.UserStatusActive}, nil
	}
	mockUC.updateUserFunc = func(ctx context.Context, user *entity.User) error {
		return nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/status", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleAdmin)
		c.Next()
	}, handler.UpdateStatus)

	body, _ := json.Marshal(UpdateUserStatusReq{Status: entity.UserStatusFrozen})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminUserHandler_UpdateStatus_InvalidID 测试无效用户 ID
func TestAdminUserHandler_UpdateStatus_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/status", handler.UpdateStatus)

	body, _ := json.Marshal(UpdateUserStatusReq{Status: entity.UserStatusFrozen})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/abc/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_UpdateStatus_InvalidBody 测试请求体无效
func TestAdminUserHandler_UpdateStatus_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/status", handler.UpdateStatus)

	// status 必须是 1/2/3
	body, _ := json.Marshal(map[string]int{"status": 99})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_UpdateStatus_UserNotFound 测试用户不存在
func TestAdminUserHandler_UpdateStatus_UserNotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/status", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.UpdateStatus)

	body, _ := json.Marshal(UpdateUserStatusReq{Status: entity.UserStatusFrozen})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/999/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// UpdateUserStatus 找不到用户返回 NotFound AppError
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAdminUserHandler_UpdateStatus_InternalError 测试内部错误
func TestAdminUserHandler_UpdateStatus_InternalError(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, apperrors.Wrap(assert.AnError, "DB 异常")
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/status", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.UpdateStatus)

	body, _ := json.Marshal(UpdateUserStatusReq{Status: entity.UserStatusFrozen})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/status", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== ResetPIN Handler 测试 ====================

// TestAdminUserHandler_ResetPIN_Success 测试重置用户 PIN 成功
func TestAdminUserHandler_ResetPIN_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, PINHash: "somehash", PINFailCount: 3, Status: entity.UserStatusActive}, nil
	}
	mockUC.updateUserFunc = func(ctx context.Context, user *entity.User) error {
		// 验证 PIN 已被清空
		assert.Equal(t, "", user.PINHash)
		assert.Equal(t, 0, user.PINFailCount)
		assert.Nil(t, user.PINLockedUntil)
		return nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/reset-pin", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleAdmin)
		c.Next()
	}, handler.ResetPIN)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/reset-pin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestAdminUserHandler_ResetPIN_InvalidID 测试无效用户 ID
func TestAdminUserHandler_ResetPIN_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/reset-pin", handler.ResetPIN)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/abc/reset-pin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_ResetPIN_UserNotFound 测试用户不存在
func TestAdminUserHandler_ResetPIN_UserNotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/reset-pin", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ResetPIN)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/999/reset-pin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAdminUserHandler_ResetPIN_InternalError 测试内部错误
func TestAdminUserHandler_ResetPIN_InternalError(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, errors.New("db error")
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/reset-pin", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.ResetPIN)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/reset-pin", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ==================== AdjustBalance Handler 测试 ====================

// TestAdminUserHandler_AdjustBalance_Success_Increase 测试增加余额成功
func TestAdminUserHandler_AdjustBalance_Success_Increase(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, Status: entity.UserStatusActive}, nil
	}
	mockUC.findWalletByUserIDCurrencyFunc = func(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
		return &entity.Wallet{ID: 10, UserID: 1, Currency: "USDT", Balance: decimal.NewFromFloat(100)}, nil
	}
	mockUC.updateWalletBalanceFunc = func(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
		assert.Equal(t, uint64(10), walletID)
		assert.True(t, amount.Equal(decimal.NewFromFloat(50)))
		return nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{
		Currency: "USDT",
		Amount:   decimal.NewFromFloat(50),
		Reason:   "测试增加余额",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestAdminUserHandler_AdjustBalance_Success_Decrease 测试扣减余额成功
func TestAdminUserHandler_AdjustBalance_Success_Decrease(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, Status: entity.UserStatusActive}, nil
	}
	mockUC.findWalletByUserIDCurrencyFunc = func(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
		return &entity.Wallet{ID: 10, UserID: 1, Currency: "USDT", Balance: decimal.NewFromFloat(100)}, nil
	}
	mockUC.updateWalletBalanceFunc = func(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
		return nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{
		Currency: "USDT",
		Amount:   decimal.NewFromFloat(-30),
		Reason:   "测试扣减余额",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminUserHandler_AdjustBalance_InvalidID 测试无效用户 ID
func TestAdminUserHandler_AdjustBalance_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{Currency: "USDT", Amount: decimal.NewFromFloat(10), Reason: "test"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/abc/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_AdjustBalance_InvalidBody 测试请求体无效
func TestAdminUserHandler_AdjustBalance_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", handler.AdjustBalance)

	// 缺少 reason 字段
	body, _ := json.Marshal(map[string]interface{}{"currency": "USDT", "amount": 10})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminUserHandler_AdjustBalance_ZeroAmount 测试金额为 0
func TestAdminUserHandler_AdjustBalance_ZeroAmount(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{Currency: "USDT", Amount: decimal.Zero, Reason: "测试"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Contains(t, resp["message"], "不能为 0")
}

// TestAdminUserHandler_AdjustBalance_InvalidCurrency 测试不支持的币种
func TestAdminUserHandler_AdjustBalance_InvalidCurrency(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, Status: entity.UserStatusActive}, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{Currency: "BTC", Amount: decimal.NewFromFloat(10), Reason: "测试"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Contains(t, resp["message"], "不支持的币种")
}

// TestAdminUserHandler_AdjustBalance_InsufficientBalance 测试扣减余额不足
func TestAdminUserHandler_AdjustBalance_InsufficientBalance(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return &entity.User{ID: 1, Status: entity.UserStatusActive}, nil
	}
	mockUC.findWalletByUserIDCurrencyFunc = func(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
		return &entity.Wallet{ID: 10, UserID: 1, Currency: "USDT", Balance: decimal.NewFromFloat(50)}, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{Currency: "USDT", Amount: decimal.NewFromFloat(-100), Reason: "测试扣减"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/1/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Contains(t, resp["message"], "余额不足")
}

// TestAdminUserHandler_AdjustBalance_UserNotFound 测试用户不存在
func TestAdminUserHandler_AdjustBalance_UserNotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findUserByIDFunc = func(ctx context.Context, id uint64) (*entity.User, error) {
		return nil, nil
	}

	handler := NewAdminUserHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/users/:id/adjust-balance", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.AdjustBalance)

	body, _ := json.Marshal(AdjustBalanceReq{Currency: "USDT", Amount: decimal.NewFromFloat(10), Reason: "测试"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/users/999/adjust-balance", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
