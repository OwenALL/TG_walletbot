package app

import (
	"context"
	"strconv"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/service"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// TransferUseCase 转账用例层
type TransferUseCase struct {
	transferSvc *service.TransferService
	walletSvc   *service.WalletService
	userSvc     *service.UserService
	logger      *zap.Logger
}

// NewTransferUseCase 创建转账用例
func NewTransferUseCase(
	transferSvc *service.TransferService,
	walletSvc *service.WalletService,
	userSvc *service.UserService,
	logger *zap.Logger,
) *TransferUseCase {
	return &TransferUseCase{
		transferSvc: transferSvc,
		walletSvc:   walletSvc,
		userSvc:     userSvc,
		logger:      logger,
	}
}

// FindRecipientByUsername 通过用户名查找收款人
func (uc *TransferUseCase) FindRecipientByUsername(ctx context.Context, username string) (*entity.User, error) {
	return uc.transferSvc.FindRecipient(ctx, username)
}

// FindRecipientByTelegramID 通过 Telegram ID 查找收款人
func (uc *TransferUseCase) FindRecipientByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error) {
	user, err := uc.userSvc.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindRecipientByID 通过数字 ID (尝试作为 Telegram ID) 查找收款人
func (uc *TransferUseCase) FindRecipientByID(ctx context.Context, idStr string) (*entity.User, error) {
	telegramID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, nil
	}
	return uc.FindRecipientByTelegramID(ctx, telegramID)
}

// ValidateTransferAmount 验证转账金额
func (uc *TransferUseCase) ValidateTransferAmount(currency string, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return apperrors.NewBadRequest("金额必须大于 0")
	}

	var minAmount decimal.Decimal
	switch currency {
	case entity.CurrencyUSDT:
		minAmount = decimal.NewFromFloat(0.01)
	case entity.CurrencyTRX:
		minAmount = decimal.NewFromInt(1)
	case entity.CurrencyCNY:
		minAmount = decimal.NewFromInt(1)
	default:
		return apperrors.NewBadRequest("不支持的币种")
	}

	if amount.LessThan(minAmount) {
		return apperrors.NewBadRequest("最低转账金额为 " + minAmount.String() + " " + currency)
	}

	return nil
}

// CheckBalance 检查余额是否充足
func (uc *TransferUseCase) CheckBalance(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) error {
	return uc.walletSvc.CheckBalance(ctx, userID, currency, amount)
}

// ExecuteTransfer 执行转账
func (uc *TransferUseCase) ExecuteTransfer(ctx context.Context, fromUserID, toUserID uint64, currency string, amount decimal.Decimal) (*service.TransferResult, error) {
	// 不能转给自己
	if fromUserID == toUserID {
		return nil, apperrors.NewTransferToSelf()
	}

	// 检查接收方是否存在
	receiver, err := uc.userSvc.GetByID(ctx, toUserID)
	if err != nil {
		return nil, err
	}
	if receiver == nil {
		return nil, apperrors.NewRecipientNotFound()
	}
	if !receiver.IsActive() {
		return nil, apperrors.NewBadRequest("收款人账户已冻结")
	}

	return uc.transferSvc.ExecuteTransfer(ctx, fromUserID, toUserID, currency, amount, entity.TransferTypeDirect)
}

// CreateReceiveRequest 创建收款请求
func (uc *TransferUseCase) CreateReceiveRequest(ctx context.Context, fromUserID, payerUserID uint64, currency string, amount decimal.Decimal) (*entity.Transfer, error) {
	// 不能向自己收款
	if fromUserID == payerUserID {
		return nil, apperrors.NewTransferToSelf()
	}

	// 检查付款人是否存在
	payer, err := uc.userSvc.GetByID(ctx, payerUserID)
	if err != nil {
		return nil, err
	}
	if payer == nil {
		return nil, apperrors.NewRecipientNotFound()
	}

	return uc.transferSvc.CreateReceiveRequest(ctx, fromUserID, payerUserID, currency, amount)
}

// ExecuteReceivePayment 执行收款请求的付款
// 根据转账记录 ID 查找并执行收款请求的付款
func (uc *TransferUseCase) ExecuteReceivePayment(ctx context.Context, transferID uint64) (*service.TransferResult, error) {
	transfer, err := uc.transferSvc.GetTransferByID(ctx, transferID)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		return nil, apperrors.NewNotFound("转账记录", transferID)
	}
	if transfer.Status != entity.TransferStatusPending {
		return nil, apperrors.NewBadRequest("该请求已处理")
	}
	if transfer.Type != entity.TransferTypeRequest {
		return nil, apperrors.NewBadRequest("非收款请求")
	}

	// 在收款请求中: FromUserID 是付款人, ToUserID 是收款人
	return uc.transferSvc.ExecuteTransfer(ctx, transfer.FromUserID, transfer.ToUserID, transfer.Currency, transfer.Amount, entity.TransferTypeRequest)
}

// GetTransferByID 根据 ID 查询转账记录
func (uc *TransferUseCase) GetTransferByID(ctx context.Context, id uint64) (*entity.Transfer, error) {
	return uc.transferSvc.GetTransferByID(ctx, id)
}

// GetUserByID 根据 ID 查询用户
func (uc *TransferUseCase) GetUserByID(ctx context.Context, id uint64) (*entity.User, error) {
	return uc.userSvc.GetByID(ctx, id)
}

// RejectReceiveRequest 拒绝收款请求
func (uc *TransferUseCase) RejectReceiveRequest(ctx context.Context, transferID uint64) error {
	return uc.transferSvc.RejectReceiveRequest(ctx, transferID)
}

// GetBalance 获取用户余额
func (uc *TransferUseCase) GetBalance(ctx context.Context, userID uint64, currency string) (decimal.Decimal, error) {
	wallet, err := uc.walletSvc.GetWallet(ctx, userID, currency)
	if err != nil {
		return decimal.Zero, err
	}
	return wallet.Balance, nil
}
