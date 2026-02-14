package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// --- admin_routes 测试 ---
// RegisterAdminRoutes 需要完整的依赖 (Config, Container, DB)，
// 属于集成测试范畴。此处仅验证辅助函数的正确性。

// TestParsePagination_Defaults 测试默认分页参数
func TestParsePagination_Defaults(t *testing.T) {
	router := setupTestRouter()
	var page, size int

	router.GET("/test", func(c *gin.Context) {
		page, size = parsePagination(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size)
}

// TestParsePagination_CustomValues 测试自定义分页参数
func TestParsePagination_CustomValues(t *testing.T) {
	router := setupTestRouter()
	var page, size int

	router.GET("/test", func(c *gin.Context) {
		page, size = parsePagination(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=3&size=50", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 3, page)
	assert.Equal(t, 50, size)
}

// TestParsePagination_InvalidValues 测试无效分页参数 (回退默认)
func TestParsePagination_InvalidValues(t *testing.T) {
	router := setupTestRouter()
	var page, size int

	router.GET("/test", func(c *gin.Context) {
		page, size = parsePagination(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=abc&size=-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size)
}

// TestParsePagination_SizeTooLarge 测试 size 超过最大值 (回退默认)
func TestParsePagination_SizeTooLarge(t *testing.T) {
	router := setupTestRouter()
	var page, size int

	router.GET("/test", func(c *gin.Context) {
		page, size = parsePagination(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=1&size=200", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size) // 200 > 100, 回退默认 20
}

// TestParsePagination_ZeroPage 测试 page=0 (回退默认)
func TestParsePagination_ZeroPage(t *testing.T) {
	router := setupTestRouter()
	var page, size int

	router.GET("/test", func(c *gin.Context) {
		page, size = parsePagination(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=0", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 1, page)
	assert.Equal(t, 20, size)
}

// TestParseIDParam_Valid 测试有效 ID 参数
func TestParseIDParam_Valid(t *testing.T) {
	router := setupTestRouter()
	var parsedID uint64
	var parseErr error

	router.GET("/test/:id", func(c *gin.Context) {
		parsedID, parseErr = parseIDParam(c, "id")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/123", nil)
	router.ServeHTTP(w, req)

	assert.NoError(t, parseErr)
	assert.Equal(t, uint64(123), parsedID)
}

// TestParseIDParam_Invalid 测试无效 ID 参数
func TestParseIDParam_Invalid(t *testing.T) {
	router := setupTestRouter()
	var parseErr error

	router.GET("/test/:id", func(c *gin.Context) {
		_, parseErr = parseIDParam(c, "id")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/abc", nil)
	router.ServeHTTP(w, req)

	assert.Error(t, parseErr)
}

// TestGetAdminID_Present 测试 context 中存在 admin_id
func TestGetAdminID_Present(t *testing.T) {
	router := setupTestRouter()
	var adminID uint64

	router.GET("/test", func(c *gin.Context) {
		c.Set("admin_id", uint64(42))
		adminID = getAdminID(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, uint64(42), adminID)
}

// TestGetAdminID_Missing 测试 context 中缺少 admin_id
func TestGetAdminID_Missing(t *testing.T) {
	router := setupTestRouter()
	var adminID uint64

	router.GET("/test", func(c *gin.Context) {
		adminID = getAdminID(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, uint64(0), adminID)
}

// TestGetAdminID_WrongType 测试 context 中 admin_id 类型不匹配
func TestGetAdminID_WrongType(t *testing.T) {
	router := setupTestRouter()
	var adminID uint64

	router.GET("/test", func(c *gin.Context) {
		c.Set("admin_id", "not_a_uint64")
		adminID = getAdminID(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, uint64(0), adminID)
}

// TestGetAdminRole_Present 测试 context 中存在 admin_role
func TestGetAdminRole_Present(t *testing.T) {
	router := setupTestRouter()
	var adminRole string

	router.GET("/test", func(c *gin.Context) {
		c.Set("admin_role", "super_admin")
		adminRole = getAdminRole(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "super_admin", adminRole)
}

// TestGetAdminRole_Missing 测试 context 中缺少 admin_role
func TestGetAdminRole_Missing(t *testing.T) {
	router := setupTestRouter()
	var adminRole string

	router.GET("/test", func(c *gin.Context) {
		adminRole = getAdminRole(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, "", adminRole)
}
