package handler

import (
	"context"
	"fmt"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// DepositHandler 充值相关回调处理器
type DepositHandler struct {
	depositUC *app.DepositUseCase
	logger    *zap.Logger
}

// NewDepositHandler 创建充值处理器
func NewDepositHandler(depositUC *app.DepositUseCase, logger *zap.Logger) *DepositHandler {
	return &DepositHandler{
		depositUC: depositUC,
		logger:    logger,
	}
}

// HandleCallback 处理充值相关的回调
// action: tronlink_deposit
func (h *DepositHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID

	// 立即响应回调
	tgpkg.AnswerCallbackWithText(ctx, b, query.ID, "确认")

	_, action := telegram.ParseCallbackData(query.Data)

	switch action {
	case "tronlink_deposit":
		h.showDepositPage(ctx, b, chatID, messageID, query.From.ID)
	default:
		h.logger.Warn("未知充值回调", zap.String("action", action))
	}
}

// showDepositPage 显示充值页面 (地址 + 二维码)
func (h *DepositHandler) showDepositPage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取充值信息
	info, err := h.depositUC.GetDepositInfo(ctx, user.ID)
	if err != nil {
		h.logger.Error("获取充值信息失败", zap.Uint64("user_id", user.ID), zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	if info.Address == "" {
		// 未分配充值地址
		tgpkg.EditMessage(ctx, b, chatID, messageID,
			"⚠️ 您的充值地址尚未分配，请联系客服。")
		return
	}

	// 构建充值页面文本
	text := fmt.Sprintf(
		"您正在使用 TRC20 付款\n\n"+
			"<b>付款金额：</b>\n"+
			"至少 1 U\n\n"+
			"<b>支持货币: TRX, USDT</b>\n\n"+
			"<b>收款地址(TRC20) :</b>\n"+
			"<code>%s</code>\n\n"+
			"点击复制钱包地址，可重复充值!\n"+
			"上面地址和二维码不一致，请不要付款!\n\n"+
			"<b>提示：</b>\n"+
			"- 对上述地址充值后, 经过3次网络确认, 充值成功!\n"+
			"- 请耐心等待, 充值成功后 Bot 会通知您!",
		info.Address,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.BackButton("Menu--main")).
		Build()

	// 先删除原消息 (因为需要发送图片消息)
	go func() {
		tgpkg.DeleteMessage(context.Background(), b, chatID, messageID)
	}()

	// 如果有二维码图片，发送图片消息
	if len(info.QRCode) > 0 {
		tgpkg.SendPhotoBytes(ctx, b, chatID, info.QRCode, "deposit_qrcode.png", text, keyboard)
	} else {
		// 没有二维码，发送纯文本
		tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
	}
}

// SendDepositNotification 发送充值到账通知 (由链上监控调用)
func (h *DepositHandler) SendDepositNotification(ctx context.Context, b *bot.Bot, telegramID int64, notification *app.DepositNotification) {
	text := fmt.Sprintf(
		"充值成功!\n\n"+
			"金额: %s %s\n"+
			"交易哈希: <code>%s</code>\n"+
			"当前余额: %s %s",
		notification.Amount.String(), notification.Currency,
		notification.TxHash,
		notification.NewBalance.String(), notification.Currency,
	)

	tgpkg.SendMessage(ctx, b, telegramID, text)
}
