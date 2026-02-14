package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/middleware"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/config"
)

// RegisterAdminRoutes 注册管理后台全部路由
// 由 cmd/api/main.go 调用，传入路由组和依赖
func RegisterAdminRoutes(v1 *gin.RouterGroup, cfg *config.Config, container *app.Container, db *gorm.DB, logger *zap.Logger) {
	// --- 初始化 Domain Service ---
	adminSvc := service.NewAdminService(container.AdminUserRepo, container.AdminLogRepo, logger)

	// --- 初始化 Use Case ---
	adminUC := app.NewAdminUseCase(
		adminSvc,
		container.UserRepo,
		container.WalletRepo,
		container.TransactionRepo,
		container.DepositRepo,
		container.WithdrawalRepo,
		container.ExchangeRateRepo,
		container.SystemConfigRepo,
		db,
		logger,
	)

	// --- 初始化 Handler ---
	authHandler := NewAuthHandler(adminUC, cfg.JWT.Secret, cfg.JWT.ExpireHours, logger)
	dashboardHandler := NewDashboardHandler(adminUC, logger)
	userHandler := NewAdminUserHandler(adminUC, logger)
	transactionHandler := NewAdminTransactionHandler(adminUC, logger)
	withdrawalHandler := NewAdminWithdrawalHandler(adminUC, logger)
	exchangeRateHandler := NewAdminExchangeRateHandler(adminUC, logger)
	configHandler := NewAdminConfigHandler(adminUC, logger)
	auditLogHandler := NewAdminAuditLogHandler(adminUC, logger)

	// --- 初始化新模块 Handler (直接注入 repo，避免 AdminUseCase 膨胀) ---
	adminRedPacketHandler := NewAdminRedPacketHandler(
		container.RedPacketRepo,
		container.RedPacketClaimRepo,
		container.UserRepo,
		db,
		logger,
	)
	adminFinanceHandler := NewAdminFinanceHandler(
		container.FinanceInvestmentRepo,
		container.SystemConfigRepo,
		container.UserRepo,
		container.AdminLogRepo,
		db,
		logger,
	)
	adminMerchantHandler := NewAdminMerchantHandler(
		container.MerchantRepo,
		container.UserRepo,
		container.AdminLogRepo,
		db,
		logger,
	)

	// --- 注册路由 ---

	// 认证路由 (无需 JWT)
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
	}

	// 需要认证的路由
	authed := v1.Group("")
	authed.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		// 当前管理员信息
		authed.GET("/me", authHandler.Me)

		// 修改密码
		authed.PUT("/auth/change-password", authHandler.ChangePassword)

		// 仪表盘
		authed.GET("/dashboard/stats", dashboardHandler.Stats)
		authed.GET("/dashboard/trend", dashboardHandler.Trend)

		// 用户管理
		users := authed.Group("/users")
		{
			users.GET("", userHandler.List)
			users.GET("/:id", userHandler.Detail)
			users.PUT("/:id/status", middleware.AdminOnly(), userHandler.UpdateStatus)
			users.PUT("/:id/reset-pin", middleware.AdminOnly(), userHandler.ResetPIN)
			users.PUT("/:id/adjust-balance", middleware.SuperAdminOnly(), userHandler.AdjustBalance)
			users.GET("/:id/transactions", userHandler.Transactions)
		}

		// 交易管理
		transactions := authed.Group("/transactions")
		{
			transactions.GET("", transactionHandler.List)
			transactions.GET("/deposits", transactionHandler.Deposits)
		}

		// 提币审核
		withdrawals := authed.Group("/withdrawals")
		{
			withdrawals.GET("", withdrawalHandler.List)
			withdrawals.PUT("/:id/approve", middleware.AdminOnly(), withdrawalHandler.Approve)
			withdrawals.PUT("/:id/reject", middleware.AdminOnly(), withdrawalHandler.Reject)
			withdrawals.PUT("/:id/complete", middleware.AdminOnly(), withdrawalHandler.Complete)
		}

		// 汇率管理
		exchangeRates := authed.Group("/exchange-rates")
		{
			exchangeRates.GET("", exchangeRateHandler.List)
			exchangeRates.PUT("", middleware.SuperAdminOnly(), exchangeRateHandler.Update)
		}

		// 系统配置 (仅超级管理员)
		configs := authed.Group("/configs")
		configs.Use(middleware.SuperAdminOnly())
		{
			configs.GET("", configHandler.List)
			configs.PUT("", configHandler.Update)
		}

		// 红包管理
		redPackets := authed.Group("/red-packets")
		{
			redPackets.GET("", adminRedPacketHandler.List)
			redPackets.GET("/stats", adminRedPacketHandler.Stats)
			redPackets.GET("/:id", adminRedPacketHandler.Detail)
		}

		// 余额宝管理
		finance := authed.Group("/finance")
		{
			finance.GET("/investments", adminFinanceHandler.ListInvestments)
			finance.GET("/stats", adminFinanceHandler.Stats)
			finance.PUT("/rate", middleware.SuperAdminOnly(), adminFinanceHandler.UpdateRate)
		}

		// 商户管理
		merchants := authed.Group("/merchants")
		{
			merchants.GET("", adminMerchantHandler.List)
			merchants.PUT("/:id/toggle-status", middleware.AdminOnly(), adminMerchantHandler.ToggleStatus)
			merchants.PUT("/:id/fee-rate", middleware.SuperAdminOnly(), adminMerchantHandler.UpdateFeeRate)
		}

		// 审计日志 (仅管理员及以上)
		authed.GET("/audit-logs", middleware.AdminOnly(), auditLogHandler.List)
	}
}
