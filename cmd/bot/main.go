// Package main Bot 服务入口
// 运行 Telegram Bot，处理消息和命令
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/adapter/notification"
	"github.com/OwenALL/TG_walletbot/internal/adapter/repository"
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	tgadapter "github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram/handler"
	"github.com/OwenALL/TG_walletbot/internal/app"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/config"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/database"
	infralogger "github.com/OwenALL/TG_walletbot/internal/infrastructure/logger"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/exchange"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/task"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/tron"
	pkglogger "github.com/OwenALL/TG_walletbot/pkg/logger"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
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

	// 初始化日志
	logger, err := infralogger.New(&cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	pkglogger.Init(logger)

	logger.Info("WalletBot 服务启动中...",
		zap.String("mode", cfg.Bot.Mode),
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

	// --- 初始化 Repository ---
	userRepo := repository.NewUserRepo(db)
	walletRepo := repository.NewWalletRepo(db)
	transactionRepo := repository.NewTransactionRepo(db)
	depositRepo := repository.NewDepositRepo(db)
	withdrawalRepo := repository.NewWithdrawalRepo(db)
	transferRepo := repository.NewTransferRepo(db)
	exchangeRepo := repository.NewExchangeRepo(db)
	exchangeRateRepo := repository.NewExchangeRateRepo(db)
	userSettingsRepo := repository.NewUserSettingsRepo(db)
	systemConfigRepo := repository.NewSystemConfigRepo(db)
	cacheRepo := repository.NewCacheRepo(rdb)
	redPacketRepo := repository.NewRedPacketRepo(db)
	redPacketClaimRepo := repository.NewRedPacketClaimRepo(db)
	redPacketCoverRepo := repository.NewRedPacketCoverRepo(db)

	// --- 初始化 WalletKeyRepo ---
	walletKeyRepo := repository.NewWalletKeyRepo(db)

	// --- 从 DB 读取 TRON 运营参数 ---
	bootCtx := context.Background()
	tronOps, err := systemConfigRepo.GetMultiple(bootCtx, []string{
		entity.ConfigTronUSDTContract,
		entity.ConfigTronFeeLimit,
		entity.ConfigTronGRPCTimeout,
		entity.ConfigTronGRPCBatchTimeout,
	})
	if err != nil {
		logger.Warn("读取 TRON 运营参数失败，使用默认值", zap.Error(err))
		tronOps = make(map[string]string)
	}
	usdtContract := getConfigStr(tronOps, entity.ConfigTronUSDTContract, tron.USDTContractAddress)
	feeLimit := getConfigInt64(tronOps, entity.ConfigTronFeeLimit, 100_000_000)
	grpcTimeout := time.Duration(getConfigInt(tronOps, entity.ConfigTronGRPCTimeout, 10)) * time.Second
	grpcBatchTimeout := time.Duration(getConfigInt(tronOps, entity.ConfigTronGRPCBatchTimeout, 30)) * time.Second

	// --- 初始化 TRON 基础设施 (受 cfg.Tron.Enabled 开关保护) ---
	var addrGen port.AddressGenerator
	var cryptoSvc port.PrivateKeyCrypto
	var withdrawalSender port.WithdrawalSender
	var walletKeyPort port.WalletKeyRepository

	if cfg.Tron.Enabled {
		logger.Info("TRON 链上集成已启用",
			zap.String("usdt_contract", usdtContract),
			zap.Int64("fee_limit", feeLimit),
			zap.Duration("grpc_timeout", grpcTimeout),
			zap.Duration("grpc_batch_timeout", grpcBatchTimeout),
		)

		tronCryptoSvc := tron.NewCryptoService()
		addrGen = tron.NewKeyGenerator()
		cryptoSvc = tronCryptoSvc
		walletKeyPort = walletKeyRepo

		// 只有配置了热钱包才启用自动发送
		if cfg.Tron.HotWalletAddress != "" && cfg.Tron.HotWalletEncryptedKey != "" {
			tronClient := tron.NewClient(cfg.Tron.APIURL, cfg.Tron.APIKey, feeLimit, logger)
			withdrawalSender = tron.NewWithdrawalSender(
				tronClient,
				tronCryptoSvc,
				cfg.Tron.HotWalletAddress,
				cfg.Tron.HotWalletEncryptedKey,
				cfg.Tron.EncryptionKey,
				systemConfigRepo,
				usdtContract,
				logger,
			)
			logger.Info("TRON 自动提币发送器已初始化",
				zap.String("hot_wallet", cfg.Tron.HotWalletAddress),
			)
		} else {
			logger.Warn("TRON 热钱包未配置，自动提币发送功能未启用")
		}
	} else {
		logger.Info("TRON 链上集成未启用")
	}

	// --- 初始化 Domain Service ---
	userSvc := service.NewUserService(userRepo, walletRepo, userSettingsRepo, systemConfigRepo)
	walletSvc := service.NewWalletService(walletRepo)
	depositSvc := service.NewDepositService(depositRepo, walletRepo, transactionRepo, systemConfigRepo)
	withdrawalSvc := service.NewWithdrawalService(withdrawalRepo, walletRepo, transactionRepo, systemConfigRepo)
	transferSvc := service.NewTransferService(db, walletRepo, transferRepo, transactionRepo, userRepo, logger)
	exchangeSvc := service.NewExchangeService(db, exchangeRateRepo, exchangeRepo, walletRepo, transactionRepo, cacheRepo, logger)
	profileSvc := service.NewProfileService(userRepo, walletRepo, userSettingsRepo, transactionRepo, withdrawalRepo)
	financeRepo := repository.NewFinanceInvestmentRepo(db)
	financeSvc := service.NewFinanceService(db, financeRepo, walletRepo, transactionRepo, systemConfigRepo, logger)
	merchantRepo := repository.NewMerchantRepo(db)
	merchantSvc := service.NewMerchantService(merchantRepo, logger)
	redPacketSvc := service.NewRedPacketService(db, redPacketRepo, redPacketClaimRepo, redPacketCoverRepo, walletRepo, transactionRepo, cacheRepo, systemConfigRepo, logger)

	// --- 初始化 App Use Cases ---
	userUC := app.NewUserUseCase(userSvc, walletSvc, depositSvc, addrGen, walletKeyPort, cryptoSvc, cfg.Tron.EncryptionKey, logger)
	depositUC := app.NewDepositUseCase(depositSvc, walletSvc, logger)
	withdrawalUC := app.NewWithdrawalUseCase(withdrawalSvc, walletSvc, userSvc, withdrawalSender, logger)
	transferUC := app.NewTransferUseCase(transferSvc, walletSvc, userSvc, logger)
	exchangeUC := app.NewExchangeUseCase(exchangeSvc, walletSvc, logger)
	profileUC := app.NewProfileUseCase(profileSvc, userSvc, logger)
	financeUC := app.NewFinanceUseCase(financeSvc, walletSvc, logger)
	merchantUC := app.NewMerchantUseCase(merchantSvc, logger)
	redPacketUC := app.NewRedPacketUseCase(redPacketSvc, walletSvc, userSvc, logger)

	// --- 初始化 Session 服务 ---
	sessionSvc := tgpkg.NewSessionService(rdb)

	// --- 初始化 Handler ---
	pinHandler := handler.NewPINHandler(userUC, sessionSvc, logger)
	inlinePayHandler := handler.NewInlinePayHandler(transferUC, userUC, pinHandler, sessionSvc, logger)
	startHandler := handler.NewStartHandler(userUC, sessionSvc, inlinePayHandler, logger)
	simpleMenuHandler := handler.NewSimpleMenuHandler(userUC, sessionSvc, logger)
	menuCallbackHandler := handler.NewMenuCallbackHandler(userUC, sessionSvc, logger)
	depositHandler := handler.NewDepositHandler(depositUC, logger)
	withdrawHandler := handler.NewWithdrawHandler(withdrawalUC, userUC, pinHandler, sessionSvc, logger)
	walletCallbackHandler := handler.NewWalletCallbackHandler(depositHandler, withdrawHandler, pinHandler, userUC, logger)
	receiveHandler := handler.NewReceiveHandler(transferUC, userUC, pinHandler, sessionSvc, logger)
	transferHandler := handler.NewTransferHandler(transferUC, userUC, pinHandler, receiveHandler, sessionSvc, logger)
	exchangeHandler := handler.NewExchangeHandler(exchangeUC, userUC, pinHandler, sessionSvc, logger)
	profileHandler := handler.NewProfileHandler(profileUC, userUC, pinHandler, sessionSvc, logger)
	financeHandler := handler.NewFinanceHandler(financeUC, userUC, pinHandler, sessionSvc, logger)
	merchantHandler := handler.NewMerchantHandler(merchantUC, userUC, pinHandler, sessionSvc, logger)
	settingHandler := handler.NewSettingHandler(userUC, profileUC, sessionSvc, logger)
	redPacketHandler := handler.NewRedPacketHandler(redPacketUC, userUC, pinHandler, sessionSvc, logger)
	inlineQueryHandler := handler.NewInlineQueryHandler(redPacketUC, transferUC, userUC, logger)
	defaultMsgHandler := handler.NewDefaultMessageHandler(pinHandler, withdrawHandler, transferHandler, receiveHandler, exchangeHandler, profileHandler, financeHandler, merchantHandler, redPacketHandler, userUC, sessionSvc, logger)

	// 验证 Bot Token
	if cfg.Bot.Token == "" {
		logger.Fatal("BOT_TOKEN 未配置")
	}

	// --- 创建路由器 ---
	router := tgadapter.NewRouter(logger)

	// 注册模块回调处理器
	router.RegisterModule("Menu", menuCallbackHandler.Handle)
	router.RegisterModule("Pin", pinHandler.HandleCallback)
	router.RegisterModule("Wallet", walletCallbackHandler.Handle)
	handler.RegisterTransferHandlers(router, transferHandler)
	handler.RegisterReceiveHandlers(router, receiveHandler)
	handler.RegisterExchangeHandlers(router, exchangeHandler)
	handler.RegisterProfileHandlers(router, profileHandler)
	// 未开发模块注册占位处理器 (按钮可点击，提示"即将上线")
	placeholderHandler := handler.NewPlaceholderHandler(logger)
	handler.RegisterFinanceHandlers(router, financeHandler)
	handler.RegisterMerchantHandlers(router, merchantHandler)
	handler.RegisterSettingHandlers(router, settingHandler)
	handler.RegisterRedPacketHandlers(router, redPacketHandler)
	handler.RegisterInlineCallbackHandlers(router, inlineQueryHandler)
	handler.RegisterInlinePayHandlers(router, inlinePayHandler)
	router.RegisterModule("Service", placeholderHandler.Handle)

	// --- 构建中间件链 ---
	// 中间件执行顺序: Recovery → Logging → Maintenance → RateLimit → Auth
	middlewareChain := []tgadapter.Middleware{
		tgadapter.RecoveryMiddleware(logger),
		tgadapter.LoggingMiddleware(logger),
		tgadapter.MaintenanceMiddleware(systemConfigRepo),
		tgadapter.RateLimitMiddleware(rdb, 30, 60*time.Second),
		tgadapter.AuthMiddleware(userRepo),
	}

	// 包装默认消息处理器 (经过完整中间件链)
	// 同时处理 Inline Query (go-telegram/bot 不支持 HandlerTypeInlineQuery，
	// 需在默认处理器中分发)
	wrappedDefault := tgadapter.ApplyMiddleware(
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// Inline Query 走红包 Inline 处理器
			if update.InlineQuery != nil {
				inlineQueryHandler.Handle(ctx, b, update)
				return
			}
			defaultMsgHandler.Handle(ctx, b, update)
		},
		middlewareChain...,
	)

	// 创建 Bot 实例
	opts := []bot.Option{
		bot.WithDefaultHandler(router.DefaultHandler(wrappedDefault)),
	}

	b, err := bot.New(cfg.Bot.Token, opts...)
	if err != nil {
		logger.Fatal("创建 Bot 失败", zap.Error(err))
	}

	// --- 初始化通知服务 (依赖 Bot 实例) ---
	notifier := notification.NewTelegramNotifier(b, logger)

	// --- 初始化 TRON 充值监控 (gRPC 区块扫描模式) ---
	var depositMonitor *tron.DepositMonitor
	if cfg.Tron.Enabled {
		grpcClient := tron.NewGrpcBlockClient(cfg.Tron.GRPCEndpoint, cfg.Tron.APIKey, grpcTimeout, grpcBatchTimeout, logger)
		depositMonitor = tron.NewDepositMonitor(
			grpcClient,
			depositUC,
			walletRepo,
			userRepo,
			notifier,
			cacheRepo,
			usdtContract,
			logger,
		)
		logger.Info("TRON 充值监控服务已初始化 (gRPC 区块扫描模式)")
	}

	// --- 注册命令处理器 (经过中间件链) ---
	startHandlerWrapped := tgadapter.ApplyMiddleware(startHandler.Handle, middlewareChain...)
	simpleMenuHandlerWrapped := tgadapter.ApplyMiddleware(simpleMenuHandler.Handle, middlewareChain...)

	commandHandlers := map[string]tgadapter.HandlerFunc{
		"start": startHandlerWrapped,
	}

	// 仅注册用户可见的 4 个命令 (help/support/privacy 返回精简菜单)
	for _, cmd := range []string{"help", "support", "privacy"} {
		commandHandlers[cmd] = simpleMenuHandlerWrapped
	}

	// 注册所有处理器到 Bot 实例 (回调处理器也经过完整中间件链)
	router.RegisterHandlers(b, commandHandlers, middlewareChain...)

	// 优雅关闭
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 启动 TRON 充值监控 (后台协程)
	if depositMonitor != nil {
		go depositMonitor.Start(ctx)
	}

	// 启动 CoinGecko 汇率自动更新 (后台协程, 每 5 分钟)
	exchangeUpdater := exchange.NewExchangeUpdater(exchangeRateRepo, cacheRepo, logger, 5*time.Minute)
	go exchangeUpdater.Start(ctx)

	// 启动余额宝每日派息任务 (后台协程, 每 1 小时检查)
	financeProfitTask := task.NewFinanceProfitTask(financeSvc, rdb, logger, 1*time.Hour)
	go financeProfitTask.Start(ctx)

	// 启动红包过期退款任务 (后台协程, 每 1 分钟检查)
	redPacketExpiryTask := task.NewRedPacketExpiryTask(redPacketSvc, logger, 1*time.Minute)
	go redPacketExpiryTask.Start(ctx)

	logger.Info("WalletBot 服务已启动")

	// 启动 Bot (长轮询模式)
	b.Start(ctx)

	logger.Info("WalletBot 服务已停止")
}

// getConfigStr 从配置 map 获取字符串值，不存在或为空则返回默认值
func getConfigStr(m map[string]string, key, defaultVal string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

// getConfigInt64 从配置 map 获取 int64 值，不存在或解析失败则返回默认值
func getConfigInt64(m map[string]string, key string, defaultVal int64) int64 {
	if v, ok := m[key]; ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return defaultVal
}

// getConfigInt 从配置 map 获取 int 值，不存在或解析失败则返回默认值
func getConfigInt(m map[string]string, key string, defaultVal int) int {
	if v, ok := m[key]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
