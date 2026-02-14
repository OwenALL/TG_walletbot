package handler

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
	apperrors "github.com/TGlimmer/TG_walletbot/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// --- mockAdminUseCase: 集中管理所有 mock 行为 ---
// 设计说明:
//   由于 AdminUseCase 是具体结构体而非接口，无法直接 mock 其方法。
//   因此通过控制底层 Repository 的返回值来间接控制 AdminUseCase 的行为。
//   对于 AdminService 层 (Authenticate/GetAdminByID) 的方法，通过
//   mockAdminUserRepo 的 findByUsernameFunc / findByIDFunc 来控制。
//   对于 AdminUseCase 中直接使用 uc.db 的方法 (如 GetDashboardStats)，
//   因需要真实 DB 连接，在单元测试中无法直接模拟。

type mockAdminUseCase struct {
	// AdminUserRepo mock
	loginFunc        func(ctx context.Context, username, password string) (*entity.AdminUser, error)
	getAdminByIDFunc func(ctx context.Context, id uint64) (*entity.AdminUser, error)

	// AdminUserRepo.Update mock (用于 ChangePassword 等需要更新管理员的场景)
	updateAdminFunc func(ctx context.Context, admin *entity.AdminUser) error

	// AdminLogRepo mock
	listAuditLogsFunc func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error)

	// ExchangeRateRepo mock
	getAllExchangeRatesFunc func(ctx context.Context) ([]*entity.ExchangeRate, error)
	updateExchangeRateFunc func(ctx context.Context, rate *entity.ExchangeRate) error

	// SystemConfigRepo mock
	getAllConfigsFunc func(ctx context.Context) ([]*entity.SystemConfig, error)
	updateConfigFunc  func(ctx context.Context, key, value string) error

	// UserRepo mock
	findUserByIDFunc func(ctx context.Context, id uint64) (*entity.User, error)
	updateUserFunc   func(ctx context.Context, user *entity.User) error

	// WalletRepo mock
	findWalletsByUserIDFunc         func(ctx context.Context, userID uint64) ([]*entity.Wallet, error)
	findWalletByUserIDCurrencyFunc  func(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error)
	updateWalletBalanceFunc         func(ctx context.Context, walletID uint64, amount decimal.Decimal) error

	// WithdrawalRepo mock
	findWithdrawalByIDFunc func(ctx context.Context, id uint64) (*entity.Withdrawal, error)
	updateWithdrawalFunc   func(ctx context.Context, withdrawal *entity.Withdrawal) error
}

// newMockAdminUseCase 创建默认空实现的 mock
func newMockAdminUseCase() *mockAdminUseCase {
	return &mockAdminUseCase{}
}

// toAdminUseCase 构建真实 *app.AdminUseCase，底层 repo 委托给 mock 函数
func (m *mockAdminUseCase) toAdminUseCase() *app.AdminUseCase {
	adminUserRepo := &mockAdminUserRepo{
		findByIDFunc:       m.getAdminByIDFunc,
		findByUsernameFunc: m.buildFindByUsernameFunc(),
		updateFunc:         m.updateAdminFunc,
	}
	adminLogRepo := &mockAdminLogRepo{
		listFunc: m.listAuditLogsFunc,
	}
	adminSvc := service.NewAdminService(adminUserRepo, adminLogRepo, zap.NewNop())

	return app.NewAdminUseCase(
		adminSvc,
		&mockUserRepo{findByIDFunc: m.findUserByIDFunc, updateFunc: m.updateUserFunc},
		&mockWalletRepo{
			findByUserIDFunc:            m.findWalletsByUserIDFunc,
			findByUserIDAndCurrencyFunc: m.findWalletByUserIDCurrencyFunc,
			updateBalanceFunc:           m.updateWalletBalanceFunc,
		},
		&mockTransactionRepo{},
		&mockDepositRepo{},
		&mockWithdrawalRepo{findByIDFunc: m.findWithdrawalByIDFunc, updateFunc: m.updateWithdrawalFunc},
		&mockExchangeRateRepo{findAllFunc: m.getAllExchangeRatesFunc, upsertFunc: m.updateExchangeRateFunc},
		&mockSystemConfigRepo{getAllFunc: m.getAllConfigsFunc, setFunc: m.updateConfigFunc},
		&gorm.DB{},
		zap.NewNop(),
	)
}

