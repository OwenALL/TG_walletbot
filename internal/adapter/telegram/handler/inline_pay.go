package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// InlinePayHandler 处理 Inline 收款的深度链接支付流程
// 用户点击 "🐲 立即支付" URL 按钮后跳转到 Bot 私聊，完成 PIN 验证并执行转账
type InlinePayHandler struct {
	transferUC *app.TransferUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewInlinePayHandler 创建 Inline 支付处理器
func NewInlinePayHandler(
	transferUC *app.TransferUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *InlinePayHandler {
	return &InlinePayHandler{
		transferUC: transferUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// RegisterInlinePayHandlers 注册 InlinePay 回调模块
func RegisterInlinePayHandlers(router *telegram.Router, h *InlinePayHandler) {
	router.RegisterModule("InlinePay", h.HandleCallback)
}

// =====================================================
// 深度链接编解码
// =====================================================

// EncodePayLink 编码收款支付深度链接参数
// 格式: pay--<base64url(requesterTgID amount currency)>
// 符合 Telegram deep link 限制: 最大 64 字符，仅 A-Za-z0-9_-
func EncodePayLink(requesterTgID int64, amount decimal.Decimal, currency string) string {
	payload := fmt.Sprintf("%d %s %s", requesterTgID, amount.String(), currency)
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return "pay--" + encoded
}

// decodePayLink 解码收款支付深度链接参数
// 返回: requesterTgID, amount, currency, error
func decodePayLink(param string) (int64, decimal.Decimal, string, error) {
	if !strings.HasPrefix(param, "pay--") {
		return 0, decimal.Zero, "", fmt.Errorf("无效的支付链接前缀")
	}

	encoded := strings.TrimPrefix(param, "pay--")
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, decimal.Zero, "", fmt.Errorf("解码失败: %w", err)
	}

	// 格式: "requesterTgID amount currency"
	parts := strings.SplitN(string(decoded), " ", 3)
	if len(parts) != 3 {
		return 0, decimal.Zero, "", fmt.Errorf("数据格式错误，需要 3 段")
	}

	requesterTgID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, decimal.Zero, "", fmt.Errorf("无效的收款人 ID: %w", err)
	}

	amount, err := decimal.NewFromString(parts[1])
	if err != nil {
		return 0, decimal.Zero, "", fmt.Errorf("无效的金额: %w", err)
	}

	currency := parts[2]
	if currency != entity.CurrencyUSDT && currency != entity.CurrencyTRX && currency != entity.CurrencyCNY {
		return 0, decimal.Zero, "", fmt.Errorf("不支持的币种: %s", currency)
	}

	return requesterTgID, amount, currency, nil
}

// =====================================================
// 深度链接入口 (由 StartHandler 调用)
// =====================================================

// HandleDeepLink 处理深度链接支付请求
// 由 StartHandler 检测到 /start pay--xxx 参数后调用
func (h *InlinePayHandler) HandleDeepLink(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, user *entity.User, param string) {
	// 解码深度链接参数
	requesterTgID, amount, currency, err := decodePayLink(param)
	if err != nil {
		h.logger.Warn("解码 Inline 支付链接失败",
			zap.String("param", param),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 无效的支付链接")
		return
	}

	shortCurrency := currencyShort(currency)

	// 不能支付给自己
	if telegramID == requesterTgID {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 不能支付给自己")
		return
	}

	// 查找收款人
	requester, err := h.transferUC.FindRecipientByTelegramID(ctx, requesterTgID)
	if err != nil || requester == nil {
		h.logger.Warn("Inline 支付收款人不存在",
			zap.Int64("requester_tg_id", requesterTgID),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 收款人不存在")
		return
	}

	// 检查余额
	if err := h.transferUC.CheckBalance(ctx, user.ID, currency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 余额不足，无法完成支付")
		return
	}

	// 保存支付参数到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "InlinePay", "confirm", map[string]interface{}{
		"pay_requester_tg_id": strconv.FormatInt(requesterTgID, 10),
		"pay_requester_name":  requester.DisplayName(),
		"pay_amount":          amount.String(),
		"pay_currency":        currency,
	})

	// 显示确认支付消息
	requesterName := requester.DisplayName()
	text := fmt.Sprintf(
		"💸 <b>确认支付</b>\n\n"+
			"收款人: %s\n"+
			"金额: <b>%s %s</b>\n\n"+
			"请确认是否支付？",
		requesterName, amount.String(), shortCurrency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("✅ 确认支付", "InlinePay--confirm"),
			tgpkg.Button("❌ 取消", "InlinePay--cancel"),
		).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// =====================================================
// 回调处理
// =====================================================

// HandleCallback 处理 InlinePay-- 回调
func (h *InlinePayHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID
	telegramID := query.From.ID

	// 立即响应回调，避免转圈
	tgpkg.AnswerCallback(ctx, b, query.ID)

	_, action := telegram.ParseCallbackData(query.Data)

	switch action {
	case "confirm":
		h.handleConfirm(ctx, b, chatID, messageID, telegramID)
	case "pin_verified":
		h.executePayment(ctx, b, chatID, telegramID)
	case "cancel":
		h.handleCancel(ctx, b, chatID, messageID, telegramID)
	default:
		h.logger.Warn("未知 InlinePay 回调", zap.String("action", action))
	}
}

// handleConfirm 处理确认支付按钮
func (h *InlinePayHandler) handleConfirm(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "⚠️ 用户信息获取失败，请重试")
		return
	}

	// 读取会话中的支付参数
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "pay_amount")
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "pay_currency")

	if amountStr == "" || currency == "" {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "⚠️ 支付信息已过期，请重新点击支付按钮")
		return
	}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "⚠️ 金额数据异常")
		return
	}

	// 未设置 PIN，先设置
	if !user.HasPIN() {
		h.pinHandler.StartPINSetup(ctx, b, chatID, messageID, telegramID, "InlinePay", "pin_verified")
		return
	}

	// 判断是否需要 PIN 验证
	needPIN, err := h.userUC.NeedPIN(ctx, user.ID, currency, amount)
	if err != nil {
		needPIN = true
	}

	if needPIN {
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "InlinePay", "pin_verified")
		return
	}

	// 不需要 PIN，直接执行支付
	tgpkg.EditMessage(ctx, b, chatID, messageID, "⏳ 处理中...")
	h.executePayment(ctx, b, chatID, telegramID)
}

