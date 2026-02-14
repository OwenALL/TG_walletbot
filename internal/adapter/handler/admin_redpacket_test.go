package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
)

// --- AdminRedPacketHandler 测试 ---
// 注意: List 和 Stats 方法内部使用 h.db 直接查询，需要真实 DB 才能测试。
// 此处仅测试 Detail 方法 (通过 repo) 和构造函数。

// TestRedPacketHandler_New 测试构造函数
func TestRedPacketHandler_New(t *testing.T) {
	handler := NewAdminRedPacketHandler(
		&mockRedPacketRepo{},
		&mockRedPacketClaimRepo{},
		&mockUserRepo{},
		nil, // db
		testLogger(),
	)
	assert.NotNil(t, handler)
}

// TestRedPacketHandler_Detail_Success 测试获取红包详情成功
func TestRedPacketHandler_Detail_Success(t *testing.T) {
	router := setupTestRouter()

	expireAt := time.Now().Add(24 * time.Hour)
	rpRepo := &mockRedPacketRepo{
		findByIDFunc: func(ctx context.Context, id uint64) (*entity.RedPacket, error) {
			return &entity.RedPacket{
				ID:          1,
				UserID:      10,
				Currency:    entity.CurrencyUSDT,
				TotalAmount: decimal.NewFromFloat(100),
				TotalCount:  10,
				Status:      entity.RedPacketStatusSent,
				ExpireAt:    expireAt,
			}, nil
		},
	}
	claimRepo := &mockRedPacketClaimRepo{
		listByPacketIDFunc: func(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
			return []*entity.RedPacketClaim{
				{ID: 1, RedPacketID: 1, UserID: 20, Amount: decimal.NewFromFloat(10)},
				{ID: 2, RedPacketID: 1, UserID: 30, Amount: decimal.NewFromFloat(15)},
			}, nil
		},
	}
	userRepo := &mockUserRepo{
		findByIDFunc: func(ctx context.Context, id uint64) (*entity.User, error) {
			switch id {
			case 10:
				return &entity.User{ID: 10, Username: "sender", FirstName: "Sender"}, nil
			case 20:
				return &entity.User{ID: 20, Username: "claimer1", FirstName: "Claimer1"}, nil
			case 30:
				return &entity.User{ID: 30, Username: "claimer2", FirstName: "Claimer2"}, nil
			}
			return nil, nil
		},
	}

	handler := NewAdminRedPacketHandler(rpRepo, claimRepo, userRepo, nil, testLogger())
	router.GET("/red-packets/:id", func(c *gin.Context) {
		c.Set("admin_id", uint64(1))
		c.Next()
	}, handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/red-packets/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseJSONResponse(t, w)
	assert.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "sender", data["sender_username"])
	assert.Equal(t, "Sender", data["sender_name"])

	claims := data["claims"].([]interface{})
	assert.Len(t, claims, 2)
}

// TestRedPacketHandler_Detail_InvalidID 测试无效红包 ID
func TestRedPacketHandler_Detail_InvalidID(t *testing.T) {
	router := setupTestRouter()
	handler := NewAdminRedPacketHandler(
		&mockRedPacketRepo{},
		&mockRedPacketClaimRepo{},
		&mockUserRepo{},
		nil,
		testLogger(),
	)
	router.GET("/red-packets/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/red-packets/abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRedPacketHandler_Detail_NotFound 测试红包不存在
func TestRedPacketHandler_Detail_NotFound(t *testing.T) {
	router := setupTestRouter()
	rpRepo := &mockRedPacketRepo{
		findByIDFunc: func(ctx context.Context, id uint64) (*entity.RedPacket, error) {
			return nil, nil
		},
	}

	handler := NewAdminRedPacketHandler(rpRepo, &mockRedPacketClaimRepo{}, &mockUserRepo{}, nil, testLogger())
	router.GET("/red-packets/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/red-packets/999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRedPacketHandler_Detail_RepoError 测试查询红包出错
func TestRedPacketHandler_Detail_RepoError(t *testing.T) {
	router := setupTestRouter()
	rpRepo := &mockRedPacketRepo{
		findByIDFunc: func(ctx context.Context, id uint64) (*entity.RedPacket, error) {
			return nil, assert.AnError
		},
	}

	handler := NewAdminRedPacketHandler(rpRepo, &mockRedPacketClaimRepo{}, &mockUserRepo{}, nil, testLogger())
	router.GET("/red-packets/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/red-packets/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRedPacketHandler_Detail_ClaimsError 测试查询领取记录出错
func TestRedPacketHandler_Detail_ClaimsError(t *testing.T) {
	router := setupTestRouter()
	rpRepo := &mockRedPacketRepo{
		findByIDFunc: func(ctx context.Context, id uint64) (*entity.RedPacket, error) {
			return &entity.RedPacket{
				ID:     1,
				UserID: 10,
				Status: entity.RedPacketStatusSent,
			}, nil
		},
	}
	claimRepo := &mockRedPacketClaimRepo{
		listByPacketIDFunc: func(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
			return nil, assert.AnError
		},
	}

	handler := NewAdminRedPacketHandler(rpRepo, claimRepo, &mockUserRepo{}, nil, testLogger())
	router.GET("/red-packets/:id", handler.Detail)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/red-packets/1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
