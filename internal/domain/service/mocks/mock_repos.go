// Package mocks 提供所有 Repository 端口接口的 mock 实现，供单元测试使用
package mocks

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
)

// --- MockUserRepository ---

// MockUserRepository 用户存储 mock
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uint64) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error) {
	args := m.Called(ctx, telegramID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) CountAll(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// --- MockWalletRepository ---

// MockWalletRepository 钱包存储 mock
type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) Create(ctx context.Context, wallet *entity.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) FindByID(ctx context.Context, id uint64) (*entity.Wallet, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Wallet), args.Error(1)
}

func (m *MockWalletRepository) FindByUserIDAndCurrency(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
	args := m.Called(ctx, userID, currency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Wallet), args.Error(1)
}

func (m *MockWalletRepository) FindByUserID(ctx context.Context, userID uint64) ([]*entity.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Wallet), args.Error(1)
}

func (m *MockWalletRepository) FindByDepositAddress(ctx context.Context, address string) (*entity.Wallet, error) {
	args := m.Called(ctx, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Wallet), args.Error(1)
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	args := m.Called(ctx, walletID, amount)
	return args.Error(0)
}

func (m *MockWalletRepository) FreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	args := m.Called(ctx, walletID, amount)
	return args.Error(0)
}

func (m *MockWalletRepository) UnfreezeBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	args := m.Called(ctx, walletID, amount)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateDepositAddress(ctx context.Context, userID uint64, address string) error {
	args := m.Called(ctx, userID, address)
	return args.Error(0)
}

func (m *MockWalletRepository) ListAllWithDepositAddress(ctx context.Context) ([]*entity.Wallet, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Wallet), args.Error(1)
}

func (m *MockWalletRepository) ListDepositAddressesCreatedAfter(ctx context.Context, after time.Time) ([]*entity.Wallet, error) {
	args := m.Called(ctx, after)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Wallet), args.Error(1)
}

// --- MockWalletKeyRepository ---

// MockWalletKeyRepository 钱包私钥存储 mock
type MockWalletKeyRepository struct {
	mock.Mock
}

func (m *MockWalletKeyRepository) Create(ctx context.Context, key *entity.WalletKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockWalletKeyRepository) FindByUserID(ctx context.Context, userID uint64) (*entity.WalletKey, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WalletKey), args.Error(1)
}

// --- MockUserSettingsRepository ---

// MockUserSettingsRepository 用户配置存储 mock
type MockUserSettingsRepository struct {
	mock.Mock
}

func (m *MockUserSettingsRepository) FindByUserID(ctx context.Context, userID uint64) (*entity.UserSettings, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UserSettings), args.Error(1)
}

func (m *MockUserSettingsRepository) Upsert(ctx context.Context, settings *entity.UserSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func (m *MockUserSettingsRepository) Update(ctx context.Context, settings *entity.UserSettings) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

// --- MockSystemConfigRepository ---

// MockSystemConfigRepository 系统配置存储 mock
type MockSystemConfigRepository struct {
	mock.Mock
}

func (m *MockSystemConfigRepository) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockSystemConfigRepository) Set(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSystemConfigRepository) GetAll(ctx context.Context) ([]*entity.SystemConfig, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.SystemConfig), args.Error(1)
}

func (m *MockSystemConfigRepository) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(map[string]string), args.Error(1)
}

// --- MockDepositRepository ---

// MockDepositRepository 充值记录存储 mock
type MockDepositRepository struct {
	mock.Mock
}

func (m *MockDepositRepository) Create(ctx context.Context, deposit *entity.Deposit) error {
	args := m.Called(ctx, deposit)
	return args.Error(0)
}

func (m *MockDepositRepository) FindByTxHash(ctx context.Context, txHash string) (*entity.Deposit, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Deposit), args.Error(1)
}

func (m *MockDepositRepository) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockDepositRepository) UpdateConfirmations(ctx context.Context, id uint64, confirmations int) error {
	args := m.Called(ctx, id, confirmations)
	return args.Error(0)
}

func (m *MockDepositRepository) ListPending(ctx context.Context) ([]*entity.Deposit, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Deposit), args.Error(1)
}

func (m *MockDepositRepository) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Deposit, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*entity.Deposit), args.Get(1).(int64), args.Error(2)
}

// --- MockTransactionRepository ---

// MockTransactionRepository 交易流水存储 mock
type MockTransactionRepository struct {
	mock.Mock
}

func (m *MockTransactionRepository) Create(ctx context.Context, tx *entity.Transaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id uint64) (*entity.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Transaction), args.Error(1)
}

func (m *MockTransactionRepository) ListByUserID(ctx context.Context, userID uint64, filter *port.TransactionFilter, offset, limit int) ([]*entity.Transaction, int64, error) {
	args := m.Called(ctx, userID, filter, offset, limit)
	return args.Get(0).([]*entity.Transaction), args.Get(1).(int64), args.Error(2)
}

// --- MockWithdrawalRepository ---

