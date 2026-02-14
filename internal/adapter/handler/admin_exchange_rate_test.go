package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
)

// --- AdminExchangeRateHandler 测试 ---

// TestExchangeRateHandler_List_Success 测试获取汇率列表成功
func TestExchangeRateHandler_List_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.getAllExchangeRatesFunc = func(ctx context.Context) ([]*entity.ExchangeRate, error) {
		return []*entity.ExchangeRate{
			{
				ID:           1,
				FromCurrency: entity.CurrencyUSDT,
				ToCurrency:   entity.CurrencyCNY,
				Rate:         decimal.NewFromFloat(7.2),
				Enabled:      true,
			},
			{
				ID:           2,
				FromCurrency: entity.CurrencyUSDT,
				ToCurrency:   entity.CurrencyTRX,
				Rate:         decimal.NewFromFloat(6.5),
				Enabled:      true,
			},
		}, nil
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/exchange-rates", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/exchange-rates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

// TestExchangeRateHandler_List_Empty 测试空汇率列表
func TestExchangeRateHandler_List_Empty(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.getAllExchangeRatesFunc = func(ctx context.Context) ([]*entity.ExchangeRate, error) {
		return []*entity.ExchangeRate{}, nil
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/exchange-rates", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/exchange-rates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestExchangeRateHandler_List_Error 测试获取汇率列表失败
func TestExchangeRateHandler_List_Error(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.getAllExchangeRatesFunc = func(ctx context.Context) ([]*entity.ExchangeRate, error) {
		return nil, assert.AnError
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.GET("/exchange-rates", handler.List)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/exchange-rates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestExchangeRateHandler_Update_Success 测试更新汇率成功
func TestExchangeRateHandler_Update_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.updateExchangeRateFunc = func(ctx context.Context, rate *entity.ExchangeRate) error {
		return nil
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleSuperAdmin)
		c.Next()
	}, handler.Update)

	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         "7.2",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestExchangeRateHandler_Update_MissingFields 测试更新汇率 - 缺少必填字段
func TestExchangeRateHandler_Update_MissingFields(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", handler.Update)

	body, _ := json.Marshal(map[string]string{"from_currency": "USDT"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestExchangeRateHandler_Update_InvalidRate 测试更新汇率 - 无效汇率值
func TestExchangeRateHandler_Update_InvalidRate(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", handler.Update)

	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         "-1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestExchangeRateHandler_Update_ZeroRate 测试更新汇率 - 零汇率
func TestExchangeRateHandler_Update_ZeroRate(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", handler.Update)

	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         "0",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestExchangeRateHandler_Update_NonNumericRate 测试更新汇率 - 非数字汇率
func TestExchangeRateHandler_Update_NonNumericRate(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", handler.Update)

	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         "abc",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestExchangeRateHandler_Update_WithOptionalFields 测试更新汇率 - 携带可选字段
func TestExchangeRateHandler_Update_WithOptionalFields(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	var capturedRate *entity.ExchangeRate
	mockUC.updateExchangeRateFunc = func(ctx context.Context, rate *entity.ExchangeRate) error {
		capturedRate = rate
		return nil
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Update)

	enabled := false
	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "TRX",
		Rate:         "6.5",
		Spread:       "0.5",
		MinAmount:    "10",
		MaxAmount:    "10000",
		Enabled:      &enabled,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, capturedRate)
	assert.Equal(t, "USDT", capturedRate.FromCurrency)
	assert.Equal(t, "TRX", capturedRate.ToCurrency)
	assert.True(t, capturedRate.Rate.Equal(decimal.NewFromFloat(6.5)))
	assert.True(t, capturedRate.Spread.Equal(decimal.NewFromFloat(0.5)))
	assert.True(t, capturedRate.MinAmount.Equal(decimal.NewFromFloat(10)))
	assert.True(t, capturedRate.MaxAmount.Equal(decimal.NewFromFloat(10000)))
	assert.False(t, capturedRate.Enabled)
}

// TestExchangeRateHandler_Update_InternalError 测试更新汇率 - 内部错误
func TestExchangeRateHandler_Update_InternalError(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.updateExchangeRateFunc = func(ctx context.Context, rate *entity.ExchangeRate) error {
		return assert.AnError
	}

	handler := NewAdminExchangeRateHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/exchange-rates", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Update)

	body, _ := json.Marshal(UpdateExchangeRateReq{
		FromCurrency: "USDT",
		ToCurrency:   "CNY",
		Rate:         "7.2",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/exchange-rates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
