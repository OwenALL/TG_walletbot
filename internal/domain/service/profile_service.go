package service

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// ProfileService 个人中心领域服务
type ProfileService struct {
	userRepo       port.UserRepository
	walletRepo     port.WalletRepository
	settingsRepo   port.UserSettingsRepository
	txRepo         port.TransactionRepository
	withdrawalRepo port.WithdrawalRepository
}

// NewProfileService 创建个人中心领域服务
func NewProfileService(
	userRepo port.UserRepository,
	walletRepo port.WalletRepository,
	settingsRepo port.UserSettingsRepository,
	txRepo port.TransactionRepository,
	withdrawalRepo port.WithdrawalRepository,
) *ProfileService {
	return &ProfileService{
		userRepo:       userRepo,
		walletRepo:     walletRepo,
		settingsRepo:   settingsRepo,
		txRepo:         txRepo,
		withdrawalRepo: withdrawalRepo,
	}
}

// ProfileInfo 个人中心主页数据
type ProfileInfo struct {
	User     *entity.User
	Balances map[string]decimal.Decimal
}

// GetProfileInfo 获取个人中心信息
func (s *ProfileService) GetProfileInfo(ctx context.Context, userID uint64) (*ProfileInfo, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户失败")
	}
	if user == nil {
		return nil, apperrors.NewNotFound("用户", userID)
	}

	wallets, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询钱包失败")
	}

	balances := make(map[string]decimal.Decimal)
	for _, w := range wallets {
		balances[w.Currency] = w.Balance
	}

	return &ProfileInfo{
		User:     user,
		Balances: balances,
	}, nil
}

// BillsPage 账单列表分页数据
type BillsPage struct {
	Items      []*entity.Transaction
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// GetBills 获取账单列表 (支持币种筛选 + 分页)
func (s *ProfileService) GetBills(ctx context.Context, userID uint64, currency string, page, pageSize int) (*BillsPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	filter := &port.TransactionFilter{}
	if currency != "" && currency != "ALL" {
		filter.Currency = currency
	}

	offset := (page - 1) * pageSize
	items, total, err := s.txRepo.ListByUserID(ctx, userID, filter, offset, pageSize)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询账单失败")
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &BillsPage{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// WithdrawalHistoryPage 提币历史分页数据
type WithdrawalHistoryPage struct {
	Items      []*entity.Withdrawal
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// GetWithdrawalHistory 获取提币历史 (分页)
func (s *ProfileService) GetWithdrawalHistory(ctx context.Context, userID uint64, page, pageSize int) (*WithdrawalHistoryPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	items, total, err := s.withdrawalRepo.ListByUserID(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询提币历史失败")
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &WithdrawalHistoryPage{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// SmallFreeInfo 小额免密额度信息
type SmallFreeInfo struct {
	USDTLimit decimal.Decimal
	TRXLimit  decimal.Decimal
}

// GetSmallFreeInfo 获取小额免密额度
func (s *ProfileService) GetSmallFreeInfo(ctx context.Context, userID uint64) (*SmallFreeInfo, error) {
	settings, err := s.settingsRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询用户配置失败")
	}
	if settings == nil {
		// 返回默认值
		return &SmallFreeInfo{
			USDTLimit: decimal.NewFromInt(100),
			TRXLimit:  decimal.NewFromInt(100),
		}, nil
	}

	return &SmallFreeInfo{
		USDTLimit: settings.SmallFreeUSDT,
		TRXLimit:  settings.SmallFreeTRX,
	}, nil
}

// UpdateLanguage 更新用户语言设置
func (s *ProfileService) UpdateLanguage(ctx context.Context, userID uint64, language string) error {
	settings, err := s.settingsRepo.FindByUserID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户配置失败")
	}

	if settings == nil {
		// 创建默认配置
		settings = &entity.UserSettings{
			UserID:              userID,
			Language:            language,
			SmallFreeUSDT:       decimal.NewFromInt(100),
			SmallFreeTRX:        decimal.NewFromInt(100),
			NotificationEnabled: true,
		}
		return s.settingsRepo.Upsert(ctx, settings)
	}

	settings.Language = language
	return s.settingsRepo.Update(ctx, settings)
}

// UpdateSmallFreeLimit 更新小额免密额度
func (s *ProfileService) UpdateSmallFreeLimit(ctx context.Context, userID uint64, currency string, newLimit decimal.Decimal) error {
	if newLimit.IsNegative() {
		return apperrors.NewBadRequest("额度不能为负数")
	}

	settings, err := s.settingsRepo.FindByUserID(ctx, userID)
	if err != nil {
		return apperrors.Wrap(err, "查询用户配置失败")
	}

	if settings == nil {
		// 创建默认配置
		settings = &entity.UserSettings{
			UserID:              userID,
			SmallFreeUSDT:       decimal.NewFromInt(100),
			SmallFreeTRX:        decimal.NewFromInt(100),
			NotificationEnabled: true,
		}
	}

	switch currency {
	case entity.CurrencyUSDT:
		settings.SmallFreeUSDT = newLimit
	case entity.CurrencyTRX:
		settings.SmallFreeTRX = newLimit
	default:
		return apperrors.NewBadRequest("不支持该币种的免密设置")
	}

	if settings.ID == 0 {
		return s.settingsRepo.Upsert(ctx, settings)
	}
	return s.settingsRepo.Update(ctx, settings)
}
