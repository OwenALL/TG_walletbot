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

// --- AdminWithdrawalHandler 测试 ---
// 注意: ListWithdrawals 使用 uc.db 直接查询，无法在无 DB 环境下测试。
// Approve 使用 withdrawalRepo，可以测试。
// Reject 和 Complete 内部使用 uc.db.Transaction，无法完整测试。
// 此处覆盖 Approve 的成功/失败路径，以及 Complete/Reject 的参数校验。

// TestAdminWithdrawalHandler_Approve_Success 测试批准提币成功
func TestAdminWithdrawalHandler_Approve_Success(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()

	mockUC.findWithdrawalByIDFunc = func(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
		return &entity.Withdrawal{
			ID:       1,
			UserID:   10,
			Currency: entity.CurrencyUSDT,
			Amount:   decimal.NewFromFloat(100),
			Status:   entity.WithdrawalStatusPending,
		}, nil
	}
	mockUC.updateWithdrawalFunc = func(ctx context.Context, w *entity.Withdrawal) error {
		return nil
	}

	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/approve", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Set("admin_role", entity.AdminRoleAdmin)
		c.Next()
	}, handler.Approve)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/1/approve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])
}

// TestAdminWithdrawalHandler_Approve_InvalidID 测试无效提币 ID
func TestAdminWithdrawalHandler_Approve_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/approve", handler.Approve)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/abc/approve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminWithdrawalHandler_Approve_NotFound 测试提币记录不存在
func TestAdminWithdrawalHandler_Approve_NotFound(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findWithdrawalByIDFunc = func(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
		return nil, nil
	}

	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/approve", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Approve)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/999/approve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAdminWithdrawalHandler_Approve_NotPending 测试非待审核状态
func TestAdminWithdrawalHandler_Approve_NotPending(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	mockUC.findWithdrawalByIDFunc = func(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
		return &entity.Withdrawal{
			ID:     1,
			Status: entity.WithdrawalStatusCompleted, // 已完成
		}, nil
	}

	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/approve", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Approve)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/1/approve", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminWithdrawalHandler_Reject_InvalidID 测试拒绝提币 - 无效 ID
func TestAdminWithdrawalHandler_Reject_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/reject", handler.Reject)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/abc/reject", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminWithdrawalHandler_Reject_MissingReason 测试拒绝提币 - 缺少原因
func TestAdminWithdrawalHandler_Reject_MissingReason(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/reject", handler.Reject)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/1/reject", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminWithdrawalHandler_Complete_InvalidID 测试手动完成 - 无效 ID
func TestAdminWithdrawalHandler_Complete_InvalidID(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/complete", handler.Complete)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/abc/complete", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminWithdrawalHandler_Complete_MissingTxHash 测试手动完成 - 缺少交易哈希
func TestAdminWithdrawalHandler_Complete_MissingTxHash(t *testing.T) {
	router := setupTestRouter()
	mockUC := newMockAdminUseCase()
	handler := NewAdminWithdrawalHandler(mockUC.toAdminUseCase(), testLogger())
	router.PUT("/withdrawals/:id/complete", handler.Complete)

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/withdrawals/1/complete", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
