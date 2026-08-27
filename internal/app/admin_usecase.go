package app

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// AdminUseCase 管理后台用例层
type AdminUseCase struct {
	adminSvc         *service.AdminService
	userRepo         port.UserRepository
	walletRepo       port.WalletRepository
	transactionRepo  port.TransactionRepository
	depositRepo      port.DepositRepository
	withdrawalRepo   port.WithdrawalRepository
	exchangeRateRepo port.ExchangeRateRepository
	systemConfigRepo port.SystemConfigRepository
	db               *gorm.DB
	logger           *zap.Logger
}

// NewAdminUseCase 创建管理后台用例
func NewAdminUseCase(
	adminSvc *service.AdminService,
	userRepo port.UserRepository,
	walletRepo port.WalletRepository,
	transactionRepo port.TransactionRepository,
	depositRepo port.DepositRepository,
	withdrawalRepo port.WithdrawalRepository,
	exchangeRateRepo port.ExchangeRateRepository,
	systemConfigRepo port.SystemConfigRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *AdminUseCase {
	return &AdminUseCase{
		adminSvc:         adminSvc,
		userRepo:         userRepo,
		walletRepo:       walletRepo,
		transactionRepo:  transactionRepo,
		depositRepo:      depositRepo,
		withdrawalRepo:   withdrawalRepo,
		exchangeRateRepo: exchangeRateRepo,
		systemConfigRepo: systemConfigRepo,
		db:               db,
		logger:           logger,
	}
}

// --- 认证 ---

// Login 管理员登录
func (uc *AdminUseCase) Login(ctx context.Context, username, password string) (*entity.AdminUser, error) {
	return uc.adminSvc.Authenticate(ctx, username, password)
}

// GetAdminByID 获取管理员信息
func (uc *AdminUseCase) GetAdminByID(ctx context.Context, id uint64) (*entity.AdminUser, error) {
	return uc.adminSvc.GetAdminByID(ctx, id)
}

// ChangePassword 修改管理员密码 (委托给 AdminService)
func (uc *AdminUseCase) ChangePassword(ctx context.Context, adminID uint64, oldPassword, newPassword string) error {
	return uc.adminSvc.ChangePassword(ctx, adminID, oldPassword, newPassword)
}

// --- 仪表盘 ---

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalUsers       int64           `json:"total_users"`
	TodayNewUsers    int64           `json:"today_new_users"`
	TotalUSDTBalance decimal.Decimal `json:"total_usdt_balance"`
	TotalTRXBalance  decimal.Decimal `json:"total_trx_balance"`
	TotalCNYBalance  decimal.Decimal `json:"total_cny_balance"`
	TodayTransactions int64          `json:"today_transactions"`
	PendingWithdrawals int64         `json:"pending_withdrawals"`
}

// GetDashboardStats 获取仪表盘统计
func (uc *AdminUseCase) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// 总用户数
	totalUsers, err := uc.userRepo.CountAll(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户数失败")
	}
	stats.TotalUsers = totalUsers

	// 今日新增用户
	var todayNewUsers int64
	today := time.Now().Format("2006-01-02")
	uc.db.WithContext(ctx).Model(&entity.User{}).
		Where("DATE(created_at) = ?", today).
		Count(&todayNewUsers)
	stats.TodayNewUsers = todayNewUsers

	// 各币种总余额
	type BalanceResult struct {
		Currency string
		Total    decimal.Decimal
	}
	var balances []BalanceResult
	uc.db.WithContext(ctx).Model(&entity.Wallet{}).
		Select("currency, COALESCE(SUM(balance), 0) as total").
		Group("currency").
		Scan(&balances)
	for _, b := range balances {
		switch b.Currency {
		case entity.CurrencyUSDT:
			stats.TotalUSDTBalance = b.Total
		case entity.CurrencyTRX:
			stats.TotalTRXBalance = b.Total
		case entity.CurrencyCNY:
			stats.TotalCNYBalance = b.Total
		}
	}

	// 今日交易笔数
	var todayTx int64
	uc.db.WithContext(ctx).Model(&entity.Transaction{}).
		Where("DATE(created_at) = ?", today).
		Count(&todayTx)
	stats.TodayTransactions = todayTx

	// 待审核提币数
	var pendingCount int64
	uc.db.WithContext(ctx).Model(&entity.Withdrawal{}).
		Where("status = ?", entity.WithdrawalStatusPending).
		Count(&pendingCount)
	stats.PendingWithdrawals = pendingCount

	return stats, nil
}

