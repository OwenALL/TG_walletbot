package handler

import (
	"context"
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

// TransferHandler 转账模块 Bot Handler
type TransferHandler struct {
	transferUC     *app.TransferUseCase
	userUC         *app.UserUseCase
	pinHandler     *PINHandler
	receiveHandler *ReceiveHandler
	sessionSvc     *tgpkg.SessionService
	logger         *zap.Logger
}

// NewTransferHandler 创建转账处理器
func NewTransferHandler(
	transferUC *app.TransferUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	receiveHandler *ReceiveHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *TransferHandler {
	return &TransferHandler{
		transferUC:     transferUC,
		userUC:         userUC,
		pinHandler:     pinHandler,
		receiveHandler: receiveHandler,
		sessionSvc:     sessionSvc,
		logger:         logger,
	}
}

// HandleCallback 处理 Transfer-- 回调
func (h *TransferHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID
	telegramID := query.From.ID

	// 立即响应回调
	tgpkg.AnswerCallback(ctx, b, query.ID)

	_, action := telegram.ParseCallbackData(query.Data)

	switch action {
	case "send_dashboard":
		h.showSendDashboard(ctx, b, chatID, messageID, telegramID)
	case "receive_dashboard":
		h.receiveHandler.ShowReceiveDashboard(ctx, b, chatID, messageID, telegramID)
	case "cancel":
		h.handleCancel(ctx, b, chatID, telegramID)
	default:
		// 处理动态回调 (如选择币种)
		h.handleDynamicCallback(ctx, b, chatID, messageID, telegramID, action)
	}
}

// showSendDashboard 显示转账主页面 (编辑原消息)
func (h *TransferHandler) showSendDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	// 设置会话状态
	_ = h.sessionSvc.SetStep(ctx, telegramID, "Transfer", "select_recipient")

	text := "请使用以下方法选择一个收款人：\n\n" +
		"- 发送收款人的 <code>@username</code>\n" +
		"- 转发一条收款人的消息到这里\n" +
		"- 发送收款人的【钱包页面】->【账户ID】"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.SwitchInlineCurrentChatButton("选择收款人", "")).
		Row(tgpkg.Button("取消", "Transfer--cancel")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// HandleTextMessage 处理转账模块的文本消息输入
// 返回 true 表示已处理
func (h *TransferHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Transfer" {
		return false
	}

	switch step {
	case "select_recipient":
		h.handleSelectRecipient(ctx, b, chatID, telegramID, update)
	case "input_amount":
		h.handleInputAmount(ctx, b, chatID, telegramID, update.Message.Text)
	case "pin_verified":
		h.executeTransfer(ctx, b, chatID, telegramID)
	default:
		return false
	}

	return true
}

// HandleForwardedMessage 处理转发的消息 (用于选择收款人)
func (h *TransferHandler) HandleForwardedMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Transfer" || step != "select_recipient" {
		return false
	}

	// 检查是否为转发消息
	if update.Message.ForwardOrigin == nil {
		return false
	}

	chatID := update.Message.Chat.ID

	// 尝试从转发来源获取用户
	var forwardedUserID int64
	if origin := update.Message.ForwardOrigin.MessageOriginUser; origin != nil {
		forwardedUserID = origin.SenderUser.ID
	} else {
		tgpkg.SendMessage(ctx, b, chatID, "无法识别转发消息的发送者，请尝试其他方式选择收款人。")
		return true
	}

	if forwardedUserID == 0 {
		tgpkg.SendMessage(ctx, b, chatID, "无法获取发送者信息，对方可能设置了隐私保护。")
		return true
	}

	h.confirmRecipient(ctx, b, chatID, telegramID, forwardedUserID)
	return true
}

