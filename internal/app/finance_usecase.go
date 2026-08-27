package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
)

// FinanceDashboard 余额宝面板展示数据
type FinanceDashboard struct {
	AnnualRate      decimal.Decimal            // 年化利率 (百分比)
	MinInvest       decimal.Decimal            // 最低投资金额
	MaxInvest       decimal.Decimal            // 最高投资金额
	Investments     []*entity.FinanceInvestment // 持有中的投资列表
	TotalAmount     decimal.Decimal            // 总持仓
	TotalProfit     decimal.Decimal            // 累计收益
	YesterdayProfit decimal.Decimal            // 昨日收益 (基于当前持仓计算的预估日收益)
	USDTBalance     decimal.Decimal            // 当前 USDT 余额
}

// FinanceUseCase 余额宝用例层
type FinanceUseCase struct {
	financeSvc *service.FinanceService
	walletSvc  *service.WalletService
	logger     *zap.Logger
}

// NewFinanceUseCase 创建余额宝用例
func NewFinanceUseCase(
	financeSvc *service.FinanceService,
	walletSvc *service.WalletService,
	logger *zap.Logger,
) *FinanceUseCase {
	return &FinanceUseCase{
		financeSvc: financeSvc,
		walletSvc:  walletSvc,
		logger:     logger,
	}
}

// GetDashboardData 获取余额宝面板数据
// 聚合配置、持仓列表、汇总信息和 USDT 余额
func (uc *FinanceUseCase) GetDashboardData(ctx context.Context, userID uint64) (*FinanceDashboard, error) {
	// 并行获取配置、投资列表、USDT 余额
	type configResult struct {
		cfg *service.FinanceConfig
		err error
	}
	type investResult struct {
		investments []*entity.FinanceInvestment
		err         error
	}
	type balanceResult struct {
		balance decimal.Decimal
		err     error
	}

	cfgCh := make(chan configResult, 1)
	invCh := make(chan investResult, 1)
	balCh := make(chan balanceResult, 1)

	go func() {
		cfg, err := uc.financeSvc.GetFinanceConfig(ctx)
		cfgCh <- configResult{cfg, err}
	}()

	go func() {
		investments, err := uc.financeSvc.GetUserInvestments(ctx, userID)
		invCh <- investResult{investments, err}
	}()

	go func() {
		wallet, err := uc.walletSvc.GetWallet(ctx, userID, entity.CurrencyUSDT)
		if err != nil {
			balCh <- balanceResult{decimal.Zero, err}
			return
		}
		balCh <- balanceResult{wallet.Balance, nil}
	}()

	cfgRes := <-cfgCh
	invRes := <-invCh
	balRes := <-balCh

	if cfgRes.err != nil {
		return nil, cfgRes.err
	}
	if invRes.err != nil {
		return nil, invRes.err
	}
	// 余额查询失败不影响主流程，使用零值
	usdtBalance := balRes.balance

	// 计算汇总
	totalAmount := decimal.Zero
	totalProfit := decimal.Zero
	yesterdayProfit := decimal.Zero
	for _, inv := range invRes.investments {
		totalAmount = totalAmount.Add(inv.Amount)
		totalProfit = totalProfit.Add(inv.TotalProfit)
		// 昨日收益 = 每笔持仓的日收益之和 (已开始计息的投资)
		yesterdayProfit = yesterdayProfit.Add(inv.DailyProfit())
	}

	return &FinanceDashboard{
		AnnualRate:      cfgRes.cfg.AnnualRate,
		MinInvest:       cfgRes.cfg.MinInvest,
		MaxInvest:       cfgRes.cfg.MaxInvest,
		Investments:     invRes.investments,
		TotalAmount:     totalAmount,
		TotalProfit:     totalProfit,
		YesterdayProfit: yesterdayProfit,
		USDTBalance:     usdtBalance,
	}, nil
}

// Invest 买入余额宝
func (uc *FinanceUseCase) Invest(ctx context.Context, userID uint64, amount decimal.Decimal) (*entity.FinanceInvestment, error) {
	return uc.financeSvc.Invest(ctx, userID, amount)
}

// Withdraw 取出余额宝投资
func (uc *FinanceUseCase) Withdraw(ctx context.Context, investmentID uint64, userID uint64) error {
	_, err := uc.financeSvc.Withdraw(ctx, investmentID, userID)
	return err
}

// GetFinanceConfig 获取余额宝配置 (供 handler 使用)
func (uc *FinanceUseCase) GetFinanceConfig(ctx context.Context) (*service.FinanceConfig, error) {
	return uc.financeSvc.GetFinanceConfig(ctx)
}

// GetUserInvestments 获取用户持有中的投资列表
func (uc *FinanceUseCase) GetUserInvestments(ctx context.Context, userID uint64) ([]*entity.FinanceInvestment, error) {
	return uc.financeSvc.GetUserInvestments(ctx, userID)
}

// GetUSDTBalance 获取用户 USDT 余额
func (uc *FinanceUseCase) GetUSDTBalance(ctx context.Context, userID uint64) (decimal.Decimal, error) {
	wallet, err := uc.walletSvc.GetWallet(ctx, userID, entity.CurrencyUSDT)
	if err != nil {
		return decimal.Zero, err
	}
	return wallet.Balance, nil
}
