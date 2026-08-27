// Package port 定义领域层端口接口
// 所有适配器（MySQL/Redis 等）必须实现这些接口
package port

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
)

// UserRepository 用户存储端口
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *entity.User) error
	// FindByID 根据 ID 查询用户
	FindByID(ctx context.Context, id uint64) (*entity.User, error)
	// FindByTelegramID 根据 Telegram ID 查询用户
	FindByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error)
	// FindByUsername 根据用户名查询用户
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	// Update 更新用户信息
	Update(ctx context.Context, user *entity.User) error
	// List 获取用户列表 (分页)
	List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error)
	// CountAll 获取总用户数
	CountAll(ctx context.Context) (int64, error)
}

// WalletRepository 钱包存储端口
type WalletRepository interface {
	// Create 创建钱包
	Create(ctx context.Context, wallet *entity.Wallet) error
	// FindByID 根据 ID 查询钱包
	FindByID(ctx context.Context, id uint64) (*entity.Wallet, error)
	// FindByUserIDAndCurrency 根据用户 ID 和币种查询钱包
	FindByUserIDAndCurrency(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error)
	// FindByUserID 获取用户所有钱包
	FindByUserID(ctx context.Context, userID uint64) ([]*entity.Wallet, error)
	// FindByDepositAddress 根据充值地址查询钱包
	FindByDepositAddress(ctx context.Context, address string) (*entity.Wallet, error)
	// UpdateBalance 更新余额 (需在事务中使用)
	UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error
	// FreezeBalance 冻结余额
	FreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error
	// UnfreezeBalance 解冻余额
	UnfreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error
	// UpdateDepositAddress 更新用户所有链上钱包的充值地址
	UpdateDepositAddress(ctx context.Context, userID uint64, address string) error
	// ListAllWithDepositAddress 获取所有已分配充值地址的钱包 (充值监控用)
	ListAllWithDepositAddress(ctx context.Context) ([]*entity.Wallet, error)
	// ListDepositAddressesCreatedAfter 获取指定时间之后创建的充值地址 (增量刷新用)
	ListDepositAddressesCreatedAfter(ctx context.Context, after time.Time) ([]*entity.Wallet, error)
}

// WalletKeyRepository 钱包私钥存储端口
type WalletKeyRepository interface {
	// Create 创建钱包私钥记录
	Create(ctx context.Context, key *entity.WalletKey) error
	// FindByUserID 根据用户 ID 查询钱包私钥
	FindByUserID(ctx context.Context, userID uint64) (*entity.WalletKey, error)
}

// TransactionRepository 交易流水存储端口
type TransactionRepository interface {
	// Create 创建交易记录
	Create(ctx context.Context, tx *entity.Transaction) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.Transaction, error)
	// ListByUserID 获取用户交易列表 (分页、筛选)
	ListByUserID(ctx context.Context, userID uint64, filter *TransactionFilter, offset, limit int) ([]*entity.Transaction, int64, error)
}

// TransactionFilter 交易流水筛选条件
type TransactionFilter struct {
	Currency    string // 筛选币种
	Type        string // 筛选交易类型
	HideFinance bool   // 隐藏投资收益
}

// DepositRepository 充值记录存储端口
type DepositRepository interface {
	// Create 创建充值记录
	Create(ctx context.Context, deposit *entity.Deposit) error
	// FindByTxHash 根据交易哈希查询
	FindByTxHash(ctx context.Context, txHash string) (*entity.Deposit, error)
	// UpdateStatus 更新充值状态
	UpdateStatus(ctx context.Context, id uint64, status int8) error
	// UpdateConfirmations 更新确认数
	UpdateConfirmations(ctx context.Context, id uint64, confirmations int) error
	// ListPending 获取待确认的充值列表
	ListPending(ctx context.Context) ([]*entity.Deposit, error)
	// ListByUserID 获取用户充值列表 (分页)
	ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Deposit, int64, error)
}