// handleSelectRecipient 处理收款人选择 (文本输入)
func (h *TransferHandler) handleSelectRecipient(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, update *models.Update) {
	text := strings.TrimSpace(update.Message.Text)

	// 检查是否为转发消息
	if update.Message.ForwardOrigin != nil {
		h.HandleForwardedMessage(ctx, b, update)
		return
	}

	if text == "" {
		return
	}

	// 方式1: @username
	if strings.HasPrefix(text, "@") {
		recipient, err := h.transferUC.FindRecipientByUsername(ctx, text)
		if err != nil {
			h.logger.Error("查找收款人失败", zap.Error(err))
			tgpkg.SendMessage(ctx, b, chatID, "查找用户失败，请稍后重试。")
			return
		}
		if recipient == nil {
			tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
			return
		}
		h.confirmRecipient(ctx, b, chatID, telegramID, recipient.TelegramID)
		return
	}

	// 方式2: 数字 ID (尝试作为 Telegram ID)
	if _, err := strconv.ParseInt(text, 10, 64); err == nil {
		recipient, err := h.transferUC.FindRecipientByID(ctx, text)
		if err != nil {
			h.logger.Error("查找收款人失败", zap.Error(err))
			tgpkg.SendMessage(ctx, b, chatID, "查找用户失败，请稍后重试。")
			return
		}
		if recipient == nil {
			tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
			return
		}
		h.confirmRecipient(ctx, b, chatID, telegramID, recipient.TelegramID)
		return
	}

	// 无法识别
	tgpkg.SendMessage(ctx, b, chatID, "无法识别输入内容，请发送 <code>@username</code> 或用户 ID。")
}

// confirmRecipient 确认收款人并进入输入金额步骤
func (h *TransferHandler) confirmRecipient(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, recipientTelegramID int64) {
	// 不能转给自己
	if telegramID == recipientTelegramID {
		tgpkg.SendMessage(ctx, b, chatID, "不能向自己转账。")
		return
	}

	// 查找收款人信息
	recipient, err := h.transferUC.FindRecipientByTelegramID(ctx, recipientTelegramID)
	if err != nil || recipient == nil {
		tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
		return
	}

	// 保存收款人信息到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Transfer", "input_amount", map[string]interface{}{
		"to_user_id":   recipient.ID,
		"to_tg_id":     recipient.TelegramID,
		"to_name":      recipient.DisplayName(),
		"to_username":  recipient.Username,
	})

	text := fmt.Sprintf(
		"收款人: <a href=\"tg://user?id=%d\">%s</a>\n"+
			"ID: <code>%d</code>\n\n"+
			"请输入转账金额 (USDT)",
		recipient.TelegramID, recipient.DisplayName(),
		recipient.TelegramID,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("取消", "Transfer--cancel")).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// handleInputAmount 处理金额输入
func (h *TransferHandler) handleInputAmount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	amount, err := decimal.NewFromString(text)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "请输入有效的数字金额。")
		return
	}

	// 默认币种为 USDT，后续选择币种步骤可覆盖
	currency := entity.CurrencyUSDT

	// 验证金额
	if err := h.transferUC.ValidateTransferAmount(currency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, err.Error())
		return
	}

	// 保存金额到会话
	_ = h.sessionSvc.SetData(ctx, telegramID, "amount", text)

	// 进入选择币种步骤
	_ = h.sessionSvc.SetStep(ctx, telegramID, "Transfer", "select_currency")

	text2 := "请选择转账币种"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("USDT", "Transfer--currency_USDT"),
			tgpkg.Button("TRX", "Transfer--currency_TRX"),
			tgpkg.Button("CNY", "Transfer--currency_CNY"),
		).
		Row(tgpkg.Button("取消", "Transfer--cancel")).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text2, keyboard)
}

// handleDynamicCallback 处理动态回调
func (h *TransferHandler) handleDynamicCallback(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	switch {
	case strings.HasPrefix(action, "currency_"):
		h.handleSelectCurrency(ctx, b, chatID, messageID, telegramID, action)
	case action == "confirm":
		h.handleConfirmTransfer(ctx, b, chatID, messageID, telegramID)
	}
}

