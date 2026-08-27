package tron

import (
	"context"
	"fmt"
	"math/big"
	"unsafe"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
)

// WithdrawalSender 混合模式提币发送器
// amount <= threshold: 自动签名广播
// amount > threshold: 标记为需人工处理
type WithdrawalSender struct {
	client        *Client
	crypto        *CryptoService
	hotWalletAddr string
	hotWalletKey  string                       // 加密后的热钱包私钥
	encryptionKey string                       // AES 解密密钥
	configRepo    port.SystemConfigRepository  // 系统配置仓库 (动态读取阈值等运营参数)
	usdtContract  string                       // USDT TRC20 合约地址
	logger        *zap.Logger
}

// NewWithdrawalSender 创建提币发送器
// configRepo 用于动态读取 withdraw_usdt_auto_threshold 等运营参数
func NewWithdrawalSender(
	client *Client,
	crypto *CryptoService,
	hotWalletAddr string,
	hotWalletKey string,
	encryptionKey string,
	configRepo port.SystemConfigRepository,
	usdtContract string,
	logger *zap.Logger,
) *WithdrawalSender {
	return &WithdrawalSender{
		client:        client,
		crypto:        crypto,
		hotWalletAddr: hotWalletAddr,
		hotWalletKey:  hotWalletKey,
		encryptionKey: encryptionKey,
		configRepo:    configRepo,
		usdtContract:  usdtContract,
		logger:        logger,
	}
}

// SendWithdrawal 执行提币发送
// 返回: (txHash, isManual, error)
// amount <= threshold: 自动签名广播，返回 txHash
// amount > threshold: 返回 ("", true, nil) 表示需人工处理
// 阈值从 system_configs 表动态读取，管理员可实时调整无需重启
func (s *WithdrawalSender) SendWithdrawal(ctx context.Context, toAddress, currency string, amount decimal.Decimal) (string, bool, error) {
	// 从 DB 动态读取自动发送阈值 (默认 1000)
	threshold := decimal.NewFromInt(1000)
	if thresholdStr, err := s.configRepo.Get(ctx, entity.ConfigWithdrawUSDTAutoThreshold); err == nil && thresholdStr != "" {
		if parsed, err := decimal.NewFromString(thresholdStr); err == nil {
			threshold = parsed
		}
	}

	// 检查是否超过自动发送阈值
	if amount.GreaterThan(threshold) {
		s.logger.Info("提币金额超过自动发送阈值，转人工处理",
			zap.String("to_address", toAddress),
			zap.String("currency", currency),
			zap.String("amount", amount.String()),
			zap.String("threshold", threshold.String()),
		)
		return "", true, nil
	}

	// 解密热钱包私钥
	privateKeyHex, err := s.crypto.DecryptPrivateKey(s.hotWalletKey, s.encryptionKey)
	if err != nil {
		return "", false, fmt.Errorf("解密热钱包私钥失败: %w", err)
	}
	defer clearString(&privateKeyHex) // 签名完成后清零内存

	// 根据币种调用不同的发送方法
	switch currency {
	case entity.CurrencyTRX:
		return s.sendTRX(ctx, toAddress, amount, privateKeyHex)
	case entity.CurrencyUSDT:
		return s.sendUSDT(ctx, toAddress, amount, privateKeyHex)
	default:
		return "", false, fmt.Errorf("不支持的提币币种: %s", currency)
	}
}

// sendTRX 发送 TRX
func (s *WithdrawalSender) sendTRX(ctx context.Context, toAddress string, amount decimal.Decimal, privateKeyHex string) (string, bool, error) {
	// 转换为 sun (1 TRX = 1,000,000 sun)
	amountSun := amount.Shift(6).IntPart()

	result, err := s.client.SendTRX(ctx, s.hotWalletAddr, toAddress, amountSun, privateKeyHex)
	if err != nil {
		return "", false, fmt.Errorf("发送 TRX 失败: %w", err)
	}

	if !result.Success {
		return "", false, fmt.Errorf("TRX 交易广播失败")
	}

	s.logger.Info("TRX 提币发送成功",
		zap.String("tx_id", result.TxID),
		zap.String("to_address", toAddress),
		zap.String("amount", amount.String()),
	)

	return result.TxID, false, nil
}

// sendUSDT 发送 USDT TRC20
func (s *WithdrawalSender) sendUSDT(ctx context.Context, toAddress string, amount decimal.Decimal, privateKeyHex string) (string, bool, error) {
	// USDT TRC20 精度 6 位
	amountBig := amount.Shift(6).BigInt()
	if amountBig.Cmp(big.NewInt(0)) <= 0 {
		return "", false, fmt.Errorf("USDT 金额无效: %s", amount.String())
	}

	result, err := s.client.SendTRC20(ctx, s.hotWalletAddr, toAddress, s.usdtContract, amountBig, privateKeyHex)
	if err != nil {
		return "", false, fmt.Errorf("发送 USDT 失败: %w", err)
	}

	if !result.Success {
		return "", false, fmt.Errorf("USDT 交易广播失败")
	}

	s.logger.Info("USDT 提币发送成功",
		zap.String("tx_id", result.TxID),
		zap.String("to_address", toAddress),
		zap.String("amount", amount.String()),
	)

	return result.TxID, false, nil
}

// clearString 清零字符串底层内存 (安全: 用完后清除内存中的私钥)
// 使用 unsafe 直接访问字符串底层字节数组，避免 []byte(*s) 产生副本导致清零无效
func clearString(s *string) {
	if len(*s) == 0 {
		return
	}
	p := unsafe.StringData(*s)
	b := unsafe.Slice(p, len(*s))
	for i := range b {
		b[i] = 0
	}
	*s = ""
}