// TrendDataPoint 交易趋势数据点
type TrendDataPoint struct {
	Date   string          `json:"date"`
	Count  int64           `json:"count"`
	Volume decimal.Decimal `json:"volume"`
}

// GetDashboardTrend 获取交易趋势数据
// 按日期分组统计最近 N 天的交易笔数和交易量，没有交易的日期自动补零
func (uc *AdminUseCase) GetDashboardTrend(ctx context.Context, days int) ([]*TrendDataPoint, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	// 计算起始日期 (包含今天)
	startDate := time.Now().AddDate(0, 0, -(days - 1))
	startDateStr := startDate.Format("2006-01-02")

	// 查询数据库，按日期分组统计
	type trendRow struct {
		Date   string          `gorm:"column:date"`
		Count  int64           `gorm:"column:count"`
		Volume decimal.Decimal `gorm:"column:volume"`
	}

	var rows []trendRow
	err := uc.db.WithContext(ctx).Model(&entity.Transaction{}).
		Select("DATE(created_at) as date, COUNT(*) as count, COALESCE(SUM(amount), 0) as volume").
		Where("DATE(created_at) >= ?", startDateStr).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, apperrors.Wrap(err, "查询交易趋势失败")
	}

	// 构建日期 -> 数据映射
	rowMap := make(map[string]*trendRow, len(rows))
	for i := range rows {
		rowMap[rows[i].Date] = &rows[i]
	}

	// 生成完整的日期序列，没有数据的日期补零
	result := make([]*TrendDataPoint, 0, days)
	for i := 0; i < days; i++ {
		dateStr := startDate.AddDate(0, 0, i).Format("2006-01-02")
		point := &TrendDataPoint{
			Date:   dateStr,
			Count:  0,
			Volume: decimal.Zero,
		}
		if row, ok := rowMap[dateStr]; ok {
			point.Count = row.Count
			point.Volume = row.Volume
		}
		result = append(result, point)
	}

	return result, nil
}

// --- 用户管理 ---

// UserListFilter 用户列表筛选条件
type UserListFilter struct {
	Search string // Telegram ID 或 username 搜索
	Status *int8  // 用户状态筛选
}

// UserListItem 用户列表项 (含余额摘要)
type UserListItem struct {
	*entity.User
	USDTBalance decimal.Decimal `json:"usdt_balance"`
	TRXBalance  decimal.Decimal `json:"trx_balance"`
	CNYBalance  decimal.Decimal `json:"cny_balance"`
}

// ListUsers 获取用户列表 (支持搜索和筛选)，返回带钱包余额的列表项
func (uc *AdminUseCase) ListUsers(ctx context.Context, filter *UserListFilter, page, size int) ([]*UserListItem, int64, error) {
	offset := (page - 1) * size

	db := uc.db.WithContext(ctx).Model(&entity.User{})

	if filter != nil {
		if filter.Search != "" {
			db = db.Where("username LIKE ? OR CAST(telegram_id AS CHAR) LIKE ?",
				"%"+filter.Search+"%", "%"+filter.Search+"%")
		}
		if filter.Status != nil {
			db = db.Where("status = ?", *filter.Status)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询用户数失败")
	}

	var users []*entity.User
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询用户列表失败")
	}

	// 无用户时直接返回空列表
	if len(users) == 0 {
		return []*UserListItem{}, total, nil
	}

	// 批量获取用户 ID
	userIDs := make([]uint64, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	// 批量查询钱包余额
	var wallets []*entity.Wallet
	if err := uc.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&wallets).Error; err != nil {
		uc.logger.Warn("批量查询钱包余额失败，返回零余额", zap.Error(err))
		// 查询失败不阻塞列表返回，余额显示为零
	}

	// 构建 user_id -> currency -> balance 映射
	balanceMap := make(map[uint64]map[string]decimal.Decimal, len(users))
	for _, w := range wallets {
		if _, ok := balanceMap[w.UserID]; !ok {
			balanceMap[w.UserID] = make(map[string]decimal.Decimal)
		}
		balanceMap[w.UserID][w.Currency] = w.Balance
	}

	// 组装 UserListItem
	items := make([]*UserListItem, len(users))
	for i, u := range users {
		item := &UserListItem{User: u}
		item.HasPINSet = u.PINHash != ""
		if bm, ok := balanceMap[u.ID]; ok {
			item.USDTBalance = bm[entity.CurrencyUSDT]
			item.TRXBalance = bm[entity.CurrencyTRX]
			item.CNYBalance = bm[entity.CurrencyCNY]
		}
		items[i] = item
	}

	return items, total, nil
}

