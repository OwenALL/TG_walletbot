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

// ReceiveHandler 收钱模块 Bot Handler
type ReceiveHandler struct {
	transferUC *app.TransferUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewReceiveHandler 创建收钱处理器
func NewReceiveHandler(
	transferUC *app.TransferUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *ReceiveHandler {
	return &ReceiveHandler{
		transferUC: transferUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// ShowReceiveDashboard 显示收钱主页面 (编辑原消息)
// 由 TransferHandler 的回调路由调用
func (h *ReceiveHandler) ShowReceiveDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	// 设置会话状态
	_ = h.sessionSvc.SetStep(ctx, telegramID, "Receive", "input_amount")

	text := "您要转多少钱 (U)? 例: <code>8.88</code>\n\n" +
		"在下面的输入框中输入金额并发送。"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.SwitchInlineCurrentChatButton("选择付款人", "")).
		Row(tgpkg.Button("取消", "Receive--cancel")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// HandleCallback 处理 Receive-- 回调
func (h *ReceiveHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	case "cancel":
		h.handleCancel(ctx, b, chatID, messageID, telegramID)
	default:
		h.handleDynamicCallback(ctx, b, chatID, messageID, telegramID, action)
	}
}

// HandleTextMessage 处理收钱模块的文本消息输入
// 返回 true 表示已处理
func (h *ReceiveHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Receive" {
		return false
	}

	switch step {
	case "input_amount":
		h.handleInputAmount(ctx, b, chatID, telegramID, update.Message.Text)
	case "select_payer":
		h.handleSelectPayer(ctx, b, chatID, telegramID, update)
	case "execute_pay":
		// PIN 验证完成后执行付款
		h.executeReceivePayment(ctx, b, chatID, telegramID)
	default:
		return false
	}

	return true
}

// handleInputAmount 处理收款金额输入
func (h *ReceiveHandler) handleInputAmount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	amount, err := decimal.NewFromString(text)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "请输入有效的数字金额。")
		return
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.SendMessage(ctx, b, chatID, "金额必须大于 0。")
		return
	}

	minAmount := decimal.NewFromFloat(0.01)
	if amount.LessThan(minAmount) {
		tgpkg.SendMessage(ctx, b, chatID, "最低金额为 0.01 USDT。")
		return
	}

	// 保存金额并进入选择付款人步骤
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Receive", "select_payer", map[string]interface{}{
		"amount":   text,
		"currency": entity.CurrencyUSDT,
	})

	msg := fmt.Sprintf(
		"收款金额: <b>%s USDT</b>\n\n"+
			"请选择付款人：\n"+
			"- 发送付款人的 <code>@username</code>\n"+
			"- 转发一条付款人的消息到这里\n"+
			"- 发送付款人的用户 ID",
		text,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.SwitchInlineCurrentChatButton("选择付款人", "")).
		Row(tgpkg.Button("取消", "Receive--cancel")).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, msg, keyboard)
}

// handleSelectPayer 处理付款人选择
func (h *ReceiveHandler) handleSelectPayer(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, update *models.Update) {
	// 检查是否为转发消息
	if update.Message.ForwardOrigin != nil {
		var forwardedUserID int64
		if origin := update.Message.ForwardOrigin.MessageOriginUser; origin != nil {
			forwardedUserID = origin.SenderUser.ID
		} else {
			tgpkg.SendMessage(ctx, b, chatID, "无法识别转发消息的发送者。")
			return
		}
		if forwardedUserID == 0 {
			tgpkg.SendMessage(ctx, b, chatID, "无法获取发送者信息。")
			return
		}
		h.confirmPayerAndSendRequest(ctx, b, chatID, telegramID, forwardedUserID)
		return
	}

	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return
	}

	// @username
	if strings.HasPrefix(text, "@") {
		recipient, err := h.transferUC.FindRecipientByUsername(ctx, text)
		if err != nil || recipient == nil {
			tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
			return
		}
		h.confirmPayerAndSendRequest(ctx, b, chatID, telegramID, recipient.TelegramID)
		return
	}

	// 数字 ID
	if _, err := strconv.ParseInt(text, 10, 64); err == nil {
		recipient, err := h.transferUC.FindRecipientByID(ctx, text)
		if err != nil || recipient == nil {
			tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
			return
		}
		h.confirmPayerAndSendRequest(ctx, b, chatID, telegramID, recipient.TelegramID)
		return
	}

	tgpkg.SendMessage(ctx, b, chatID, "无法识别输入内容，请发送 <code>@username</code> 或用户 ID。")
}

