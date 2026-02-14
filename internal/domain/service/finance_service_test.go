package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestFinanceService 创建测试用的 FinanceService 及其 mock 依赖
// 注意: db 传 nil，只测试不依赖事务的方法
func newTestFinanceService() (
	*FinanceService,
	*mocks.MockFinanceInvestmentRepository,
	*mocks.MockWalletRepository,
	*mocks.MockTransactionRepository,
	*mocks.MockSystemConfigRepository,
) {
	financeRepo := new(mocks.MockFinanceInvestmentRepository)
	walletRepo := new(mocks.MockWalletRepository)
	transactionRepo := new(mocks.MockTransactionRepository)
	systemConfigRepo := new(mocks.MockSystemConfigRepository)
	logger := zap.NewNop()

	svc := NewFinanceService(
		nil, // db 为 nil，事务方法不测
		financeRepo,
		walletRepo,
		transactionRepo,
		systemConfigRepo,
		logger,
	)

	return svc, financeRepo, walletRepo, transactionRepo, systemConfigRepo
}

// ==================== NewFinanceService 构造测试 ====================

func TestNewFinanceService(t *testing.T) {
	svc, financeRepo, walletRepo, transactionRepo, systemConfigRepo := newTestFinanceService()

	assert.NotNil(t, svc)
	assert.Equal(t, financeRepo, svc.financeRepo)
	assert.Equal(t, walletRepo, svc.walletRepo)
	assert.Equal(t, transactionRepo, svc.transactionRepo)
	assert.Equal(t, systemConfigRepo, svc.systemConfigRepo)
	assert.NotNil(t, svc.logger)
}

// ==================== GetFinanceConfig 测试 ====================

func TestFinanceService_GetFinanceConfig_所有配置存在(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "50",
		entity.ConfigFinanceMax:        "100000",
		entity.ConfigFinanceAnnualRate: "12.5",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	cfg, err := svc.GetFinanceConfig(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.True(t, cfg.MinInvest.Equal(decimal.NewFromInt(50)))
	assert.True(t, cfg.MaxInvest.Equal(decimal.NewFromInt(100000)))
	assert.True(t, cfg.AnnualRate.Equal(decimal.NewFromFloat(12.5)))
	systemConfigRepo.AssertExpectations(t)
}

func TestFinanceService_GetFinanceConfig_使用默认值(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	// 返回空 map，所有配置都不存在
	configMap := map[string]string{}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	cfg, err := svc.GetFinanceConfig(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// 验证默认值
	assert.True(t, cfg.MinInvest.Equal(decimal.NewFromInt(10)))
	assert.True(t, cfg.MaxInvest.Equal(decimal.NewFromInt(500000)))
	assert.True(t, cfg.AnnualRate.Equal(decimal.NewFromInt(8)))
}

func TestFinanceService_GetFinanceConfig_部分配置存在(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	// 只设置了 min，其他使用默认值
	configMap := map[string]string{
		entity.ConfigFinanceMin: "100",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	cfg, err := svc.GetFinanceConfig(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.True(t, cfg.MinInvest.Equal(decimal.NewFromInt(100)))
	// 其他字段使用默认值
	assert.True(t, cfg.MaxInvest.Equal(decimal.NewFromInt(500000)))
	assert.True(t, cfg.AnnualRate.Equal(decimal.NewFromInt(8)))
}

func TestFinanceService_GetFinanceConfig_非法数值使用默认值(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "abc",      // 非法数值
		entity.ConfigFinanceMax:        "",          // 空字符串
		entity.ConfigFinanceAnnualRate: "not_a_num", // 非法数值
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	cfg, err := svc.GetFinanceConfig(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// 非法数值回退到默认值
	assert.True(t, cfg.MinInvest.Equal(decimal.NewFromInt(10)))
	assert.True(t, cfg.MaxInvest.Equal(decimal.NewFromInt(500000)))
	assert.True(t, cfg.AnnualRate.Equal(decimal.NewFromInt(8)))
}

func TestFinanceService_GetFinanceConfig_查询失败(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(map[string]string{}, errors.New("db error"))

	cfg, err := svc.GetFinanceConfig(ctx)

	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "读取余额宝配置失败")
}

// ==================== Invest 参数校验测试 ====================

func TestFinanceService_Invest_金额低于最低限制(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "10",
		entity.ConfigFinanceMax:        "500000",
		entity.ConfigFinanceAnnualRate: "8",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	result, err := svc.Invest(ctx, 1, decimal.NewFromFloat(5))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "最低投资金额为 10 USDT")
}

func TestFinanceService_Invest_金额超过最高限制(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "10",
		entity.ConfigFinanceMax:        "500000",
		entity.ConfigFinanceAnnualRate: "8",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	result, err := svc.Invest(ctx, 1, decimal.NewFromInt(600000))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "最高投资金额为 500000 USDT")
}

func TestFinanceService_Invest_读取配置失败(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(map[string]string{}, errors.New("db error"))

	result, err := svc.Invest(ctx, 1, decimal.NewFromInt(100))

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "读取余额宝配置失败")
}

func TestFinanceService_Invest_金额等于最低限制_通过校验(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "10",
		entity.ConfigFinanceMax:        "500000",
		entity.ConfigFinanceAnnualRate: "8",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	// 金额 = 10 (等于最低限制)，参数校验应通过
	// 但由于 db 为 nil，事务会 panic，使用 recover 捕获以验证参数校验通过
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "参数校验通过后由于 db 为 nil 应该 panic")
		}()
		_, _ = svc.Invest(ctx, 1, decimal.NewFromInt(10))
	}()
}