// MockWithdrawalRepository 提币记录存储 mock
type MockWithdrawalRepository struct {
	mock.Mock
}

func (m *MockWithdrawalRepository) Create(ctx context.Context, withdrawal *entity.Withdrawal) error {
	args := m.Called(ctx, withdrawal)
	return args.Error(0)
}

func (m *MockWithdrawalRepository) FindByID(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Withdrawal), args.Error(1)
}

func (m *MockWithdrawalRepository) Update(ctx context.Context, withdrawal *entity.Withdrawal) error {
	args := m.Called(ctx, withdrawal)
	return args.Error(0)
}

func (m *MockWithdrawalRepository) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Withdrawal, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*entity.Withdrawal), args.Get(1).(int64), args.Error(2)
}

func (m *MockWithdrawalRepository) ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Withdrawal, int64, error) {
	args := m.Called(ctx, status, offset, limit)
	return args.Get(0).([]*entity.Withdrawal), args.Get(1).(int64), args.Error(2)
}

func (m *MockWithdrawalRepository) SumTodayByUserID(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	args := m.Called(ctx, userID, currency)
	return args.Get(0).(decimal.Decimal), args.Error(1)
}

// --- MockTransferRepository ---

// MockTransferRepository 转账记录存储 mock
type MockTransferRepository struct {
	mock.Mock
}

func (m *MockTransferRepository) Create(ctx context.Context, transfer *entity.Transfer) error {
	args := m.Called(ctx, transfer)
	return args.Error(0)
}

func (m *MockTransferRepository) FindByID(ctx context.Context, id uint64) (*entity.Transfer, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Transfer), args.Error(1)
}

func (m *MockTransferRepository) Update(ctx context.Context, transfer *entity.Transfer) error {
	args := m.Called(ctx, transfer)
	return args.Error(0)
}

func (m *MockTransferRepository) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Transfer, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*entity.Transfer), args.Get(1).(int64), args.Error(2)
}

// --- MockExchangeRateRepository ---

// MockExchangeRateRepository 汇率配置存储 mock
type MockExchangeRateRepository struct {
	mock.Mock
}

func (m *MockExchangeRateRepository) FindByPair(ctx context.Context, fromCurrency, toCurrency string) (*entity.ExchangeRate, error) {
	args := m.Called(ctx, fromCurrency, toCurrency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.ExchangeRate), args.Error(1)
}

func (m *MockExchangeRateRepository) FindAll(ctx context.Context) ([]*entity.ExchangeRate, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.ExchangeRate), args.Error(1)
}

func (m *MockExchangeRateRepository) Upsert(ctx context.Context, rate *entity.ExchangeRate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

// --- MockExchangeRepository ---

// MockExchangeRepository 闪兑记录存储 mock
type MockExchangeRepository struct {
	mock.Mock
}

func (m *MockExchangeRepository) Create(ctx context.Context, exchange *entity.Exchange) error {
	args := m.Called(ctx, exchange)
	return args.Error(0)
}

func (m *MockExchangeRepository) FindByID(ctx context.Context, id uint64) (*entity.Exchange, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Exchange), args.Error(1)
}

func (m *MockExchangeRepository) ListByUserID(ctx context.Context, userID uint64, offset, limit int) ([]*entity.Exchange, int64, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*entity.Exchange), args.Get(1).(int64), args.Error(2)
}

// --- MockCacheRepository ---

// MockCacheRepository 缓存存储 mock
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) Incr(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheRepository) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	args := m.Called(ctx, key, expiration)
	return args.Error(0)
}

func (m *MockCacheRepository) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, expiration)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) HSet(ctx context.Context, key string, values ...interface{}) error {
	args := m.Called(ctx, key, values)
	return args.Error(0)
}

func (m *MockCacheRepository) HGet(ctx context.Context, key, field string) (string, error) {
	args := m.Called(ctx, key, field)
	return args.String(0), args.Error(1)
}

func (m *MockCacheRepository) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(map[string]string), args.Error(1)
}

// --- MockAdminUserRepository ---

// MockAdminUserRepository 管理员存储 mock
type MockAdminUserRepository struct {
	mock.Mock
}

func (m *MockAdminUserRepository) Create(ctx context.Context, admin *entity.AdminUser) error {
	args := m.Called(ctx, admin)
	return args.Error(0)
}

func (m *MockAdminUserRepository) FindByID(ctx context.Context, id uint64) (*entity.AdminUser, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AdminUser), args.Error(1)
}

func (m *MockAdminUserRepository) FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.AdminUser), args.Error(1)
}

func (m *MockAdminUserRepository) Update(ctx context.Context, admin *entity.AdminUser) error {
	args := m.Called(ctx, admin)
	return args.Error(0)
}

func (m *MockAdminUserRepository) List(ctx context.Context, offset, limit int) ([]*entity.AdminUser, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*entity.AdminUser), args.Get(1).(int64), args.Error(2)
}

// --- MockAdminLogRepository ---

