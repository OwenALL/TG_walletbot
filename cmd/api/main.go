// Package main API 服务入口
// 提供管理后台 RESTful API
package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/handler"
	"github.com/TGlimmer/TG_walletbot/internal/adapter/middleware"
	"github.com/TGlimmer/TG_walletbot/internal/adapter/repository"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/config"
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/database"
	infralogger "github.com/TGlimmer/TG_walletbot/internal/infrastructure/logger"
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/server"
	pkglogger "github.com/TGlimmer/TG_walletbot/pkg/logger"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 验证 API 服务必需的安全配置 (JWT Secret 长度、弱密钥检查)
	if err := cfg.ValidateForAPI(); err != nil {
		fmt.Fprintf(os.Stderr, "配置安全检查失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger, err := infralogger.New(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	pkglogger.Init(logger)

	// 弱密钥警告 (长度够但在常见列表中的情况不会到达这里，已在 ValidateForAPI 中拦截)
	if config.IsWeakJWTSecret(cfg.JWT.Secret) {
		logger.Warn("JWT Secret 为弱密钥，强烈建议更换为高强度随机字符串")
	}

	logger.Info("WalletBot Admin API 服务启动中...",
		zap.String("env", cfg.App.Env),
		zap.Int("port", cfg.Server.Port),
	)

	// 连接数据库
	db, err := database.NewMySQL(&cfg.Database, logger)
	if err != nil {
		logger.Fatal("连接数据库失败", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 连接 Redis
	rdb, err := database.NewRedis(&cfg.Redis, logger)
	if err != nil {
		logger.Fatal("连接 Redis 失败", zap.Error(err))
	}
	defer rdb.Close()

	// 初始化依赖容器
	container := app.NewContainer(db, rdb, logger)
	container.UserRepo = repository.NewUserRepo(db)
	container.WalletRepo = repository.NewWalletRepo(db)
	container.TransactionRepo = repository.NewTransactionRepo(db)
	container.DepositRepo = repository.NewDepositRepo(db)
	container.WithdrawalRepo = repository.NewWithdrawalRepo(db)
	container.CNYWithdrawalRepo = repository.NewCNYWithdrawalRepo(db)
	container.TransferRepo = repository.NewTransferRepo(db)
	container.ExchangeRepo = repository.NewExchangeRepo(db)
	container.ExchangeRateRepo = repository.NewExchangeRateRepo(db)
	container.RedPacketRepo = repository.NewRedPacketRepo(db)
	container.RedPacketClaimRepo = repository.NewRedPacketClaimRepo(db)
	container.RedPacketCoverRepo = repository.NewRedPacketCoverRepo(db)
	container.UserSettingsRepo = repository.NewUserSettingsRepo(db)
	container.FinanceInvestmentRepo = repository.NewFinanceInvestmentRepo(db)
	container.SystemConfigRepo = repository.NewSystemConfigRepo(db)
	container.AdminUserRepo = repository.NewAdminUserRepo(db)
	container.AdminLogRepo = repository.NewAdminLogRepo(db)
	container.CacheRepo = repository.NewCacheRepo(rdb)
	container.MerchantRepo = repository.NewMerchantRepo(db)

	// 初始化 Gin 引擎
	engine := server.NewGin(cfg, logger)

	// 全局中间件
	engine.Use(middleware.CORS())

	// 处理器
	healthHandler := handler.NewHealthHandler()

	// API v1 路由组
	v1 := engine.Group("/api/v1")
	{
		// 健康检查 (无需认证)
		v1.GET("/health", healthHandler.Check)

		// 注册管理后台路由 (认证/仪表盘/用户/交易/提币/汇率/配置/审计日志)
		handler.RegisterAdminRoutes(v1, cfg, container, db, logger)
	}

	// 启动服务器
	if err := server.Run(engine, &cfg.Server, logger); err != nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}
}