func TestFinanceService_Invest_金额等于最高限制_通过校验(t *testing.T) {
	svc, _, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	configMap := map[string]string{
		entity.ConfigFinanceMin:        "10",
		entity.ConfigFinanceMax:        "500000",
		entity.ConfigFinanceAnnualRate: "8",
	}
	systemConfigRepo.On("GetMultiple", ctx, []string{
		entity.ConfigFinanceMin,
		entity.ConfigFinanceMax,
		entity.ConfigFinanceAnnualRate,
	}).Return(configMap, nil)

	// 金额 = 500000 (等于最高限制)，参数校验应通过
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "参数校验通过后由于 db 为 nil 应该 panic")
		}()
		_, _ = svc.Invest(ctx, 1, decimal.NewFromInt(500000))
	}()
}

// ==================== Withdraw 参数校验测试 ====================

func TestFinanceService_Withdraw_投资记录不存在(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.Withdraw(ctx, 999, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestFinanceService_Withdraw_查询投资记录失败(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.Withdraw(ctx, 1, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询投资记录失败")
}

func TestFinanceService_Withdraw_无权操作(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	investment := &entity.FinanceInvestment{
		ID:     1,
		UserID: 100, // 属于用户 100
		Status: entity.FinanceStatusHolding,
	}
	financeRepo.On("FindByID", ctx, uint64(1)).Return(investment, nil)

	// 用户 200 尝试取出
	result, err := svc.Withdraw(ctx, 1, 200)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "无权操作")
}

func TestFinanceService_Withdraw_投资已取出(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	investment := &entity.FinanceInvestment{
		ID:     1,
		UserID: 1,
		Status: entity.FinanceStatusWithdrawn, // 已取出
	}
	financeRepo.On("FindByID", ctx, uint64(1)).Return(investment, nil)

	result, err := svc.Withdraw(ctx, 1, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "该投资已取出")
}

func TestFinanceService_Withdraw_校验通过进入事务(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	investment := &entity.FinanceInvestment{
		ID:     1,
		UserID: 1,
		Status: entity.FinanceStatusHolding,
		Amount: decimal.NewFromInt(1000),
	}
	financeRepo.On("FindByID", ctx, uint64(1)).Return(investment, nil)

	// 参数校验通过后由于 db 为 nil 会 panic
	func() {
		defer func() {
			r := recover()
			assert.NotNil(t, r, "参数校验通过后由于 db 为 nil 应该 panic")
		}()
		_, _ = svc.Withdraw(ctx, 1, 1)
	}()
}

// ==================== GetUserInvestments 测试 ====================

func TestFinanceService_GetUserInvestments_成功(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	expectedList := []*entity.FinanceInvestment{
		{ID: 1, UserID: 1, Amount: decimal.NewFromInt(1000), Status: entity.FinanceStatusHolding},
		{ID: 2, UserID: 1, Amount: decimal.NewFromInt(2000), Status: entity.FinanceStatusHolding},
	}
	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return(expectedList, nil)

	result, err := svc.GetUserInvestments(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedList, result)
	financeRepo.AssertExpectations(t)
}

func TestFinanceService_GetUserInvestments_空列表(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return([]*entity.FinanceInvestment{}, nil)

	result, err := svc.GetUserInvestments(ctx, 1)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestFinanceService_GetUserInvestments_查询失败(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetUserInvestments(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询投资列表失败")
}

// ==================== GetUserInvestmentSummary 测试 ====================

func TestFinanceService_GetUserInvestmentSummary_有投资记录(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	investments := []*entity.FinanceInvestment{
		{ID: 1, UserID: 1, Amount: decimal.NewFromInt(1000), TotalProfit: decimal.NewFromFloat(5.5)},
		{ID: 2, UserID: 1, Amount: decimal.NewFromInt(2000), TotalProfit: decimal.NewFromFloat(12.3)},
		{ID: 3, UserID: 1, Amount: decimal.NewFromFloat(500.5), TotalProfit: decimal.NewFromFloat(2.1)},
	}
	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return(investments, nil)

	summary, err := svc.GetUserInvestmentSummary(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, 3, summary.Count)
	// 总金额: 1000 + 2000 + 500.5 = 3500.5
	assert.True(t, summary.TotalAmount.Equal(decimal.NewFromFloat(3500.5)))
	// 总收益: 5.5 + 12.3 + 2.1 = 19.9
	assert.True(t, summary.TotalProfit.Equal(decimal.NewFromFloat(19.9)))
	financeRepo.AssertExpectations(t)
}

func TestFinanceService_GetUserInvestmentSummary_无投资记录(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return([]*entity.FinanceInvestment{}, nil)

	summary, err := svc.GetUserInvestmentSummary(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, 0, summary.Count)
	assert.True(t, summary.TotalAmount.Equal(decimal.Zero))
	assert.True(t, summary.TotalProfit.Equal(decimal.Zero))
}

func TestFinanceService_GetUserInvestmentSummary_查询失败(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	summary, err := svc.GetUserInvestmentSummary(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "查询投资列表失败")
}

func TestFinanceService_GetUserInvestmentSummary_单笔投资(t *testing.T) {
	svc, financeRepo, _, _, _ := newTestFinanceService()
	ctx := context.Background()

	investments := []*entity.FinanceInvestment{
		{ID: 1, UserID: 1, Amount: decimal.NewFromFloat(888.88), TotalProfit: decimal.NewFromFloat(3.14)},
	}
	financeRepo.On("ListActiveByUserID", ctx, uint64(1)).Return(investments, nil)

	summary, err := svc.GetUserInvestmentSummary(ctx, 1)

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, 1, summary.Count)
	assert.True(t, summary.TotalAmount.Equal(decimal.NewFromFloat(888.88)))
	assert.True(t, summary.TotalProfit.Equal(decimal.NewFromFloat(3.14)))
}

// ==================== DistributeDailyProfits 参数校验/前置逻辑测试 ====================

func TestFinanceService_DistributeDailyProfits_查询活跃投资失败(t *testing.T) {
	svc, financeRepo, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	// 配置读取成功
	systemConfigRepo.On("Get", ctx, entity.ConfigFinanceAnnualRate).Return("8", nil)
	// 投资列表查询失败
	financeRepo.On("ListAllActive", ctx).Return(nil, errors.New("db error"))

	count, err := svc.DistributeDailyProfits(ctx)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "查询活跃投资列表失败")
}

func TestFinanceService_DistributeDailyProfits_无活跃投资(t *testing.T) {
	svc, financeRepo, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	systemConfigRepo.On("Get", ctx, entity.ConfigFinanceAnnualRate).Return("8", nil)
	financeRepo.On("ListAllActive", ctx).Return([]*entity.FinanceInvestment{}, nil)

	count, err := svc.DistributeDailyProfits(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFinanceService_DistributeDailyProfits_利率配置失败使用默认值(t *testing.T) {
	svc, financeRepo, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	// 利率读取失败
	systemConfigRepo.On("Get", ctx, entity.ConfigFinanceAnnualRate).Return("", errors.New("redis error"))
	// 无投资记录
	financeRepo.On("ListAllActive", ctx).Return([]*entity.FinanceInvestment{}, nil)

	count, err := svc.DistributeDailyProfits(ctx)

	// 即使利率配置失败，也不应报错（使用默认值 8%）
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestFinanceService_DistributeDailyProfits_利率配置非法使用默认值(t *testing.T) {
	svc, financeRepo, _, _, systemConfigRepo := newTestFinanceService()
	ctx := context.Background()

	// 利率值非法
	systemConfigRepo.On("Get", ctx, entity.ConfigFinanceAnnualRate).Return("not_a_number", nil)
	// 无投资记录
	financeRepo.On("ListAllActive", ctx).Return([]*entity.FinanceInvestment{}, nil)

	count, err := svc.DistributeDailyProfits(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}