// buildFindByUsernameFunc 根据 loginFunc 构建 FindByUsername 的委托行为
// AdminService.Authenticate 内部流程:
//   1. FindByUsername(username) -> admin, err
//   2. if err != nil -> 包装为 500 错误返回
//   3. if admin == nil -> 返回 401 "用户名或密码错误"
//   4. bcrypt.Compare(admin.PasswordHash, password)
//   5. if 密码不匹配 -> 返回 401 "用户名或密码错误"
//
// 因此 mock 策略:
//   - loginFunc 返回成功 (admin, nil): FindByUsername 返回带已知密码哈希的 admin
//   - loginFunc 返回 AppError (401/403等): FindByUsername 返回 nil (触发 Authenticate 自然返回 401)
//   - loginFunc 返回普通 error: FindByUsername 返回该 error (触发 Authenticate 包装为 500)
func (m *mockAdminUseCase) buildFindByUsernameFunc() func(ctx context.Context, username string) (*entity.AdminUser, error) {
	if m.loginFunc == nil {
		return nil
	}
	return func(ctx context.Context, username string) (*entity.AdminUser, error) {
		result, err := m.loginFunc(ctx, username, "")
		if err != nil {
			// 判断是否为 AppError (业务错误)
			if _, ok := err.(*apperrors.AppError); ok {
				// 业务错误 (如 Unauthorized): 返回 nil 让 Authenticate 自然生成对应错误
				return nil, nil
			}
			// 非业务错误 (如 assert.AnError): 直接传递给 Authenticate 的错误处理
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		// 设置已知密码哈希，让 bcrypt 验证通过
		// 对应密码 "password123" (测试中统一使用)
		result.PasswordHash = knownPasswordHash
		return result, nil
	}
}

// knownPasswordHash 预生成的 bcrypt 哈希值 (对应密码 "password123")
var knownPasswordHash string

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		panic("生成测试密码哈希失败: " + err.Error())
	}
	knownPasswordHash = string(hash)
}

// --- mock Repository 实现 ---

// mockAdminUserRepo 模拟 AdminUserRepository
type mockAdminUserRepo struct {
	findByIDFunc       func(ctx context.Context, id uint64) (*entity.AdminUser, error)
	findByUsernameFunc func(ctx context.Context, username string) (*entity.AdminUser, error)
	updateFunc         func(ctx context.Context, admin *entity.AdminUser) error
}

func (r *mockAdminUserRepo) Create(_ context.Context, _ *entity.AdminUser) error { return nil }
func (r *mockAdminUserRepo) FindByID(ctx context.Context, id uint64) (*entity.AdminUser, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (r *mockAdminUserRepo) FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error) {
	if r.findByUsernameFunc != nil {
		return r.findByUsernameFunc(ctx, username)
	}
	return nil, nil
}
func (r *mockAdminUserRepo) Update(ctx context.Context, admin *entity.AdminUser) error {
	if r.updateFunc != nil {
		return r.updateFunc(ctx, admin)
	}
	return nil
}
func (r *mockAdminUserRepo) List(_ context.Context, _, _ int) ([]*entity.AdminUser, int64, error) {
	return nil, 0, nil
}

// mockAdminLogRepo 模拟 AdminLogRepository
type mockAdminLogRepo struct {
	listFunc func(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error)
}

func (r *mockAdminLogRepo) Create(_ context.Context, _ *entity.AdminLog) error { return nil }
func (r *mockAdminLogRepo) List(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
	if r.listFunc != nil {
		return r.listFunc(ctx, filter, offset, limit)
	}
	return nil, 0, nil
}

