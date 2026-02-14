package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- HealthHandler 测试 ---

func TestHealthHandler_Check_Success(t *testing.T) {
	router := setupTestRouter()
	handler := NewHealthHandler()
	router.GET("/health", handler.Check)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "ok", data["status"])
	assert.Equal(t, "walletbot-api", data["service"])
}