// WithdrawalRepository 提币记录存储端口
type WithdrawalRepository interface {
	// Create 创建提币记录
	Create(ctx context.Context, withdrawal *entity.Withdrawal) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.Withdrawal, error)
	// Update 更新提币记录
	Update(ctx context.Context, withdrawal *entity.Withdrawal) error
	// ListByUserID 获取用户提币列表 (分页)
	ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Withdrawal, int64, error)
	// ListByStatus 获取指定状态的提币列表 (管理后台审核用)
	ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Withdrawal, int64, error)
	// SumTodayByUserID 统计用户今日提币总额
	SumTodayByUserID(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error)
}

// CNYWithdrawalRepository CNY 提现记录存储端口
type CNYWithdrawalRepository interface {
	// Create 创建 CNY 提现记录
	Create(ctx context.Context, withdrawal *entity.CNYWithdrawal) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.CNYWithdrawal, error)
	// Update 更新记录
	Update(ctx context.Context, withdrawal *entity.CNYWithdrawal) error
	// ListByUserID 获取用户提现列表 (分页)
	ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.CNYWithdrawal, int64, error)
	// ListByStatus 获取指定状态的列表 (管理后台审核用)
	ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.CNYWithdrawal, int64, error)
}

// TransferRepository 转账记录存储端口
type TransferRepository interface {
	// Create 创建转账记录
	Create(ctx context.Context, transfer *entity.Transfer) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.Transfer, error)
	// Update 更新转账记录
	Update(ctx context.Context, transfer *entity.Transfer) error
	// ListByUserID 获取用户相关的转账列表 (作为发送方或接收方)
	ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Transfer, int64, error)
}

// ExchangeRepository 闪兑记录存储端口
type ExchangeRepository interface {
	// Create 创建闪兑记录
	Create(ctx context.Context, exchange *entity.Exchange) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.Exchange, error)
	// ListByUserID 获取用户闪兑列表 (分页)
	ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Exchange, int64, error)
}

// ExchangeRateRepository 汇率配置存储端口
type ExchangeRateRepository interface {
	// FindByPair 根据交易对查询汇率
	FindByPair(ctx context.Context, fromCurrency, toCurrency string) (*entity.ExchangeRate, error)
	// FindAll 获取所有汇率配置
	FindAll(ctx context.Context) ([]*entity.ExchangeRate, error)
	// Upsert 创建或更新汇率
	Upsert(ctx context.Context, rate *entity.ExchangeRate) error
}

// RedPacketRepository 红包存储端口
type RedPacketRepository interface {
	// Create 创建红包
	Create(ctx context.Context, packet *entity.RedPacket) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.RedPacket, error)
	// Update 更新红包
	Update(ctx context.Context, packet *entity.RedPacket) error
	// ListByUserID 获取用户创建的红包列表
	ListByUserID(ctx context.Context, userID uint64, status *int8, offset, limit int) ([]*entity.RedPacket, int64, error)
	// ListByUserIDAndStatuses 获取用户指定多个状态的红包列表
	ListByUserIDAndStatuses(ctx context.Context, userID uint64, statuses []int8, offset, limit int) ([]*entity.RedPacket, int64, error)
	// ListExpired 获取已过期未处理的红包
	ListExpired(ctx context.Context) ([]*entity.RedPacket, error)
}

// RedPacketClaimRepository 红包领取记录存储端口
type RedPacketClaimRepository interface {
	// Create 创建领取记录
	Create(ctx context.Context, claim *entity.RedPacketClaim) error
	// FindByPacketAndUser 查询某用户是否已领取
	FindByPacketAndUser(ctx context.Context, packetID, userID uint64) (*entity.RedPacketClaim, error)
	// ListByPacketID 获取红包的所有领取记录
	ListByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error)
}

// RedPacketCoverRepository 红包封面存储端口
type RedPacketCoverRepository interface {
	// Create 创建封面
	Create(ctx context.Context, cover *entity.RedPacketCover) error
	// ListByUserID 获取用户的封面列表
	ListByUserID(ctx context.Context, userID uint64) ([]*entity.RedPacketCover, error)
	// Delete 删除封面
	Delete(ctx context.Context, id uint64) error
}