// confirmPayerAndSendRequest 确认付款人并发送收款请求
func (h *ReceiveHandler) confirmPayerAndSendRequest(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, payerTelegramID int64) {
	// 不能向自己收款
	if telegramID == payerTelegramID {
		tgpkg.SendMessage(ctx, b, chatID, "不能向自己发送收款请求。")
		return
	}

	// 查找付款人
	payer, err := h.transferUC.FindRecipientByTelegramID(ctx, payerTelegramID)
	if err != nil || payer == nil {
		tgpkg.SendMessage(ctx, b, chatID, "未找到该用户，请确认对方已使用本 Bot。")
		return
	}

	// 获取收款人信息
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 获取金额和币种
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")
	if currency == "" {
		currency = entity.CurrencyUSDT
	}

	// 创建收款请求
	transfer, err := h.transferUC.CreateReceiveRequest(ctx, user.ID, payer.ID, currency, amount)
	if err != nil {
		h.logger.Error("创建收款请求失败", zap.Error(err))
		tgpkg.SendMessage(ctx, b, chatID, "创建收款请求失败，请稍后重试。")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除收款人会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 通知收款人: 请求已发送
	senderText := fmt.Sprintf(
		"<b>收款请求已发送!</b>\n\n"+
			"付款人: <a href=\"tg://user?id=%d\">%s</a>\n"+
			"金额: %s %s\n\n"+
			"等待对方确认...",
		payer.TelegramID, payer.DisplayName(),
		amount.String(), currency,
	)
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("主菜单", "Menu--main")).
		Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, senderText, keyboard)

	// 向付款人发送收款请求通知
	go func() {
		payerText := fmt.Sprintf(
			"<a href=\"tg://user?id=%d\">%s</a> 向您发起了一笔收款请求\n\n"+
				"金额: <b>%s %s</b>\n\n"+
				"请确认是否支付",
			user.TelegramID, user.DisplayName(),
			amount.String(), currency,
		)

		payerKeyboard := tgpkg.NewInlineKeyboard().
			Row(
				tgpkg.Button("确认支付", fmt.Sprintf("Receive--pay_%d", transfer.ID)),
				tgpkg.Button("拒绝", fmt.Sprintf("Receive--reject_%d", transfer.ID)),
			).
			Build()

		tgpkg.SendMessageWithKeyboard(context.Background(), b, payer.TelegramID, payerText, payerKeyboard)
	}()
}

// handleDynamicCallback 处理动态回调
func (h *ReceiveHandler) handleDynamicCallback(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	switch {
	case strings.HasPrefix(action, "pay_"):
		h.handlePayConfirm(ctx, b, chatID, messageID, telegramID, action)
	case strings.HasPrefix(action, "reject_"):
		h.handlePayReject(ctx, b, chatID, messageID, telegramID, action)
	}
}