// UserDetail 用户详情 (匿名嵌入实现扁平 JSON 结构)
type UserDetail struct {
	*entity.User
	HasPIN  bool             `json:"has_pin"`
	Wallets []*entity.Wallet `json:"wallets"`
}

// GetUserDetail 获取用户详情
func (uc *AdminUseCase) GetUserDetail(ctx context.Context, userID uint64) (*UserDetail, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("用户", userID)
	}

	wallets, err := uc.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}

	return &UserDetail{
		User:    user,
		HasPIN:  user.PINHash != "",
		Wallets: wallets,
	}, nil
}

// UpdateUserStatus 更新用户状态 (冻结/解冻)
func (uc *AdminUseCase) UpdateUserStatus(ctx context.Context, userID uint64, status int8) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return apperrors.NewNotFound("用户", userID)
	}
	user.Status = status
	return uc.userRepo.Update(ctx, user)
}

// GetUserTransactions 获取用户交易记录
func (uc *AdminUseCase) GetUserTransactions(ctx context.Context, userID uint64, filter *port.TransactionFilter, page, size int) ([]*entity.Transaction, int64, error) {
	offset := (page - 1) * size
	return uc.transactionRepo.ListByUserID(ctx, userID, filter, offset, size)
}

// ResetUserPIN 重置用户支付密码（清空 PIN，解锁）
func (uc *AdminUseCase) ResetUserPIN(ctx context.Context, userID uint64) error {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return apperrors.NewNotFound("用户", userID)
	}

	user.PINHash = ""
	user.PINFailCount = 0
	user.PINLockedUntil = nil

	return uc.userRepo.Update(ctx, user)
}

// AdjustUserBalance 管理员调整用户余额
// amount 为正数表示增加，负数表示扣减
func (uc *AdminUseCase) AdjustUserBalance(ctx context.Context, userID uint64, currency string, amount decimal.Decimal, reason string) error {
	// 1. 查找用户是否存在
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return apperrors.NewNotFound("用户", userID)
	}

	// 2. 验证币种 (USDT/TRX/CNY)
	if !entity.IsValidCurrency(currency) {
		return apperrors.NewBadRequest("不支持的币种")
	}

	// 3. 获取钱包
	wallet, err := uc.walletRepo.FindByUserIDAndCurrency(ctx, userID, currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return apperrors.NewNotFound("钱包", currency)
	}

	// 4. 扣减时检查余额是否足够
	if amount.IsNegative() && wallet.Balance.Add(amount).IsNegative() {
		return apperrors.NewBadRequest("余额不足，当前余额: " + wallet.Balance.String() + " " + currency)
	}

	// 5. 更新余额
	if err := uc.walletRepo.UpdateBalance(ctx, wallet.ID, amount); err != nil {
		return apperrors.Wrap(err, "更新余额失败")
	}

	return nil
}

// --- 交易管理 ---

// TransactionListFilter 交易列表筛选
type TransactionListFilter struct {
	UserID   *uint64
	Type     string
	Currency string
}