// UserSettingsRepository 用户配置存储端口
type UserSettingsRepository interface {
	// FindByUserID 根据用户 ID 查询
	FindByUserID(ctx context.Context, userID uint64) (*entity.UserSettings, error)
	// Upsert 创建或更新用户配置
	Upsert(ctx context.Context, settings *entity.UserSettings) error
	// Update 更新配置
	Update(ctx context.Context, settings *entity.UserSettings) error
}

// SubAccountRepository 备用账号存储端口
type SubAccountRepository interface {
	// Create 创建备用账号关联
	Create(ctx context.Context, sub *entity.SubAccount) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.SubAccount, error)
	// Update 更新
	Update(ctx context.Context, sub *entity.SubAccount) error
	// ListByMasterUserID 获取主账号的备用账号列表
	ListByMasterUserID(ctx context.Context, masterUserID uint64) ([]*entity.SubAccount, error)
	// ListBySubUserID 获取备用账号关联的主账号列表
	ListBySubUserID(ctx context.Context, subUserID uint64) ([]*entity.SubAccount, error)
}

// FinanceInvestmentRepository 余额宝投资存储端口
type FinanceInvestmentRepository interface {
	// Create 创建投资记录
	Create(ctx context.Context, investment *entity.FinanceInvestment) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.FinanceInvestment, error)
	// Update 更新
	Update(ctx context.Context, investment *entity.FinanceInvestment) error
	// ListActiveByUserID 获取用户持有中的投资列表
	ListActiveByUserID(ctx context.Context, userID uint64) ([]*entity.FinanceInvestment, error)
	// ListAllActive 获取所有持有中的投资 (每日派息用)
	ListAllActive(ctx context.Context) ([]*entity.FinanceInvestment, error)
}

// SystemConfigRepository 系统配置存储端口
type SystemConfigRepository interface {
	// Get 获取配置值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置配置值
	Set(ctx context.Context, key, value string) error
	// GetAll 获取所有配置
	GetAll(ctx context.Context) ([]*entity.SystemConfig, error)
	// GetMultiple 批量获取配置
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
}

// AdminUserRepository 管理员存储端口
type AdminUserRepository interface {
	// Create 创建管理员
	Create(ctx context.Context, admin *entity.AdminUser) error
	// FindByID 根据 ID 查询
	FindByID(ctx context.Context, id uint64) (*entity.AdminUser, error)
	// FindByUsername 根据用户名查询
	FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error)
	// Update 更新管理员
	Update(ctx context.Context, admin *entity.AdminUser) error
	// List 获取管理员列表
	List(ctx context.Context, offset, limit int) ([]*entity.AdminUser, int64, error)
}

// AdminLogRepository 管理员操作日志存储端口
type AdminLogRepository interface {
	// Create 创建操作日志
	Create(ctx context.Context, log *entity.AdminLog) error
	// List 获取操作日志列表 (分页)
	List(ctx context.Context, filter *AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error)
}

// AdminLogFilter 管理员日志筛选条件
type AdminLogFilter struct {
	AdminID    *uint64
	Action     string
	TargetType string
	StartTime  string
	EndTime    string
}

// MerchantRepository 商户存储端口
type MerchantRepository interface {
	// Create 创建商户
	Create(ctx context.Context, merchant *entity.Merchant) error
	// FindByID 根据 ID 查询商户
	FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	// ListByUserID 获取用户的所有商户列表
	ListByUserID(ctx context.Context, userID uint64) ([]*entity.Merchant, error)
	// CountByUserID 统计用户商户数量
	CountByUserID(ctx context.Context, userID uint64) (int64, error)
	// FindByAPIKey 根据 API Key 查询商户
	FindByAPIKey(ctx context.Context, apiKey string) (*entity.Merchant, error)
	// Update 更新商户
	Update(ctx context.Context, merchant *entity.Merchant) error
	// ListByStatus 获取指定状态的商户列表 (分页)
	ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Merchant, int64, error)
}