// handlePayConfirm 处理付款人点击 [确认支付]
func (h *ReceiveHandler) handlePayConfirm(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	transferIDStr := strings.TrimPrefix(action, "pay_")
	transferID, err := strconv.ParseUint(transferIDStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 查询转账记录获取详情
	transfer, err := h.transferUC.GetTransferByID(ctx, transferID)
	if err != nil || transfer == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "该收款请求不存在或已处理。")
		return
	}
	if transfer.Status != entity.TransferStatusPending {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "该收款请求已处理。")
		return
	}

	// 验证付款人身份
	user := telegram.UserFromContext(ctx)
	if user == nil || user.ID != transfer.FromUserID {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "您不是该收款请求的付款人。")
		return
	}

	// 保存转账 ID 到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Receive", "execute_pay", map[string]interface{}{
		"pay_transfer_id": transferID,
	})

	// 检查是否需要 PIN
	needPIN, pinErr := h.userUC.NeedPIN(ctx, user.ID, transfer.Currency, transfer.Amount)
	if pinErr != nil {
		needPIN = true
	}

	if needPIN {
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Receive", "execute_pay")
		return
	}

	// 编辑请求消息为处理中状态
	tgpkg.EditMessage(ctx, b, chatID, messageID, "⏳ 处理中...")
	h.executeReceivePayment(ctx, b, chatID, telegramID)
}

// handlePayReject 处理付款人点击 [拒绝]
func (h *ReceiveHandler) handlePayReject(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	transferIDStr := strings.TrimPrefix(action, "reject_")
	transferID, err := strconv.ParseUint(transferIDStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 拒绝收款请求
	if err := h.transferUC.RejectReceiveRequest(ctx, transferID); err != nil {
		h.logger.Error("拒绝收款请求失败", zap.Uint64("transfer_id", transferID), zap.Error(err))
	}

	tgpkg.EditMessage(ctx, b, chatID, messageID, "已拒绝该收款请求。")
}

// executeReceivePayment 执行收款请求的付款 (PIN 验证通过后)
func (h *ReceiveHandler) executeReceivePayment(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	transferIDFloat := h.sessionSvc.GetDataFloat(ctx, telegramID, "pay_transfer_id")
	transferID := uint64(transferIDFloat)
	if transferID == 0 {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 执行收款请求的付款
	result, err := h.transferUC.ExecuteReceivePayment(ctx, transferID)
	if err != nil {
		h.logger.Error("收款请求付款失败",
			zap.Uint64("transfer_id", transferID),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, "支付失败，请稍后重试。")
		return
	}

	// 查询转账记录获取详情
	transfer, _ := h.transferUC.GetTransferByID(ctx, transferID)
	if transfer == nil {
		tgpkg.SendMessage(ctx, b, chatID, "支付成功!")
		return
	}

	// 查询收款人信息
	receiver, _ := h.transferUC.GetUserByID(ctx, transfer.ToUserID)
	receiverName := "用户"
	var receiverTgID int64
	if receiver != nil {
		receiverName = receiver.DisplayName()
		receiverTgID = receiver.TelegramID
	}

	// 发送成功消息给付款人
	payerText := fmt.Sprintf(
		"<b>支付成功!</b>\n\n"+
			"收款人: %s\n"+
			"金额: %s %s\n"+
			"余额: %s %s",
		receiverName,
		transfer.Amount.String(), transfer.Currency,
		result.SenderNewBalance.StringFixed(4), transfer.Currency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("主菜单", "Menu--main")).
		Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, payerText, keyboard)

	// 异步通知收款人
	if receiverTgID != 0 {
		go func() {
			receiverText := fmt.Sprintf(
				"<b>收到付款!</b>\n\n"+
					"付款人: <a href=\"tg://user?id=%d\">%s</a>\n"+
					"金额: %s %s\n"+
					"余额: %s %s",
				telegramID, user.DisplayName(),
				transfer.Amount.String(), transfer.Currency,
				result.ReceiverNewBalance.StringFixed(4), transfer.Currency,
			)
			tgpkg.SendMessage(context.Background(), b, receiverTgID, receiverText)
		}()
	}
}

// handleCancel 取消收钱 (编辑原消息回到主菜单)
func (h *ReceiveHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作已取消。")
		return
	}

	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作已取消。")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// RegisterReceiveHandlers 注册收钱相关的 handler 到 router
func RegisterReceiveHandlers(router *telegram.Router, h *ReceiveHandler) {
	router.RegisterModule("Receive", h.HandleCallback)
}
