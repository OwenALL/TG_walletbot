package app

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service"
)

// UserUseCase 用户用例层，编排注册、PIN 等业务流程
type UserUseCase struct {
	userSvc       *service.UserService
	walletSvc     *service.WalletService
	depositSvc    *service.DepositService
	addrGen       port.AddressGenerator
	walletKeyRepo port.WalletKeyRepository
	cryptoSvc     port.PrivateKeyCrypto
	encryptionKey string
	logger        *zap.Logger
}

// NewUserUseCase 创建用户用例
// depositSvc/addrGen/walletKeyRepo/cryptoSvc/encryptionKey 可传 nil (TRON 未启用时)
func NewUserUseCase(
	userSvc *service.UserService,
	walletSvc *service.WalletService,
	depositSvc *service.DepositService,
	addrGen port.AddressGenerator,
	walletKeyRepo port.WalletKeyRepository,
	cryptoSvc port.PrivateKeyCrypto,
	encryptionKey string,
	logger *zap.Logger,
) *UserUseCase {
	return &UserUseCase{
		userSvc:       userSvc,
		walletSvc:     walletSvc,
		depositSvc:    depositSvc,
		addrGen:       addrGen,
		walletKeyRepo: walletKeyRepo,
		cryptoSvc:     cryptoSvc,
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

// RegisterResult 注册结果
type RegisterResult struct {
	User    *entity.User
	IsNew   bool
	NeedPIN bool // 是否需要引导设置 PIN
}

// RegisterOrGet 注册或获取已有用户
// 新用户注册时自动生成 TRON 充值地址 (如果 TRON 已启用)
func (uc *UserUseCase) RegisterOrGet(ctx context.Context, telegramID int64, username, firstName, lastName, langCode string, isPremium bool) (*RegisterResult, error) {
	// 先查找用户
	user, err := uc.userSvc.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return &RegisterResult{
			User:    user,
			IsNew:   false,
			NeedPIN: !user.HasPIN(),
		}, nil
	}

	// 新用户注册
	user, err = uc.userSvc.Register(ctx, telegramID, username, firstName, lastName, langCode, isPremium)
	if err != nil {
		return nil, err
	}

	uc.logger.Info("新用户注册",
		zap.Int64("telegram_id", telegramID),
		zap.String("username", username),
		zap.Uint64("user_id", user.ID),
	)

	// TRON 地址生成 (异步不阻塞注册流程)
	if uc.addrGen != nil && uc.walletKeyRepo != nil && uc.cryptoSvc != nil {
		go uc.assignDepositAddress(user.ID)
	}

	return &RegisterResult{
		User:    user,
		IsNew:   true,
		NeedPIN: true,
	}, nil
}

// assignDepositAddress 为新用户生成 TRON 充值地址并存储
func (uc *UserUseCase) assignDepositAddress(userID uint64) {
	ctx := context.Background()

	// 检查用户是否已有地址 (防止重复生成)
	existing, err := uc.walletKeyRepo.FindByUserID(ctx, userID)
	if err != nil {
		uc.logger.Error("检查用户钱包私钥失败", zap.Uint64("user_id", userID), zap.Error(err))
		return
	}
	if existing != nil {
		return
	}

	// 生成 TRON 地址
	address, privateKeyHex, err := uc.addrGen.GenerateAddress()
	if err != nil {
		uc.logger.Error("生成 TRON 地址失败", zap.Uint64("user_id", userID), zap.Error(err))
		return
	}

	// 加密私钥
	encryptedKey, err := uc.cryptoSvc.EncryptPrivateKey(privateKeyHex, uc.encryptionKey)
	if err != nil {
		uc.logger.Error("加密私钥失败", zap.Uint64("user_id", userID), zap.Error(err))
		return
	}

	// 主动清零明文私钥
	clearPrivateKey(&privateKeyHex)

	// 保存加密私钥到 wallet_keys 表
	walletKey := &entity.WalletKey{
		UserID:       userID,
		Address:      address,
		EncryptedKey: encryptedKey,
		KeyVersion:   1,
	}
	if err := uc.walletKeyRepo.Create(ctx, walletKey); err != nil {
		uc.logger.Error("保存钱包私钥失败", zap.Uint64("user_id", userID), zap.Error(err))
		return
	}

	// 更新钱包表的充值地址
	if uc.depositSvc != nil {
		if err := uc.depositSvc.SetUserDepositAddress(ctx, userID, address); err != nil {
			uc.logger.Error("设置充值地址失败", zap.Uint64("user_id", userID), zap.Error(err))
			return
		}
	}

	uc.logger.Info("用户充值地址分配成功",
		zap.Uint64("user_id", userID),
		zap.String("address", address),
	)
}

// clearPrivateKey 清零私钥字符串
func clearPrivateKey(s *string) {
	b := []byte(*s)
	for i := range b {
		b[i] = 0
	}
	*s = ""
}

// WalletOverview 钱包概览 (主页用)
type WalletOverview struct {
	User     *entity.User
	Balances map[string]decimal.Decimal
}

// GetWalletOverview 获取钱包主页概览数据
func (uc *UserUseCase) GetWalletOverview(ctx context.Context, userID uint64) (*WalletOverview, error) {
	user, err := uc.userSvc.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	balances, err := uc.walletSvc.GetBalanceMap(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &WalletOverview{
		User:     user,
		Balances: balances,
	}, nil
}

// SetPIN 设置支付密码
func (uc *UserUseCase) SetPIN(ctx context.Context, userID uint64, pin string) error {
	err := uc.userSvc.SetPIN(ctx, userID, pin)
	if err != nil {
		return err
	}
	uc.logger.Info("用户设置 PIN 成功", zap.Uint64("user_id", userID))
	return nil
}

// VerifyPIN 验证支付密码
func (uc *UserUseCase) VerifyPIN(ctx context.Context, userID uint64, pin string) (bool, int, error) {
	return uc.userSvc.VerifyPIN(ctx, userID, pin)
}

// NeedPIN 判断是否需要 PIN 验证
func (uc *UserUseCase) NeedPIN(ctx context.Context, userID uint64, currency string, amount decimal.Decimal) (bool, error) {
	return uc.userSvc.NeedPIN(ctx, userID, currency, amount)
}
