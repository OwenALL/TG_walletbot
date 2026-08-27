package service

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// TransferService 转账领域服务
type TransferService struct {
	db              *gorm.DB
	walletRepo      port.WalletRepository
	transferRepo    port.TransferRepository
	transactionRepo port.TransactionRepository
	userRepo        port.UserRepository
	logger          *zap.Logger
}

// NewTransferService 创建转账领域服务
func NewTransferService(
	db *gorm.DB,
	walletRepo port.WalletRepository,
	transferRepo port.TransferRepository,
	transactionRepo port.TransactionRepository,
	userRepo port.UserRepository,
	logger *zap.Logger,
) *TransferService {
	return &TransferService{
		db:              db,
		walletRepo:      walletRepo,
		transferRepo:    transferRepo,
		transactionRepo: transactionRepo,
		userRepo:        userRepo,
		logger:          logger,
	}
}

// TransferResult 转账执行结果
type TransferResult struct {
	Transfer         *entity.Transfer
	SenderNewBalance decimal.Decimal
	ReceiverNewBalance decimal.Decimal
}

// ExecuteTransfer 执行转账 (事务)
// fromUserID: 发送方用户 ID
// toUserID: 接收方用户 ID
// currency: 币种
// amount: 转账金额
// transferType: 转账类型 (主动转账/收款请求)
func (s *TransferService) ExecuteTransfer(ctx context.Context, fromUserID, toUserID uint64, currency string, amount decimal.Decimal, transferType int8) (*TransferResult, error) {
	var result TransferResult

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 构建事务内的 repo
		walletRepo := newTxWalletRepo(tx)
		transferRepo := newTxTransferRepo(tx)
		transactionRepo := newTxTransactionRepo(tx)

		// 1. 查询发送方钱包并加锁
		senderWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, fromUserID, currency)
		if err != nil {
			return apperrors.Wrap(err, "查询发送方钱包失败")
		}
		if senderWallet == nil {
			return apperrors.NewNotFound("钱包", currency)
		}

		// 2. 验证余额
		if !senderWallet.HasSufficientBalance(amount) {
			return apperrors.NewInsufficientBalance(currency)
		}

		// 3. 查询接收方钱包并加锁
		receiverWallet, err := walletRepo.FindByUserIDAndCurrencyForUpdate(ctx, toUserID, currency)
		if err != nil {
			return apperrors.Wrap(err, "查询接收方钱包失败")
		}
		if receiverWallet == nil {
			return apperrors.NewNotFound("接收方钱包", currency)
		}

		// 4. 创建转账记录
		now := time.Now()
		transfer := &entity.Transfer{
			FromUserID:  fromUserID,
			ToUserID:    toUserID,
			Currency:    currency,
			Amount:      amount,
			Type:        transferType,
			Status:      entity.TransferStatusCompleted,
			CreatedAt:   now,
			CompletedAt: &now,
		}
		if err := transferRepo.Create(ctx, transfer); err != nil {
			return apperrors.Wrap(err, "创建转账记录失败")
		}

		// 5. 扣减发送方余额
		senderBalanceBefore := senderWallet.Balance
		senderBalanceAfter := senderBalanceBefore.Sub(amount)
		if err := walletRepo.UpdateBalance(ctx, senderWallet.ID, amount.Neg()); err != nil {
			return apperrors.Wrap(err, "扣减发送方余额失败")
		}

		// 6. 增加接收方余额
		receiverBalanceBefore := receiverWallet.Balance
		receiverBalanceAfter := receiverBalanceBefore.Add(amount)
		if err := walletRepo.UpdateBalance(ctx, receiverWallet.ID, amount); err != nil {
			return apperrors.Wrap(err, "增加接收方余额失败")
		}

		// 7. 创建发送方交易记录 (转出)
		senderTx := &entity.Transaction{
			UserID:        fromUserID,
			Type:          entity.TxTypeTransferOut,
			Currency:      currency,
			Amount:        amount,
			Fee:           decimal.Zero,
			BalanceBefore: senderBalanceBefore,
			BalanceAfter:  senderBalanceAfter,
			RelatedID:     &transfer.ID,
			RelatedType:   "transfer",
			Memo:          "转账",
			CreatedAt:     now,
		}
		if err := transactionRepo.Create(ctx, senderTx); err != nil {
			return apperrors.Wrap(err, "创建发送方交易记录失败")
		}

		// 8. 创建接收方交易记录 (转入)
		receiverTx := &entity.Transaction{
			UserID:        toUserID,
			Type:          entity.TxTypeTransferIn,
			Currency:      currency,
			Amount:        amount,
			Fee:           decimal.Zero,
			BalanceBefore: receiverBalanceBefore,
			BalanceAfter:  receiverBalanceAfter,
			RelatedID:     &transfer.ID,
			RelatedType:   "transfer",
			Memo:          "收到转账",
			CreatedAt:     now,
		}
		if err := transactionRepo.Create(ctx, receiverTx); err != nil {
			return apperrors.Wrap(err, "创建接收方交易记录失败")
		}

		result.Transfer = transfer
		result.SenderNewBalance = senderBalanceAfter
		result.ReceiverNewBalance = receiverBalanceAfter
		return nil
	})

	if err != nil {
		s.logger.Error("转账执行失败",
			zap.Uint64("from_user_id", fromUserID),
			zap.Uint64("to_user_id", toUserID),
			zap.String("currency", currency),
			zap.String("amount", amount.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("转账成功",
		zap.Uint64("from_user_id", fromUserID),
		zap.Uint64("to_user_id", toUserID),
		zap.String("currency", currency),
		zap.String("amount", amount.String()),
		zap.Uint64("transfer_id", result.Transfer.ID),
	)

	return &result, nil
}

// CreateReceiveRequest 创建收款请求
func (s *TransferService) CreateReceiveRequest(ctx context.Context, fromUserID, toUserID uint64, currency string, amount decimal.Decimal) (*entity.Transfer, error) {
	transfer := &entity.Transfer{
		FromUserID: toUserID,   // 付款人
		ToUserID:   fromUserID, // 收款人 (发起请求的人)
		Currency:   currency,
		Amount:     amount,
		Type:       entity.TransferTypeRequest,
		Status:     entity.TransferStatusPending,
		CreatedAt:  time.Now(),
	}
	if err := s.transferRepo.Create(ctx, transfer); err != nil {
		return nil, apperrors.Wrap(err, "创建收款请求失败")
	}
	return transfer, nil
}

// RejectReceiveRequest 拒绝收款请求
func (s *TransferService) RejectReceiveRequest(ctx context.Context, transferID uint64) error {
	transfer, err := s.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return apperrors.Wrap(err, "查询转账记录失败")
	}
	if transfer == nil {
		return apperrors.NewNotFound("转账记录", transferID)
	}
	if transfer.Status != entity.TransferStatusPending {
		return apperrors.NewBadRequest("该请求已处理")
	}
	transfer.Status = entity.TransferStatusRejected
	now := time.Now()
	transfer.CompletedAt = &now
	return s.transferRepo.Update(ctx, transfer)
}

// GetTransferByID 根据 ID 查询转账记录
func (s *TransferService) GetTransferByID(ctx context.Context, id uint64) (*entity.Transfer, error) {
	return s.transferRepo.FindByID(ctx, id)
}

// FindRecipient 查找收款人 (通过用户名或 Telegram ID)
func (s *TransferService) FindRecipient(ctx context.Context, identifier string) (*entity.User, error) {
	// 尝试通过用户名查找 (去除 @ 前缀)
	username := identifier
	if len(username) > 0 && username[0] == '@' {
		username = username[1:]
	}
	if username != "" {
		user, err := s.userRepo.FindByUsername(ctx, username)
		if err != nil {
			return nil, apperrors.Wrap(err, "查询用户失败")
		}
		if user != nil {
			return user, nil
		}
	}
	return nil, nil
}

// --- 事务内使用的辅助 repo (直接在事务 db 上操作) ---

type txWalletRepo struct {
	tx *gorm.DB
}

func newTxWalletRepo(tx *gorm.DB) *txWalletRepo {
	return &txWalletRepo{tx: tx}
}

// FindByUserIDAndCurrencyForUpdate 带行锁查询钱包
func (r *txWalletRepo) FindByUserIDAndCurrencyForUpdate(ctx context.Context, userID uint64, currency string) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.tx.WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE").
		Where("user_id = ? AND currency = ?", userID, currency).
		First(&wallet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

// UpdateBalance 事务内更新余额
func (r *txWalletRepo) UpdateBalance(ctx context.Context, walletID uint64, amount decimal.Decimal) error {
	result := r.tx.WithContext(ctx).Model(&entity.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", gorm.Expr("balance + ?", amount))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.NewNotFound("钱包", walletID)
	}
	return nil
}

type txTransferRepo struct {
	tx *gorm.DB
}

func newTxTransferRepo(tx *gorm.DB) *txTransferRepo {
	return &txTransferRepo{tx: tx}
}

func (r *txTransferRepo) Create(ctx context.Context, transfer *entity.Transfer) error {
	return r.tx.WithContext(ctx).Create(transfer).Error
}

type txTransactionRepo struct {
	tx *gorm.DB
}

func newTxTransactionRepo(tx *gorm.DB) *txTransactionRepo {
	return &txTransactionRepo{tx: tx}
}

func (r *txTransactionRepo) Create(ctx context.Context, transaction *entity.Transaction) error {
	return r.tx.WithContext(ctx).Create(transaction).Error
}
