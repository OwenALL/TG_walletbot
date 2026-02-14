package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
)

// --- AdminConfigHandler 测试 ---

// TestConfigHandler_List_Success 测试获取配置列表成功
func TestConfigHandler_List_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.getAllConfigsFunc = func(ctx context.Context) ([]*entity.SystemConfig, error) {
		return []*entity.SystemConfig{
			{ID: 1, ConfigKey: "withdraw_usdt_min", ConfigValue: "10", Description: "最低提币金额"},
			{ID: 2, ConfigKey: "maintenance_mode", ConfigValue: "false", Description: "维护模式"},
		}, nil
	}

	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/configs", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

// TestConfigHandler_List_Empty 测试空配置列表
func TestConfigHandler_List_Empty(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.getAllConfigsFunc = func(ctx context.Context) ([]*entity.SystemConfig, error) {
		return []*entity.SystemConfig{}, nil
	}

	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/configs", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestConfigHandler_List_Error 测试获取配置失败
func TestConfigHandler_List_Error(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.getAllConfigsFunc = func(ctx context.Context) ([]*entity.SystemConfig, error) {
		return nil, assert.AnError
	}

	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/configs", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/configs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestConfigHandler_Update_Success 测试更新配置成功
func TestConfigHandler_Update_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.updateConfigFunc = func(ctx context.Context, key, value string) error {
		return nil
	}

	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/configs", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.Update)

	body, _ := json.Marshal(UpdateConfigReq{Key: "withdraw_usdt_min", Value: "20"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestConfigHandler_Update_MissingKey 测试更新配置 - 缺少 key
func TestConfigHandler_Update_MissingKey(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/configs", handler.Update)

	body, _ := json.Marshal(map[string]string{"value": "20"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestConfigHandler_Update_MissingValue 测试更新配置 - 缺少 value
func TestConfigHandler_Update_MissingValue(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/configs", handler.Update)

	body, _ := json.Marshal(map[string]string{"key": "some_key"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestConfigHandler_Update_EmptyBody 测试更新配置 - 空请求体
func TestConfigHandler_Update_EmptyBody(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/configs", handler.Update)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/configs", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestConfigHandler_Update_Error 测试更新配置 - 内部错误
func TestConfigHandler_Update_Error(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.updateConfigFunc = func(ctx context.Context, key, value string) error {
		return assert.AnError
	}

	handler := NewAdminConfigHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/configs", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Update)

	body, _ := json.Marshal(UpdateConfigReq{Key: "some_key", Value: "some_value"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/configs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
