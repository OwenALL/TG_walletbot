package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// apiResponse 用于反序列化统一响应结构
type apiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

const testSecret = "test-jwt-secret-key-for-unit-testing"

func init() {
	gin.SetMode(gin.TestMode)
}

// ========== GenerateToken / ParseToken 单元测试 ==========

func TestGenerateToken_成功生成有效Token(t *testing.T) {
	token, err := GenerateToken(1, "admin", "super_admin", testSecret, 24)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// 验证生成的 token 可以被正确解析
	claims, err := ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), claims.AdminID)
	assert.Equal(t, "admin", claims.Username)
	assert.Equal(t, "super_admin", claims.Role)
}

func TestGenerateToken_不同参数生成不同Token(t *testing.T) {
	token1, err := GenerateToken(1, "admin1", "admin", testSecret, 24)
	require.NoError(t, err)

	token2, err := GenerateToken(2, "admin2", "viewer", testSecret, 24)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestParseToken_使用错误密钥解析失败(t *testing.T) {
	token, err := GenerateToken(1, "admin", "super_admin", testSecret, 24)
	require.NoError(t, err)

	claims, err := ParseToken(token, "wrong-secret-key")
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestParseToken_过期Token解析失败(t *testing.T) {
	// 手动构建一个已过期的 token
	claims := AdminClaims{
		AdminID:  1,
		Username: "admin",
		Role:     "super_admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 1小时前过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	result, err := ParseToken(tokenString, testSecret)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestParseToken_无效Token格式(t *testing.T) {
	invalidTokens := []string{
		"",
		"not-a-jwt-token",
		"eyJhbGciOiJIUzI1NiJ9.invalid.payload",
		"abc.def.ghi",
	}

	for _, tk := range invalidTokens {
		claims, err := ParseToken(tk, testSecret)
		assert.Error(t, err, "对于无效 token: %s 应返回错误", tk)
		assert.Nil(t, claims, "对于无效 token: %s claims 应为 nil", tk)
	}
}

func TestParseToken_Claims字段完整性(t *testing.T) {
	token, err := GenerateToken(42, "testuser", "editor", testSecret, 12)
	require.NoError(t, err)

	claims, err := ParseToken(token, testSecret)
	require.NoError(t, err)

	assert.Equal(t, uint64(42), claims.AdminID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "editor", claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	// 过期时间应在未来
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
}

// ========== JWTAuth 中间件测试 ==========

// setupRouter 创建带有 JWTAuth 中间件的测试路由
func setupJWTRouter() *gin.Engine {
	r := gin.New()
	r.Use(JWTAuth(testSecret))
	r.GET("/protected", func(c *gin.Context) {
		adminID, _ := c.Get("admin_id")
		username, _ := c.Get("admin_username")
		role, _ := c.Get("admin_role")
		c.JSON(http.StatusOK, gin.H{
			"admin_id": adminID,
			"username": username,
			"role":     role,
		})
	})
	return r
}

func TestJWTAuth_缺少Authorization头(t *testing.T) {
	router := setupJWTRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Equal(t, "缺少认证信息", resp.Message)
}

func TestJWTAuth_Authorization头为空字符串(t *testing.T) {
	router := setupJWTRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_格式错误_缺少Bearer前缀(t *testing.T) {
	token, _ := GenerateToken(1, "admin", "super_admin", testSecret, 24)

	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", token) // 缺少 "Bearer " 前缀
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "认证格式错误", resp.Message)
}

func TestJWTAuth_格式错误_使用Basic前缀(t *testing.T) {
	token, _ := GenerateToken(1, "admin", "super_admin", testSecret, 24)

	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "认证格式错误", resp.Message)
}

func TestJWTAuth_无效Token(t *testing.T) {
	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "认证已过期或无效", resp.Message)
}

func TestJWTAuth_过期Token(t *testing.T) {
	// 构建已过期的 token
	claims := AdminClaims{
		AdminID:  1,
		Username: "admin",
		Role:     "super_admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString([]byte(testSecret))

	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "认证已过期或无效", resp.Message)
}

func TestJWTAuth_错误密钥签名的Token(t *testing.T) {
	// 使用不同密钥签名
	token, _ := GenerateToken(1, "admin", "super_admin", "different-secret", 24)

	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_有效Token_上下文注入(t *testing.T) {
	token, _ := GenerateToken(99, "superadmin", "super_admin", testSecret, 24)

	router := setupJWTRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	// JSON number 默认解码为 float64
	assert.Equal(t, float64(99), body["admin_id"])
	assert.Equal(t, "superadmin", body["username"])
	assert.Equal(t, "super_admin", body["role"])
}

func TestJWTAuth_不同角色Token均正常通过(t *testing.T) {
	roles := []string{"super_admin", "admin", "editor", "viewer"}

	for _, role := range roles {
		t.Run("role_"+role, func(t *testing.T) {
			token, _ := GenerateToken(1, "user", role, testSecret, 24)

			router := setupJWTRouter()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// ========== AdminOnly 中间件测试 ==========

// setupAdminOnlyRouter 创建带有 AdminOnly 中间件的测试路由
func setupAdminOnlyRouter() *gin.Engine {
	r := gin.New()
	// 先通过 JWTAuth，再通过 AdminOnly
	r.Use(JWTAuth(testSecret))
	r.Use(AdminOnly())
	r.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})
	return r
}

func TestAdminOnly_SuperAdmin通过(t *testing.T) {
	token, _ := GenerateToken(1, "superadmin", "super_admin", testSecret, 24)

	router := setupAdminOnlyRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminOnly_Admin通过(t *testing.T) {
	token, _ := GenerateToken(2, "admin", "admin", testSecret, 24)

	router := setupAdminOnlyRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminOnly_Editor通过(t *testing.T) {
	token, _ := GenerateToken(3, "editor", "editor", testSecret, 24)

	router := setupAdminOnlyRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminOnly_Viewer被拒绝(t *testing.T) {
	token, _ := GenerateToken(4, "viewer", "viewer", testSecret, 24)

	router := setupAdminOnlyRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "只读角色无此操作权限", resp.Message)
}

func TestAdminOnly_无认证信息(t *testing.T) {
	r := gin.New()
	// 不使用 JWTAuth，直接使用 AdminOnly（模拟缺少认证信息的场景）
	r.Use(AdminOnly())
	r.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "缺少认证信息", resp.Message)
}

// ========== SuperAdminOnly 中间件测试 ==========

// setupSuperAdminRouter 创建带有 SuperAdminOnly 中间件的测试路由
func setupSuperAdminRouter() *gin.Engine {
	r := gin.New()
	r.Use(JWTAuth(testSecret))
	r.Use(SuperAdminOnly())
	r.GET("/super-admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "super admin access granted"})
	})
	return r
}

func TestSuperAdminOnly_SuperAdmin通过(t *testing.T) {
	token, _ := GenerateToken(1, "superadmin", "super_admin", testSecret, 24)

	router := setupSuperAdminRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSuperAdminOnly_Admin被拒绝(t *testing.T) {
	token, _ := GenerateToken(2, "admin", "admin", testSecret, 24)

	router := setupSuperAdminRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "需要超级管理员权限", resp.Message)
}

func TestSuperAdminOnly_Editor被拒绝(t *testing.T) {
	token, _ := GenerateToken(3, "editor", "editor", testSecret, 24)

	router := setupSuperAdminRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSuperAdminOnly_Viewer被拒绝(t *testing.T) {
	token, _ := GenerateToken(4, "viewer", "viewer", testSecret, 24)

	router := setupSuperAdminRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSuperAdminOnly_无认证信息(t *testing.T) {
	r := gin.New()
	r.Use(SuperAdminOnly())
	r.GET("/super-admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp apiResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "缺少认证信息", resp.Message)
}

// ========== 表驱动测试: JWTAuth 完整场景 ==========

func TestJWTAuth_表驱动测试(t *testing.T) {
	validToken, _ := GenerateToken(1, "admin", "super_admin", testSecret, 24)

	// 构建过期 token
	expiredClaims := AdminClaims{
		AdminID:  1,
		Username: "admin",
		Role:     "super_admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredJwt := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredToken, _ := expiredJwt.SignedString([]byte(testSecret))

	// 使用错误密钥的 token
	wrongKeyToken, _ := GenerateToken(1, "admin", "super_admin", "wrong-key", 24)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "无Authorization头",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "缺少认证信息",
		},
		{
			name:           "只有Bearer无Token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证已过期或无效",
		},
		{
			name:           "缺少Bearer前缀",
			authHeader:     validToken,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证格式错误",
		},
		{
			name:           "使用Basic前缀",
			authHeader:     "Basic " + validToken,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证格式错误",
		},
		{
			name:           "无效token字符串",
			authHeader:     "Bearer abc123",
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证已过期或无效",
		},
		{
			name:           "过期token",
			authHeader:     "Bearer " + expiredToken,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证已过期或无效",
		},
		{
			name:           "错误密钥签名的token",
			authHeader:     "Bearer " + wrongKeyToken,
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "认证已过期或无效",
		},
		{
			name:           "有效token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			expectedMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupJWTRouter()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedMsg != "" {
				var resp apiResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, resp.Message)
			}
		})
	}
}

// ========== 表驱动测试: 权限中间件 ==========

func TestAdminOnly_表驱动测试(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		expectedStatus int
	}{
		{"super_admin通过", "super_admin", http.StatusOK},
		{"admin通过", "admin", http.StatusOK},
		{"editor通过", "editor", http.StatusOK},
		{"viewer被拒绝", "viewer", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := GenerateToken(1, "user", tt.role, testSecret, 24)

			router := setupAdminOnlyRouter()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/admin-only", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSuperAdminOnly_表驱动测试(t *testing.T) {
	tests := []struct {
		name           string
		role           string
		expectedStatus int
	}{
		{"super_admin通过", "super_admin", http.StatusOK},
		{"admin被拒绝", "admin", http.StatusForbidden},
		{"editor被拒绝", "editor", http.StatusForbidden},
		{"viewer被拒绝", "viewer", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _ := GenerateToken(1, "user", tt.role, testSecret, 24)

			router := setupSuperAdminRouter()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/super-admin-only", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