// MockAdminLogRepository 管理员操作日志存储 mock
type MockAdminLogRepository struct {
	mock.Mock
}

func (m *MockAdminLogRepository) Create(ctx context.Context, log *entity.AdminLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockAdminLogRepository) List(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
	args := m.Called(ctx, filter, offset, limit)
	return args.Get(0).([]*entity.AdminLog), args.Get(1).(int64), args.Error(2)
}

// --- MockRedPacketRepository ---

// MockRedPacketRepository 红包存储 mock
type MockRedPacketRepository struct {
	mock.Mock
}

func (m *MockRedPacketRepository) Create(ctx context.Context, packet *entity.RedPacket) error {
	args := m.Called(ctx, packet)
	return args.Error(0)
}

func (m *MockRedPacketRepository) FindByID(ctx context.Context, id uint64) (*entity.RedPacket, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RedPacket), args.Error(1)
}

func (m *MockRedPacketRepository) Update(ctx context.Context, packet *entity.RedPacket) error {
	args := m.Called(ctx, packet)
	return args.Error(0)
}

func (m *MockRedPacketRepository) ListByUserID(ctx context.Context, userID uint64, status *int8, offset, limit int) ([]*entity.RedPacket, int64, error) {
	args := m.Called(ctx, userID, status, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*entity.RedPacket), args.Get(1).(int64), args.Error(2)
}

func (m *MockRedPacketRepository) ListByUserIDAndStatuses(ctx context.Context, userID uint64, statuses []int8, offset, limit int) ([]*entity.RedPacket, int64, error) {
	args := m.Called(ctx, userID, statuses, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*entity.RedPacket), args.Get(1).(int64), args.Error(2)
}

func (m *MockRedPacketRepository) ListExpired(ctx context.Context) ([]*entity.RedPacket, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.RedPacket), args.Error(1)
}

// --- MockRedPacketClaimRepository ---

// MockRedPacketClaimRepository 红包领取记录存储 mock
type MockRedPacketClaimRepository struct {
	mock.Mock
}

func (m *MockRedPacketClaimRepository) Create(ctx context.Context, claim *entity.RedPacketClaim) error {
	args := m.Called(ctx, claim)
	return args.Error(0)
}

func (m *MockRedPacketClaimRepository) FindByPacketAndUser(ctx context.Context, packetID, userID uint64) (*entity.RedPacketClaim, error) {
	args := m.Called(ctx, packetID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RedPacketClaim), args.Error(1)
}

func (m *MockRedPacketClaimRepository) ListByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
	args := m.Called(ctx, packetID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.RedPacketClaim), args.Error(1)
}

// --- MockRedPacketCoverRepository ---

// MockRedPacketCoverRepository 红包封面存储 mock
type MockRedPacketCoverRepository struct {
	mock.Mock
}

func (m *MockRedPacketCoverRepository) Create(ctx context.Context, cover *entity.RedPacketCover) error {
	args := m.Called(ctx, cover)
	return args.Error(0)
}

func (m *MockRedPacketCoverRepository) ListByUserID(ctx context.Context, userID uint64) ([]*entity.RedPacketCover, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.RedPacketCover), args.Error(1)
}

func (m *MockRedPacketCoverRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// --- MockFinanceInvestmentRepository ---

// MockFinanceInvestmentRepository 余额宝投资存储 mock
type MockFinanceInvestmentRepository struct {
	mock.Mock
}

func (m *MockFinanceInvestmentRepository) Create(ctx context.Context, investment *entity.FinanceInvestment) error {
	args := m.Called(ctx, investment)
	return args.Error(0)
}

func (m *MockFinanceInvestmentRepository) FindByID(ctx context.Context, id uint64) (*entity.FinanceInvestment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.FinanceInvestment), args.Error(1)
}

func (m *MockFinanceInvestmentRepository) Update(ctx context.Context, investment *entity.FinanceInvestment) error {
	args := m.Called(ctx, investment)
	return args.Error(0)
}

func (m *MockFinanceInvestmentRepository) ListActiveByUserID(ctx context.Context, userID uint64) ([]*entity.FinanceInvestment, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.FinanceInvestment), args.Error(1)
}

func (m *MockFinanceInvestmentRepository) ListAllActive(ctx context.Context) ([]*entity.FinanceInvestment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.FinanceInvestment), args.Error(1)
}

// --- MockMerchantRepository ---

// MockMerchantRepository 商户存储 mock
type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) FindByID(ctx context.Context, id uint64) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) ListByUserID(ctx context.Context, userID uint64) ([]*entity.Merchant, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) CountByUserID(ctx context.Context, userID uint64) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMerchantRepository) FindByAPIKey(ctx context.Context, apiKey string) (*entity.Merchant, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) Update(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) ListByStatus(ctx context.Context, status int8, offset, limit int) ([]*entity.Merchant, int64, error) {
	args := m.Called(ctx, status, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*entity.Merchant), args.Get(1).(int64), args.Error(2)
}
