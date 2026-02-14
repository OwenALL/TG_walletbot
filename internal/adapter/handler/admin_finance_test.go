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

// --- AdminFinanceHandler 测试 ---
// 注意: ListInvestments 和 Stats 方法内部使用 h.db 直接查询，需要真实 DB。
// 此处仅测试 UpdateRate 方法 (通过 systemConfigRepo) 和构造函数。

// TestFinanceHandler_New 测试构造函数
func TestFinanceHandler_New(t *testing.T) {
	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		&mockSystemConfigRepo{},
		&mockUserRepo{},
		&mockAdminLogRepo{},
		nil, // db
		testLogger(),
	)
	assert.NotNil(t, handler)
}

// TestFinanceHandler_UpdateRate_Success 测试更新年化利率成功
func TestFinanceHandler_UpdateRate_Success(t *testing.T) {
	router := setupTestRouter()

	sysConfigRepo := &mockSystemConfigRepo{}
	adminLogRepo := &mockAdminLogRepo{}

	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		sysConfigRepo,
		&mockUserRepo{},
		adminLogRepo,
		nil,
		testLogger(),
	)

	router.PUT("/finance/rate", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.UpdateRate)

	body, _ := json.Marshal(updateRateReq{AnnualRate: 8.5})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/finance/rate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "8.5000", data["annual_rate"])
}

// TestFinanceHandler_UpdateRate_InvalidBody 测试更新利率 - 无效请求体
func TestFinanceHandler_UpdateRate_InvalidBody(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		&mockSystemConfigRepo{},
		&mockUserRepo{},
		&mockAdminLogRepo{},
		nil,
		testLogger(),
	)
	router.PUT("/finance/rate", handler.UpdateRate)

	// 空 body
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/finance/rate", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFinanceHandler_UpdateRate_ZeroRate 测试更新利率 - 零利率
func TestFinanceHandler_UpdateRate_ZeroRate(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		&mockSystemConfigRepo{},
		&mockUserRepo{},
		&mockAdminLogRepo{},
		nil,
		testLogger(),
	)
	router.PUT("/finance/rate", handler.UpdateRate)

	body, _ := json.Marshal(map[string]float64{"annual_rate": 0})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/finance/rate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFinanceHandler_UpdateRate_ExceedMax 测试更新利率 - 超过 100
func TestFinanceHandler_UpdateRate_ExceedMax(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		&mockSystemConfigRepo{},
		&mockUserRepo{},
		&mockAdminLogRepo{},
		nil,
		testLogger(),
	)
	router.PUT("/finance/rate", handler.UpdateRate)

	body, _ := json.Marshal(map[string]float64{"annual_rate": 101})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/finance/rate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFinanceHandler_UpdateRate_Error 测试更新利率 - 保存失败
func TestFinanceHandler_UpdateRate_Error(t *testing.T) {
	router := setupTestRouter()

	sysConfigRepo := &mockSystemConfigRepo{
		setFunc: func(_ context.Context, _, _ string) error {
			return assert.AnError
		},
	}

	handler := NewAdminFinanceHandler(
		&mockFinanceInvestmentRepo{},
		sysConfigRepo,
		&mockUserRepo{},
		&mockAdminLogRepo{},
		nil,
		testLogger(),
	)
	router.PUT("/finance/rate", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.UpdateRate)

	body, _ := json.Marshal(updateRateReq{AnnualRate: 5.0})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/finance/rate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