// mockUserRepo 模拟 UserRepository
type mockUserRepo struct {
	findByIDFunc func(ctx context.Context, id uint64) (*entity.User, error)
	updateFunc   func(ctx context.Context, user *entity.User) error
}

func (r *mockUserRepo) Create(_ context.Context, _ *entity.User) error { return nil }
func (r *mockUserRepo) FindByID(ctx context.Context, id uint64) (*entity.User, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (r *mockUserRepo) FindByTelegramID(_ context.Context, _ int64) (*entity.User, error) {
	return nil, nil
}
func (r *mockUserRepo) FindByUsername(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (r *mockUserRepo) Update(ctx context.Context, user *entity.User) error {
	if r.updateFunc != nil {
		return r.updateFunc(ctx, user)
	}
	return nil
}
func (r *mockUserRepo) List(_ context.Context, _, _ int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (r *mockUserRepo) CountAll(_ context.Context) (int64, error) { return 0, nil }

// mockWalletRepo 模拟 WalletRepository
type mockWalletRepo struct {
	findByUserIDFunc            func(ctx context.Context, userID uint64) ([]*entity.Wallet, error)
	findByUserIDAndCurrencyFunc func(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error)
	updateBalanceFunc           func(ctx context.Context, walletID uint64, amount decimal.Decimal) error
}

func (r *mockWalletRepo) Create(_ context.Context, _ *entity.Wallet) error { return nil }
func (r *mockWalletRepo) FindByID(_ context.Context, _ uint64) (*entity.Wallet, error) {
	return nil, nil
}
func (r *mockWalletRepo) FindByUserIDAndCurrency(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
	if r.findByUserIDAndCurrencyFunc != nil {
		return r.findByUserIDAndCurrencyFunc(ctx, userID, currency)
	}
	return nil, nil
}
func (r *mockWalletRepo) FindByUserID(ctx context.Context, userID uint64) ([]*entity.Wallet, error) {
	if r.findByUserIDFunc != nil {
		return r.findByUserIDFunc(ctx, userID)
	}
	return nil, nil
}
func (r *mockWalletRepo) FindByDepositAddress(_ context.Context, _ string) (*entity.Wallet, error) {
	return nil, nil
}
func (r *mockWalletRepo) UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	if r.updateBalanceFunc != nil {
		return r.updateBalanceFunc(ctx, walletID, amount)
	}
	return nil
}
func (r *mockWalletRepo) FreezeBalance(_ context.Context, _ uint64, _ decimal.Decimal) error {
	return nil
}
func (r *mockWalletRepo) UnfreezeBalance(_ context.Context, _ uint64, _ decimal.Decimal) error {
	return nil
}
func (r *mockWalletRepo) UpdateDepositAddress(_ context.Context, _ uint64, _ string) error {
	return nil
}
func (r *mockWalletRepo) ListAllWithDepositAddress(_ context.Context) ([]*entity.Wallet, error) {
	return nil, nil
}
func (r *mockWalletRepo) ListDepositAddressesCreatedAfter(_ context.Context, _ time.Time) ([]*entity.Wallet, error) {
	return nil, nil
}

// mockTransactionRepo 模拟 TransactionRepository
type mockTransactionRepo struct{}

func (r *mockTransactionRepo) Create(_ context.Context, _ *entity.Transaction) error { return nil }
func (r *mockTransactionRepo) FindByID(_ context.Context, _ uint64) (*entity.Transaction, error) {
	return nil, nil
}
func (r *mockTransactionRepo) ListByUserID(_ context.Context, _ uint64, _ *port.TransactionFilter, _, _ int) ([]*entity.Transaction, int64, error) {
	return nil, 0, nil
}

// mockDepositRepo 模拟 DepositRepository
type mockDepositRepo struct{}

func (r *mockDepositRepo) Create(_ context.Context, _ *entity.Deposit) error { return nil }
func (r *mockDepositRepo) FindByTxHash(_ context.Context, _ string) (*entity.Deposit, error) {
	return nil, nil
}
func (r *mockDepositRepo) UpdateStatus(_ context.Context, _ uint64, _ int8) error { return nil }
func (r *mockDepositRepo) UpdateConfirmations(_ context.Context, _ uint64, _ int) error { return nil }
func (r *mockDepositRepo) ListPending(_ context.Context) ([]*entity.Deposit, error) {
	return nil, nil
}
func (r *mockDepositRepo) ListByUserID(_ context.Context, _ uint64, _, _ int) ([]*entity.Deposit, int64, error) {
	return nil, 0, nil
}

// mockWithdrawalRepo 模拟 WithdrawalRepository
type mockWithdrawalRepo struct {
	findByIDFunc func(ctx context.Context, id uint64) (*entity.Withdrawal, error)
	updateFunc   func(ctx context.Context, withdrawal *entity.Withdrawal) error
}

func (r *mockWithdrawalRepo) Create(_ context.Context, _ *entity.Withdrawal) error { return nil }
func (r *mockWithdrawalRepo) FindByID(ctx context.Context, id uint64) (*entity.Withdrawal, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (r *mockWithdrawalRepo) Update(ctx context.Context, withdrawal *entity.Withdrawal) error {
	if r.updateFunc != nil {
		return r.updateFunc(ctx, withdrawal)
	}
	return nil
}
func (r *mockWithdrawalRepo) ListByUserID(_ context.Context, _ uint64, _, _ int) ([]*entity.Withdrawal, int64, error) {
	return nil, 0, nil
}
func (r *mockWithdrawalRepo) ListByStatus(_ context.Context, _ int8, _, _ int) ([]*entity.Withdrawal, int64, error) {
	return nil, 0, nil
}
func (r *mockWithdrawalRepo) SumTodayByUserID(_ context.Context, _ uint64, _ string) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// mockExchangeRateRepo 模拟 ExchangeRateRepository
type mockExchangeRateRepo struct {
	findAllFunc func(ctx context.Context) ([]*entity.ExchangeRate, error)
	upsertFunc  func(ctx context.Context, rate *entity.ExchangeRate) error
}

func (r *mockExchangeRateRepo) FindByPair(_ context.Context, _, _ string) (*entity.ExchangeRate, error) {
	return nil, nil
}
func (r *mockExchangeRateRepo) FindAll(ctx context.Context) ([]*entity.ExchangeRate, error) {
	if r.findAllFunc != nil {
		return r.findAllFunc(ctx)
	}
	return nil, nil
}
func (r *mockExchangeRateRepo) Upsert(ctx context.Context, rate *entity.ExchangeRate) error {
	if r.upsertFunc != nil {
		return r.upsertFunc(ctx, rate)
	}
	return nil
}

// mockSystemConfigRepo 模拟 SystemConfigRepository
type mockSystemConfigRepo struct {
	getFunc    func(ctx context.Context, key string) (string, error)
	setFunc    func(ctx context.Context, key, value string) error
	getAllFunc func(ctx context.Context) ([]*entity.SystemConfig, error)
}

func (r *mockSystemConfigRepo) Get(ctx context.Context, key string) (string, error) {
	if r.getFunc != nil {
		return r.getFunc(ctx, key)
	}
	return "", nil
}
func (r *mockSystemConfigRepo) Set(ctx context.Context, key, value string) error {
	if r.setFunc != nil {
		return r.setFunc(ctx, key, value)
	}
	return nil
}
func (r *mockSystemConfigRepo) GetAll(ctx context.Context) ([]*entity.SystemConfig, error) {
	if r.getAllFunc != nil {
		return r.getAllFunc(ctx)
	}
	return nil, nil
}
func (r *mockSystemConfigRepo) GetMultiple(_ context.Context, _ []string) (map[string]string, error) {
	return nil, nil
}

// --- 直接注入型 Handler 的独立 mock ---

// mockRedPacketRepo 模拟 RedPacketRepository
type mockRedPacketRepo struct {
	findByIDFunc func(ctx context.Context, id uint64) (*entity.RedPacket, error)
}

func (r *mockRedPacketRepo) Create(_ context.Context, _ *entity.RedPacket) error { return nil }
func (r *mockRedPacketRepo) FindByID(ctx context.Context, id uint64) (*entity.RedPacket, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (r *mockRedPacketRepo) Update(_ context.Context, _ *entity.RedPacket) error { return nil }
func (r *mockRedPacketRepo) ListByUserID(_ context.Context, _ uint64, _ *int8, _, _ int) ([]*entity.RedPacket, int64, error) {
	return nil, 0, nil
}
func (r *mockRedPacketRepo) ListByUserIDAndStatuses(_ context.Context, _ uint64, _ []int8, _, _ int) ([]*entity.RedPacket, int64, error) {
	return nil, 0, nil
}
func (r *mockRedPacketRepo) ListExpired(_ context.Context) ([]*entity.RedPacket, error) {
	return nil, nil
}

// mockRedPacketClaimRepo 模拟 RedPacketClaimRepository
type mockRedPacketClaimRepo struct {
	listByPacketIDFunc func(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error)
}

func (r *mockRedPacketClaimRepo) Create(_ context.Context, _ *entity.RedPacketClaim) error {
	return nil
}
func (r *mockRedPacketClaimRepo) FindByPacketAndUser(_ context.Context, _, _ uint64) (*entity.RedPacketClaim, error) {
	return nil, nil
}
func (r *mockRedPacketClaimRepo) ListByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
	if r.listByPacketIDFunc != nil {
		return r.listByPacketIDFunc(ctx, packetID)
	}
	return nil, nil
}

// mockMerchantRepo 模拟 MerchantRepository
type mockMerchantRepo struct {
	findByIDFunc func(ctx context.Context, id uint64) (*entity.Merchant, error)
	updateFunc   func(ctx context.Context, merchant *entity.Merchant) error
}

func (r *mockMerchantRepo) Create(_ context.Context, _ *entity.Merchant) error { return nil }
func (r *mockMerchantRepo) FindByID(ctx context.Context, id uint64) (*entity.Merchant, error) {
	if r.findByIDFunc != nil {
		return r.findByIDFunc(ctx, id)
	}
	return nil, nil
}
func (r *mockMerchantRepo) ListByUserID(_ context.Context, _ uint64) ([]*entity.Merchant, error) {
	return nil, nil
}
func (r *mockMerchantRepo) CountByUserID(_ context.Context, _ uint64) (int64, error) {
	return 0, nil
}
func (r *mockMerchantRepo) FindByAPIKey(_ context.Context, _ string) (*entity.Merchant, error) {
	return nil, nil
}
func (r *mockMerchantRepo) Update(ctx context.Context, merchant *entity.Merchant) error {
	if r.updateFunc != nil {
		return r.updateFunc(ctx, merchant)
	}
	return nil
}
func (r *mockMerchantRepo) ListByStatus(_ context.Context, _ int8, _, _ int) ([]*entity.Merchant, int64, error) {
	return nil, 0, nil
}

// mockFinanceInvestmentRepo 模拟 FinanceInvestmentRepository
type mockFinanceInvestmentRepo struct{}

func (r *mockFinanceInvestmentRepo) Create(_ context.Context, _ *entity.FinanceInvestment) error {
	return nil
}
func (r *mockFinanceInvestmentRepo) FindByID(_ context.Context, _ uint64) (*entity.FinanceInvestment, error) {
	return nil, nil
}
func (r *mockFinanceInvestmentRepo) Update(_ context.Context, _ *entity.FinanceInvestment) error {
	return nil
}
func (r *mockFinanceInvestmentRepo) ListActiveByUserID(_ context.Context, _ uint64) ([]*entity.FinanceInvestment, error) {
	return nil, nil
}
func (r *mockFinanceInvestmentRepo) ListAllActive(_ context.Context) ([]*entity.FinanceInvestment, error) {
	return nil, nil
}
