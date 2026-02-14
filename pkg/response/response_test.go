package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// 辅助函数：创建 gin 测试上下文并返回 recorder
func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

// 辅助函数：从 recorder 解析 JSON 响应
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) Response {
	t.Helper()
	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "响应 JSON 解析失败")
	return resp
}

// ---------- Success 测试 ----------

func TestSuccess(t *testing.T) {
	c, w := createTestContext()
	data := map[string]string{"name": "test"}

	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.NotNil(t, resp.Data)

	// 验证 data 内容
	dataMap, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test", dataMap["name"])
}

func TestSuccess_NilData(t *testing.T) {
	c, w := createTestContext()

	Success(c, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)
	assert.Nil(t, resp.Data)
}

func TestSuccess_SliceData(t *testing.T) {
	c, w := createTestContext()
	data := []string{"a", "b", "c"}

	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	items, ok := resp.Data.([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 3)
}

// ---------- Created 测试 ----------

func TestCreated(t *testing.T) {
	c, w := createTestContext()
	data := map[string]int{"id": 1}

	Created(c, data)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "created", resp.Message)
	assert.NotNil(t, resp.Data)
}

// ---------- SuccessWithPagination 测试 ----------

func TestSuccessWithPagination(t *testing.T) {
	c, w := createTestContext()
	list := []string{"item1", "item2"}

	SuccessWithPagination(c, list, 50, 1, 20)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	// 解析分页数据
	dataMap, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(50), dataMap["total"])
	assert.Equal(t, float64(1), dataMap["page"])
	assert.Equal(t, float64(20), dataMap["size"])

	items, ok := dataMap["list"].([]interface{})
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestSuccessWithPagination_EmptyList(t *testing.T) {
	c, w := createTestContext()

	SuccessWithPagination(c, []string{}, 0, 1, 20)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	dataMap, ok := resp.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(0), dataMap["total"])

	items, ok := dataMap["list"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, items)
}

// ---------- Error 测试 ----------

func TestError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "400 错误",
			statusCode: http.StatusBadRequest,
			message:    "参数错误",
		},
		{
			name:       "500 错误",
			statusCode: http.StatusInternalServerError,
			message:    "内部错误",
		},
		{
			name:       "404 错误",
			statusCode: http.StatusNotFound,
			message:    "资源不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := createTestContext()

			Error(c, tt.statusCode, tt.message)

			assert.Equal(t, tt.statusCode, w.Code)
			resp := parseResponse(t, w)
			assert.Equal(t, tt.statusCode, resp.Code)
			assert.Equal(t, tt.message, resp.Message)
			assert.Nil(t, resp.Data)
		})
	}
}

// ---------- ErrorFromAppError 测试 ----------

func TestErrorFromAppError(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *apperrors.AppError
		wantCode int
		wantMsg  string
	}{
		{
			name:     "BadRequest 错误",
			appErr:   apperrors.NewBadRequest("输入不合法"),
			wantCode: http.StatusBadRequest,
			wantMsg:  "输入不合法",
		},
		{
			name:     "NotFound 错误",
			appErr:   apperrors.NewNotFound("用户", 42),
			wantCode: http.StatusNotFound,
			wantMsg:  "用户 (ID: 42) 不存在",
		},
		{
			name:     "Unauthorized 错误",
			appErr:   apperrors.NewUnauthorized("token 已过期"),
			wantCode: http.StatusUnauthorized,
			wantMsg:  "token 已过期",
		},
		{
			name:     "Forbidden 错误",
			appErr:   apperrors.NewForbidden("权限不足"),
			wantCode: http.StatusForbidden,
			wantMsg:  "权限不足",
		},
		{
			name:     "Internal 错误不暴露内部详情",
			appErr:   apperrors.NewInternal("服务器错误", nil),
			wantCode: http.StatusInternalServerError,
			wantMsg:  "服务器错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, w := createTestContext()

			ErrorFromAppError(c, tt.appErr)

			assert.Equal(t, tt.wantCode, w.Code)
			resp := parseResponse(t, w)
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Equal(t, tt.wantMsg, resp.Message)
			assert.Nil(t, resp.Data)
		})
	}
}

// ---------- 便捷错误函数测试 ----------

func TestBadRequest(t *testing.T) {
	c, w := createTestContext()
	BadRequest(c, "参数缺失")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "参数缺失", resp.Message)
}

func TestUnauthorized(t *testing.T) {
	c, w := createTestContext()
	Unauthorized(c, "请先登录")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Equal(t, "请先登录", resp.Message)
}

func TestForbidden(t *testing.T) {
	c, w := createTestContext()
	Forbidden(c, "无权访问")
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, "无权访问", resp.Message)
}

func TestNotFound(t *testing.T) {
	c, w := createTestContext()
	NotFound(c, "找不到资源")
	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "找不到资源", resp.Message)
}

func TestInternalError(t *testing.T) {
	c, w := createTestContext()
	InternalError(c, "服务器异常")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Equal(t, "服务器异常", resp.Message)
}

// ---------- 响应 Content-Type 验证 ----------

func TestResponse_ContentType(t *testing.T) {
	c, w := createTestContext()
	Success(c, nil)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// ---------- Response 结构体 JSON 序列化验证 ----------

func TestResponse_JSONStructure(t *testing.T) {
	c, w := createTestContext()
	Success(c, "hello")

	// 直接解析为 map 检查字段
	var raw map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &raw)
	require.NoError(t, err)

	// 必须包含 code, message, data 三个字段
	_, hasCode := raw["code"]
	_, hasMessage := raw["message"]
	_, hasData := raw["data"]
	assert.True(t, hasCode, "响应缺少 code 字段")
	assert.True(t, hasMessage, "响应缺少 message 字段")
	assert.True(t, hasData, "响应缺少 data 字段")

	// 不应有多余字段
	assert.Len(t, raw, 3, "响应应只有 code/message/data 三个字段")
}
