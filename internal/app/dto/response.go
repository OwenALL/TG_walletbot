package dto

// LoginResponse 管理员登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"` // 秒
}

// DashboardResponse 仪表盘数据响应
type DashboardResponse struct {
	TotalUsers        int64   `json:"total_users"`
	TotalDeposits     float64 `json:"total_deposits"`
	TotalWithdrawals  float64 `json:"total_withdrawals"`
	TotalTransfers    float64 `json:"total_transfers"`
	PendingWithdrawals int64  `json:"pending_withdrawals"`
	TodayNewUsers     int64   `json:"today_new_users"`
}
