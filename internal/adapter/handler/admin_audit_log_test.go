package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
)

// --- AdminAuditLogHandler 测试 ---

// TestAuditLogHandler_List_Success 测试获取审计日志列表成功
func TestAuditLogHandler_List_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	targetID := uint64(10)
	mockUC.listAuditLogsFunc = func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
		return []*entity.AdminLog{
			{
				ID:         1,
				AdminID:    1,
				Action:     "login",
				TargetType: "admin",
				TargetID:   nil,
				IPAddress:  "127.0.0.1",
				CreatedAt:  time.Now(),
			},
			{
				ID:         2,
				AdminID:    1,
				Action:     "update_user_status",
				TargetType: "user",
				TargetID:   &targetID,
				Detail:     "冻结用户",
				IPAddress:  "127.0.0.1",
				CreatedAt:  time.Now(),
			},
		}, 2, nil
	}

	handler := NewAdminAuditLogHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/audit-logs", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleAdmin)
		c.Next()
	}, handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	assert.Len(t, list, 2)
	assert.Equal(t, float64(2), data["total"])
}

// TestAuditLogHandler_List_WithFilter 测试带筛选条件的审计日志列表
func TestAuditLogHandler_List_WithFilter(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	var capturedFilter *port.AdminLogFilter
	mockUC.listAuditLogsFunc = func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
		capturedFilter = filter
		return []*entity.AdminLog{}, 0, nil
	}

	handler := NewAdminAuditLogHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/audit-logs", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs?action=login&target_type=admin&admin_id=1&page=2&size=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedFilter)
	assert.Equal(t, "login", capturedFilter.Action)
	assert.Equal(t, "admin", capturedFilter.TargetType)
	assert.NotNil(t, capturedFilter.AdminID)
	assert.Equal(t, uint64(1), *capturedFilter.AdminID)
}

// TestAuditLogHandler_List_Error 测试获取审计日志失败
func TestAuditLogHandler_List_Error(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.listAuditLogsFunc = func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
		return nil, 0, assert.AnError
	}

	handler := NewAdminAuditLogHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/audit-logs", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAuditLogHandler_List_DefaultPagination 测试默认分页参数
func TestAuditLogHandler_List_DefaultPagination(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	var capturedOffset, capturedLimit int
	mockUC.listAuditLogsFunc = func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
		capturedOffset = offset
		capturedLimit = limit
		return []*entity.AdminLog{}, 0, nil
	}

	handler := NewAdminAuditLogHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/audit-logs", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/audit-logs", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// 默认 page=1, size=20, offset = (1-1)*20 = 0
	assert.Equal(t, 0, capturedOffset)
	assert.Equal(t, 20, capturedLimit)
}
