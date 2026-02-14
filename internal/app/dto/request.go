// Package dto 定义数据传输对象 (请求/响应)
package dto

// LoginRequest 管理员登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// UpdateExchangeRateRequest 更新汇率请求
type UpdateExchangeRateRequest struct {
	FromCurrency string  `json:"from_currency" binding:"required"`
	ToCurrency   string  `json:"to_currency" binding:"required"`
	Rate         float64 `json:"rate" binding:"required,gt=0"`
	Spread       float64 `json:"spread" binding:"gte=0"`
	MinAmount    float64 `json:"min_amount" binding:"gte=0"`
	MaxAmount    float64 `json:"max_amount" binding:"gte=0"`
	Enabled      *bool   `json:"enabled"`
}

// ReviewWithdrawalRequest 审核提币请求
type ReviewWithdrawalRequest struct {
	Action string `json:"action" binding:"required,oneof=approve reject"` // approve / reject
	Note   string `json:"note"`
}

// UpdateSystemConfigRequest 更新系统配置请求
type UpdateSystemConfigRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status int8 `json:"status" binding:"required,oneof=1 2 3"` // 1:正常 2:冻结 3:封禁
}

// PaginationRequest 分页请求
type PaginationRequest struct {
	Page int `form:"page" binding:"gte=1"`
	Size int `form:"size" binding:"gte=1,lte=100"`
}
