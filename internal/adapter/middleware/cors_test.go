package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ========== CORS 中间件测试 ==========

// setupCORSRouter 创建带有 CORS 中间件的测试路由
func setupCORSRouter() *gin.Engine {
	r := gin.New()
	r.Use(CORS())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "created"})
	})
	return r
}

func TestCORS_响应头正确设置(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Origin, Content-Type, Accept, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_OPTIONS预检请求返回204(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Origin, Content-Type, Accept, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	// OPTIONS 请求不应有响应体
	assert.Empty(t, w.Body.String())
}

func TestCORS_OPTIONS预检请求不继续执行后续Handler(t *testing.T) {
	handlerCalled := false
	r := gin.New()
	r.Use(CORS())
	r.GET("/api/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, handlerCalled, "OPTIONS 请求不应执行后续 handler")
}

func TestCORS_非OPTIONS请求继续执行后续Handler(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestCORS_POST请求正确设置头部(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_不同HTTP方法均设置CORS头(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	// 注册所有方法的处理器
	handler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": c.Request.Method})
	}
	r.GET("/api/test", handler)
	r.POST("/api/test", handler)
	r.PUT("/api/test", handler)
	r.DELETE("/api/test", handler)
	r.PATCH("/api/test", handler)

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/api/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"),
				"%s 请求应设置 Access-Control-Allow-Origin", method)
		})
	}
}

func TestCORS_带Origin头的请求(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// 当前实现使用 * 通配，无论 Origin 是什么
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_预检请求带Origin头(t *testing.T) {
	router := setupCORSRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/api/test", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}