// ListTransactions 获取交易流水列表
func (uc *AdminUseCase) ListTransactions(ctx context.Context, filter *TransactionListFilter, page, size int) ([]*entity.Transaction, int64, error) {
	offset := (page - 1) * size

	db := uc.db.WithContext(ctx).Model(&entity.Transaction{})

	if filter != nil {
		if filter.UserID != nil {
			db = db.Where("user_id = ?", *filter.UserID)
		}
		if filter.Type != "" {
			db = db.Where("type = ?", filter.Type)
		}
		if filter.Currency != "" {
			db = db.Where("currency = ?", filter.Currency)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询交易数失败")
	}

	var txs []*entity.Transaction
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&txs).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询交易列表失败")
	}

	return txs, total, nil
}

// ListDeposits 获取充值记录列表
func (uc *AdminUseCase) ListDeposits(ctx context.Context, page, size int) ([]*entity.Deposit, int64, error) {
	offset := (page - 1) * size

	var total int64
	db := uc.db.WithContext(ctx).Model(&entity.Deposit{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询充值数失败")
	}

	var deposits []*entity.Deposit
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&deposits).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询充值列表失败")
	}

	return deposits, total, nil
}

// --- 提币审核 ---

// WithdrawalListFilter 提币列表筛选
type WithdrawalListFilter struct {
	Status *int8
}

// ListWithdrawals 获取提币记录列表
func (uc *AdminUseCase) ListWithdrawals(ctx context.Context, filter *WithdrawalListFilter, page, size int) ([]*entity.Withdrawal, int64, error) {
	offset := (page - 1) * size

	db := uc.db.WithContext(ctx).Model(&entity.Withdrawal{})
	if filter != nil && filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询提币数失败")
	}

	var list []*entity.Withdrawal
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, apperrors.Wrap(err, "查询提币列表失败")
	}

	return list, total, nil
}

// ApproveWithdrawal 批准提币
func (uc *AdminUseCase) ApproveWithdrawal(ctx context.Context, withdrawalID uint64, reviewerID uint64) error {
	w, err := uc.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if w == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}
	if w.Status != entity.WithdrawalStatusPending {
		return apperrors.NewBadRequest("该提币记录非待审核状态")
	}

	now := time.Now()
	w.Status = entity.WithdrawalStatusProcessing
	w.ReviewerID = &reviewerID
	w.ReviewedAt = &now

	return uc.withdrawalRepo.Update(ctx, w)
}

// RejectWithdrawal 拒绝提币 (退回余额)
func (uc *AdminUseCase) RejectWithdrawal(ctx context.Context, withdrawalID uint64, reviewerID uint64, reason string) error {
	w, err := uc.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if w == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}
	if w.Status != entity.WithdrawalStatusPending {
		return apperrors.NewBadRequest("该提币记录非待审核状态")
	}

	now := time.Now()
	w.Status = entity.WithdrawalStatusRejected
	w.ReviewerID = &reviewerID
	w.ReviewNote = reason
	w.ReviewedAt = &now

	// 事务：更新状态 + 退回余额
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(w).Error; err != nil {
			return apperrors.Wrap(err, "更新提币状态失败")
		}
		// 退回冻结金额到可用余额
		result := tx.Model(&entity.Wallet{}).
			Where("user_id = ? AND currency = ?", w.UserID, w.Currency).
			Updates(map[string]interface{}{
				"balance":        gorm.Expr("balance + ?", w.Amount),
				"frozen_balance": gorm.Expr("frozen_balance - ?", w.Amount),
			})
		if result.Error != nil {
			return apperrors.Wrap(result.Error, "退回余额失败")
		}
		return nil
	})
}