// executePayment 执行 Inline 收款的支付转账
func (h *InlinePayHandler) executePayment(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	// 读取会话中的支付参数
	requesterTgIDStr := h.sessionSvc.GetDataString(ctx, telegramID, "pay_requester_tg_id")
	requesterName := h.sessionSvc.GetDataString(ctx, telegramID, "pay_requester_name")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "pay_amount")
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "pay_currency")

	if requesterTgIDStr == "" || amountStr == "" || currency == "" {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 支付信息已过期，请重新点击支付按钮")
		return
	}

	requesterTgID, err := strconv.ParseInt(requesterTgIDStr, 10, 64)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 数据异常")
		return
	}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 金额数据异常")
		return
	}

	// 获取付款人
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 用户信息获取失败")
		return
	}

	// 获取收款人
	requester, err := h.transferUC.FindRecipientByTelegramID(ctx, requesterTgID)
	if err != nil || requester == nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 收款人不存在")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 执行转账 (付款人 → 收款人)
	result, err := h.transferUC.ExecuteTransfer(ctx, user.ID, requester.ID, currency, amount)
	if err != nil {
		errMsg := "⚠️ 支付失败，请稍后重试"
		if strings.Contains(err.Error(), "余额不足") || strings.Contains(err.Error(), "insufficient") {
			errMsg = "⚠️ 余额不足"
		}
		h.logger.Error("Inline 收款支付失败",
			zap.Int64("payer_tg_id", telegramID),
			zap.Int64("requester_tg_id", requesterTgID),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	shortCurrency := currencyShort(currency)

	// 支付成功提示
	text := fmt.Sprintf(
		"✅ <b>支付成功</b>\n\n"+
			"收款人: %s\n"+
			"金额: %s %s\n"+
			"余额: %s %s",
		requesterName,
		amount.String(), shortCurrency,
		result.SenderNewBalance.StringFixed(4), shortCurrency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)

	// 异步通知收款人
	go func() {
		payerName := user.DisplayName()
		notifyText := fmt.Sprintf(
			"✅ 收到付款\n\n"+
				"付款人: %s\n"+
				"金额: %s %s\n"+
				"余额: %s %s",
			payerName,
			amount.String(), shortCurrency,
			result.ReceiverNewBalance.StringFixed(4), shortCurrency,
		)
		tgpkg.SendMessage(context.Background(), b, requesterTgID, notifyText)
	}()
}

// handleCancel 处理取消支付
func (h *InlinePayHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)

	text := "已取消支付"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.MainMenuButton()).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}