// handleSelectCurrency 处理币种选择
func (h *TransferHandler) handleSelectCurrency(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	currency := strings.TrimPrefix(action, "currency_")
	if currency != entity.CurrencyUSDT && currency != entity.CurrencyTRX && currency != entity.CurrencyCNY {
		return
	}

	// 获取金额
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 重新验证金额 (针对选中的币种)
	if err := h.transferUC.ValidateTransferAmount(currency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, err.Error())
		return
	}

	// 检查余额
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}
	if err := h.transferUC.CheckBalance(ctx, user.ID, currency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "余额不足，请先充值。")
		return
	}

	// 保存币种
	_ = h.sessionSvc.SetData(ctx, telegramID, "currency", currency)

	// 显示确认页面 (编辑原消息)
	toName := h.sessionSvc.GetDataString(ctx, telegramID, "to_name")

	text := fmt.Sprintf(
		"<b>转账确认</b>\n\n"+
			"收款人: %s\n"+
			"金额: %s %s\n"+
			"手续费: 0\n\n"+
			"确认转账请点击下方按钮",
		toName, amount.String(), currency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("确认转账", "Transfer--confirm"),
			tgpkg.Button("取消", "Transfer--cancel"),
		).
		Build()

	_ = h.sessionSvc.SetStep(ctx, telegramID, "Transfer", "confirm")
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleConfirmTransfer 处理确认转账 (发起 PIN 验证)
func (h *TransferHandler) handleConfirmTransfer(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 检查是否需要 PIN 验证
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, _ := decimal.NewFromString(amountStr)
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")

	needPIN, err := h.userUC.NeedPIN(ctx, user.ID, currency, amount)
	if err != nil {
		needPIN = true
	}

	if needPIN {
		// 发起 PIN 验证，验证通过后回到 Transfer/pin_verified
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Transfer", "pin_verified")
		return
	}

	// 免密，直接执行转账
	h.executeTransfer(ctx, b, chatID, telegramID)
}

// executeTransfer 执行转账 (PIN 验证通过后调用)
func (h *TransferHandler) executeTransfer(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 从会话中读取转账参数
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")
	toUserIDFloat := h.sessionSvc.GetDataFloat(ctx, telegramID, "to_user_id")
	toUserID := uint64(toUserIDFloat)
	toName := h.sessionSvc.GetDataString(ctx, telegramID, "to_name")
	toTgIDFloat := h.sessionSvc.GetDataFloat(ctx, telegramID, "to_tg_id")
	toTgID := int64(toTgIDFloat)

	// 执行转账
	result, err := h.transferUC.ExecuteTransfer(ctx, user.ID, toUserID, currency, amount)
	if err != nil {
		h.logger.Error("转账执行失败",
			zap.Uint64("from_user_id", user.ID),
			zap.Uint64("to_user_id", toUserID),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, "转账失败，请稍后重试。")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送成功消息给转出方
	senderText := fmt.Sprintf(
		"<b>转账成功!</b>\n\n"+
			"收款人: %s\n"+
			"金额: %s %s\n"+
			"余额: %s %s",
		toName,
		amount.String(), currency,
		result.SenderNewBalance.StringFixed(4), currency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("主菜单", "Menu--main")).
		Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, senderText, keyboard)

	// 异步通知收款方
	go func() {
		receiverText := fmt.Sprintf(
			"<b>收到转账!</b>\n\n"+
				"转账人: <a href=\"tg://user?id=%d\">%s</a>\n"+
				"金额: %s %s\n"+
				"余额: %s %s",
			telegramID, user.DisplayName(),
			amount.String(), currency,
			result.ReceiverNewBalance.StringFixed(4), currency,
		)
		tgpkg.SendMessage(context.Background(), b, toTgID, receiverText)
	}()
}

// handleCancel 取消转账
func (h *TransferHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendMessage(ctx, b, chatID, "操作已取消。")
		return
	}

	// 显示主菜单
	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "操作已取消。")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// RegisterTransferHandlers 注册转账相关的 handler 到 router
func RegisterTransferHandlers(router *telegram.Router, h *TransferHandler) {
	router.RegisterModule("Transfer", h.HandleCallback)
}
