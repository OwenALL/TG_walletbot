// Package app 应用层 (用例编排层)
// 协调 domain service 和 port 完成具体用例
package app

import (
	"github.com/redis/go-redis/v9"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Container 依赖容器，持有所有 Repository 和 Service 引用
type Container struct {
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *zap.Logger

	// Repositories
	UserRepo             port.UserRepository
	WalletRepo           port.WalletRepository
	TransactionRepo      port.TransactionRepository
	DepositRepo          port.DepositRepository
	WithdrawalRepo       port.WithdrawalRepository
	CNYWithdrawalRepo    port.CNYWithdrawalRepository
	TransferRepo         port.TransferRepository
	ExchangeRepo         port.ExchangeRepository
	ExchangeRateRepo     port.ExchangeRateRepository
	RedPacketRepo        port.RedPacketRepository
	RedPacketClaimRepo   port.RedPacketClaimRepository
	RedPacketCoverRepo   port.RedPacketCoverRepository
	UserSettingsRepo     port.UserSettingsRepository
	SubAccountRepo       port.SubAccountRepository
	FinanceInvestmentRepo port.FinanceInvestmentRepository
	SystemConfigRepo     port.SystemConfigRepository
	AdminUserRepo        port.AdminUserRepository
	AdminLogRepo         port.AdminLogRepository
	CacheRepo            port.CacheRepository
	MerchantRepo         port.MerchantRepository
}

// NewContainer 创建依赖容器
func NewContainer(db *gorm.DB, rdb *redis.Client, logger *zap.Logger) *Container {
	return &Container{
		DB:     db,
		Redis:  rdb,
		Logger: logger,
	}
}
