// Package notification 提供通知服务的适配器实现
package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// TelegramNotifier 通过 Telegram Bot 发送通知
// 实现 port.NotificationService 接口
// 所有通知异步发送 (goroutine)，不阻塞主流程
type TelegramNotifier struct {
	bot        *bot.Bot
	logger     *zap.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewTelegramNotifier 创建 Telegram 通知服务
func NewTelegramNotifier(b *bot.Bot, logger *zap.Logger) *TelegramNotifier {
	return &TelegramNotifier{
		bot:        b,
		logger:     logger,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

// SendDepositNotification 发送充值到账通知
func (n *TelegramNotifier) SendDepositNotification(ctx context.Context, telegramID int64, currency, amount, txHash, newBalance string) {
	text := fmt.Sprintf(
		"<b>充值到账通知</b>\n\n"+
			"币种: <code>%s</code>\n"+
			"金额: <code>%s</code>\n"+
			"交易哈希: <code>%s</code>\n"+
			"当前余额: <code>%s %s</code>",
		currency, amount, txHash, newBalance, currency,
	)
	n.sendAsync(telegramID, text)
}

// SendWithdrawalResult 发送提币处理结果通知
func (n *TelegramNotifier) SendWithdrawalResult(ctx context.Context, telegramID int64, approved bool, currency, amount, txHash, reason string) {
	var text string
	if approved {
		text = fmt.Sprintf(
			"<b>提币完成通知</b>\n\n"+
				"币种: <code>%s</code>\n"+
				"金额: <code>%s</code>\n"+
				"交易哈希: <code>%s</code>\n\n"+
				"提币已成功发送至目标地址",
			currency, amount, txHash,
		)
	} else {
		reasonText := reason
		if reasonText == "" {
			reasonText = "审核未通过"
		}
		text = fmt.Sprintf(
			"<b>提币审核通知</b>\n\n"+
				"币种: <code>%s</code>\n"+
				"金额: <code>%s</code>\n"+
				"状态: 已拒绝\n"+
				"原因: %s\n\n"+
				"冻结金额已退回",
			currency, amount, reasonText,
		)
	}
	n.sendAsync(telegramID, text)
}

// SendTransferReceived 发送收到转账通知
func (n *TelegramNotifier) SendTransferReceived(ctx context.Context, telegramID int64, senderName, currency, amount, newBalance string) {
	text := fmt.Sprintf(
		"<b>收到转账</b>\n\n"+
			"转账人: %s\n"+
			"币种: <code>%s</code>\n"+
			"金额: <code>%s</code>\n"+
			"当前余额: <code>%s %s</code>",
		senderName, currency, amount, newBalance, currency,
	)
	n.sendAsync(telegramID, text)
}

// SendPaymentRequest 发送收款请求通知
func (n *TelegramNotifier) SendPaymentRequest(ctx context.Context, telegramID int64, requesterName, currency, amount string, requestID uint64) {
	text := fmt.Sprintf(
		"<b>收到收款请求</b>\n\n"+
			"请求人: %s\n"+
			"币种: <code>%s</code>\n"+
			"金额: <code>%s</code>",
		requesterName, currency, amount,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("确认付款", fmt.Sprintf("Transfer--pay_confirm_%d", requestID)),
			tgpkg.Button("拒绝", fmt.Sprintf("Transfer--pay_reject_%d", requestID)),
		).
		Build()

	n.sendAsyncWithKeyboard(telegramID, text, keyboard)
}

// SendSecurityAlert 发送安全提醒
func (n *TelegramNotifier) SendSecurityAlert(ctx context.Context, telegramID int64, alertType, message string) {
	text := fmt.Sprintf(
		"<b>安全提醒</b>\n\n"+
			"类型: %s\n"+
			"详情: %s",
		n.alertTypeText(alertType), message,
	)
	n.sendAsync(telegramID, text)
}

// alertTypeText 将 alertType 转换为友好文本
func (n *TelegramNotifier) alertTypeText(alertType string) string {
	switch alertType {
	case "pin_failed":
		return "支付密码错误"
	case "account_frozen":
		return "账户已冻结"
	case "login_new_device":
		return "新设备登录"
	case "large_withdrawal":
		return "大额提币"
	default:
		return "系统通知"
	}
}

// sendAsync 异步发送消息，带重试机制
func (n *TelegramNotifier) sendAsync(telegramID int64, text string) {
	go func() {
		ctx := context.Background()
		for i := 0; i < n.maxRetries; i++ {
			_, err := tgpkg.SendMessage(ctx, n.bot, telegramID, text)
			if err == nil {
				return
			}
			n.logger.Warn("发送通知失败，准备重试",
				zap.Int64("telegram_id", telegramID),
				zap.Int("retry", i+1),
				zap.Error(err),
			)
			time.Sleep(n.retryDelay * time.Duration(i+1))
		}
		n.logger.Error("发送通知最终失败",
			zap.Int64("telegram_id", telegramID),
			zap.String("text_preview", truncate(text, 100)),
		)
	}()
}

// sendAsyncWithKeyboard 异步发送带键盘的消息，带重试机制
func (n *TelegramNotifier) sendAsyncWithKeyboard(telegramID int64, text string, keyboard *models.InlineKeyboardMarkup) {
	go func() {
		ctx := context.Background()
		for i := 0; i < n.maxRetries; i++ {
			_, err := tgpkg.SendMessageWithKeyboard(ctx, n.bot, telegramID, text, keyboard)
			if err == nil {
				return
			}
			n.logger.Warn("发送通知失败，准备重试",
				zap.Int64("telegram_id", telegramID),
				zap.Int("retry", i+1),
				zap.Error(err),
			)
			time.Sleep(n.retryDelay * time.Duration(i+1))
		}
		n.logger.Error("发送通知最终失败",
			zap.Int64("telegram_id", telegramID),
			zap.String("text_preview", truncate(text, 100)),
		)
	}()
}

// truncate 截断字符串用于日志输出
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
