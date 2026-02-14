// Package pagination 提供分页查询的通用工具
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPage 默认页码
	DefaultPage = 1
	// DefaultSize 默认每页数量
	DefaultSize = 20
	// MaxSize 最大每页数量
	MaxSize = 100
)

// Params 分页参数
type Params struct {
	Page int `json:"page"` // 页码 (从 1 开始)
	Size int `json:"size"` // 每页数量
}

// Offset 计算数据库查询偏移量
func (p *Params) Offset() int {
	return (p.Page - 1) * p.Size
}

// GetFromContext 从 Gin 上下文中解析分页参数
func GetFromContext(c *gin.Context) *Params {
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(DefaultPage)))
	size, _ := strconv.Atoi(c.DefaultQuery("size", strconv.Itoa(DefaultSize)))

	if page < 1 {
		page = DefaultPage
	}
	if size < 1 {
		size = DefaultSize
	}
	if size > MaxSize {
		size = MaxSize
	}

	return &Params{Page: page, Size: size}
}