// CompleteWithdrawalManually 管理员手动完成提币 (回填 txHash)
// 用于超过自动发送阈值的提币，管理员手动发送后回填交易哈希
func (uc *AdminUseCase) CompleteWithdrawalManually(ctx context.Context, withdrawalID uint64, txHash string, adminID uint64) error {
	w, err := uc.withdrawalRepo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperrors.Wrap(err, "查询提币记录失败")
	}
	if w == nil {
		return apperrors.NewNotFound("提币记录", withdrawalID)
	}

	// 只有处理中 (processing) 或待审核 (pending) 的提币可以手动完成
	if w.Status != entity.WithdrawalStatusProcessing && w.Status != entity.WithdrawalStatusPending {
		return apperrors.NewBadRequest("该提币记录状态不支持手动完成")
	}

	if txHash == "" {
		return apperrors.NewBadRequest("交易哈希不能为空")
	}

	// 获取钱包
	wallet, err := uc.walletRepo.FindByUserIDAndCurrency(ctx, w.UserID, w.Currency)
	if err != nil {
		return apperrors.Wrap(err, "查询钱包失败")
	}
	if wallet == nil {
		return apperrors.NewNotFound("钱包", w.Currency)
	}

	balanceBefore := wallet.Balance

	// 事务: 解冻余额 + 扣减余额 + 更新提币状态 + 创建流水
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 解冻余额
		result := tx.Model(&entity.Wallet{}).
			Where("id = ? AND frozen_balance >= ?", wallet.ID, w.Amount).
			Update("frozen_balance", gorm.Expr("frozen_balance - ?", w.Amount))
		if result.Error != nil {
			return apperrors.Wrap(result.Error, "解冻余额失败")
		}

		// 扣减余额
		result = tx.Model(&entity.Wallet{}).
			Where("id = ?", wallet.ID).
			Update("balance", gorm.Expr("balance - ?", w.Amount))
		if result.Error != nil {
			return apperrors.Wrap(result.Error, "扣减余额失败")
		}

		// 创建交易流水
		txRecord := &entity.Transaction{
			UserID:        w.UserID,
			Type:          entity.TxTypeWithdraw,
			Currency:      w.Currency,
			Amount:        w.Amount,
			Fee:           w.Fee,
			BalanceBefore: balanceBefore,
			BalanceAfter:  balanceBefore.Sub(w.Amount),
			RelatedID:     &w.ID,
			RelatedType:   "withdrawal",
			Memo:          "提币 " + w.ToAddress,
		}
		if err := tx.Create(txRecord).Error; err != nil {
			return apperrors.Wrap(err, "创建提币流水失败")
		}

		// 更新提币记录
		now := time.Now()
		w.Status = entity.WithdrawalStatusCompleted
		w.TxHash = txHash
		w.CompletedAt = &now
		w.ReviewerID = &adminID
		if w.ReviewedAt == nil {
			w.ReviewedAt = &now
		}

		if err := tx.Save(w).Error; err != nil {
			return apperrors.Wrap(err, "更新提币状态失败")
		}

		return nil
	})
}

// --- 汇率管理 ---

// GetAllExchangeRates 获取所有汇率
func (uc *AdminUseCase) GetAllExchangeRates(ctx context.Context) ([]*entity.ExchangeRate, error) {
	return uc.exchangeRateRepo.FindAll(ctx)
}

// UpdateExchangeRate 更新汇率
func (uc *AdminUseCase) UpdateExchangeRate(ctx context.Context, rate *entity.ExchangeRate) error {
	rate.UpdatedAt = time.Now()
	return uc.exchangeRateRepo.Upsert(ctx, rate)
}

// --- 系统配置 ---

// GetAllConfigs 获取所有系统配置
func (uc *AdminUseCase) GetAllConfigs(ctx context.Context) ([]*entity.SystemConfig, error) {
	return uc.systemConfigRepo.GetAll(ctx)
}

// UpdateConfig 更新系统配置
func (uc *AdminUseCase) UpdateConfig(ctx context.Context, key, value string) error {
	return uc.systemConfigRepo.Set(ctx, key, value)
}

// --- 审计日志 ---

// CreateAuditLog 创建审计日志 (委托给 AdminService)
func (uc *AdminUseCase) CreateAuditLog(ctx context.Context, adminID uint64, action, targetType string, targetID *uint64, detail, ipAddress string) {
	uc.adminSvc.CreateAuditLog(ctx, adminID, action, targetType, targetID, detail, ipAddress)
}

// ListAuditLogs 获取审计日志列表 (委托给 AdminService)
func (uc *AdminUseCase) ListAuditLogs(ctx context.Context, filter *port.AdminLogFilter, page, size int) ([]*entity.AdminLog, int64, error) {
	offset := (page - 1) * size
	return uc.adminSvc.ListAuditLogs(ctx, filter, offset, size)
}
