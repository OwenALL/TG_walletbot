package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- Params.Offset() 测试 ----------

func TestParams_Offset(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		size       int
		wantOffset int
	}{
		{
			name:       "第一页",
			page:       1,
			size:       20,
			wantOffset: 0,
		},
		{
			name:       "第二页",
			page:       2,
			size:       20,
			wantOffset: 20,
		},
		{
			name:       "第三页 每页10条",
			page:       3,
			size:       10,
			wantOffset: 20,
		},
		{
			name:       "大页码",
			page:       100,
			size:       50,
			wantOffset: 4950,
		},
		{
			name:       "每页1条 第5页",
			page:       5,
			size:       1,
			wantOffset: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Params{Page: tt.page, Size: tt.size}
			assert.Equal(t, tt.wantOffset, p.Offset())
		})
	}
}

// ---------- GetFromContext 测试 ----------

// 辅助函数：创建带查询参数的 gin.Context
func createTestContext(query string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	c.Request = req
	return c
}

func TestGetFromContext_Default(t *testing.T) {
	// 无参数时使用默认值
	c := createTestContext("")
	p := GetFromContext(c)
	assert.Equal(t, DefaultPage, p.Page)
	assert.Equal(t, DefaultSize, p.Size)
}

func TestGetFromContext_NormalValues(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{
			name:     "正常分页参数",
			query:    "page=3&size=15",
			wantPage: 3,
			wantSize: 15,
		},
		{
			name:     "第一页 每页1条",
			query:    "page=1&size=1",
			wantPage: 1,
			wantSize: 1,
		},
		{
			name:     "最大允许 size",
			query:    "page=1&size=100",
			wantPage: 1,
			wantSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			p := GetFromContext(c)
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantSize, p.Size)
		})
	}
}

func TestGetFromContext_BoundaryValues(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{
			name:     "page=0 修正为默认值",
			query:    "page=0&size=20",
			wantPage: DefaultPage,
			wantSize: 20,
		},
		{
			name:     "page 负数修正为默认值",
			query:    "page=-1&size=20",
			wantPage: DefaultPage,
			wantSize: 20,
		},
		{
			name:     "page 极大负数修正为默认值",
			query:    "page=-999&size=20",
			wantPage: DefaultPage,
			wantSize: 20,
		},
		{
			name:     "size=0 修正为默认值",
			query:    "page=1&size=0",
			wantPage: 1,
			wantSize: DefaultSize,
		},
		{
			name:     "size 负数修正为默认值",
			query:    "page=1&size=-10",
			wantPage: 1,
			wantSize: DefaultSize,
		},
		{
			name:     "size 超过 MaxSize 修正为 MaxSize",
			query:    "page=1&size=200",
			wantPage: 1,
			wantSize: MaxSize,
		},
		{
			name:     "size 刚好超过 MaxSize 一个",
			query:    "page=1&size=101",
			wantPage: 1,
			wantSize: MaxSize,
		},
		{
			name:     "page 和 size 都异常",
			query:    "page=-5&size=999",
			wantPage: DefaultPage,
			wantSize: MaxSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			p := GetFromContext(c)
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantSize, p.Size)
		})
	}
}

func TestGetFromContext_InvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{
			name:     "page 非数字使用默认值",
			query:    "page=abc&size=20",
			wantPage: DefaultPage,
			wantSize: 20,
		},
		{
			name:     "size 非数字使用默认值",
			query:    "page=1&size=xyz",
			wantPage: 1,
			wantSize: DefaultSize,
		},
		{
			name:     "两个参数都非数字",
			query:    "page=hello&size=world",
			wantPage: DefaultPage,
			wantSize: DefaultSize,
		},
		{
			name:     "只传 page 不传 size",
			query:    "page=5",
			wantPage: 5,
			wantSize: DefaultSize,
		},
		{
			name:     "只传 size 不传 page",
			query:    "size=30",
			wantPage: DefaultPage,
			wantSize: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			p := GetFromContext(c)
			assert.Equal(t, tt.wantPage, p.Page)
			assert.Equal(t, tt.wantSize, p.Size)
		})
	}
}

// ---------- Offset 与 GetFromContext 联合测试 ----------

func TestGetFromContext_Offset_Integration(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantOffset int
	}{
		{
			name:       "默认参数 offset=0",
			query:      "",
			wantOffset: 0,
		},
		{
			name:       "第2页 每页20条 offset=20",
			query:      "page=2&size=20",
			wantOffset: 20,
		},
		{
			name:       "page=0 修正为1 offset=0",
			query:      "page=0&size=10",
			wantOffset: 0,
		},
		{
			name:       "大页码 offset 正确",
			query:      "page=10&size=50",
			wantOffset: 450,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			p := GetFromContext(c)
			assert.Equal(t, tt.wantOffset, p.Offset())
		})
	}
}

// ---------- 常量值验证 ----------

func TestConstants(t *testing.T) {
	assert.Equal(t, 1, DefaultPage)
	assert.Equal(t, 20, DefaultSize)
	assert.Equal(t, 100, MaxSize)
}
