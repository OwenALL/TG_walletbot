package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// ExchangeHandler 闪兑模块 Bot Handler
type ExchangeHandler struct {
	exchangeUC *app.ExchangeUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewExchangeHandler 创建闪兑处理器
func NewExchangeHandler(
	exchangeUC *app.ExchangeUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *ExchangeHandler {
	return &ExchangeHandler{
		exchangeUC: exchangeUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理 Exchange-- 回调
func (h *ExchangeHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	case "dashboard":
		h.showDashboard(ctx, b, chatID, messageID)
	case "usdt_to_cny":
		h.startExchange(ctx, b, chatID, messageID, telegramID, action)
	case "cny_to_usdt":
		h.startExchange(ctx, b, chatID, messageID, telegramID, action)
	case "usdt_to_ton":
		h.startExchange(ctx, b, chatID, messageID, telegramID, action)
	case "usdt_to_trx":
		h.startExchange(ctx, b, chatID, messageID, telegramID, action)
	case "trx_to_usdt":
		h.startExchange(ctx, b, chatID, messageID, telegramID, action)
	case "confirm":
		h.handleConfirmExchange(ctx, b, chatID, messageID, telegramID)
	case "cancel":
		h.handleCancel(ctx, b, chatID, messageID, telegramID)
	default:
		h.logger.Warn("未知闪兑回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理闪兑模块的文本消息输入
// 返回 true 表示已处理
func (h *ExchangeHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Exchange" {
		return false
	}

	switch step {
	case "input_amount":
		h.handleInputAmount(ctx, b, chatID, telegramID, update.Message.Text)
	case "pin_verified":
		h.executeExchange(ctx, b, chatID, telegramID)
	default:
		return false
	}

	return true
}

// showDashboard 显示闪兑菜单 (EditMessageText)
func (h *ExchangeHandler) showDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	// 获取所有汇率
	rates, err := h.exchangeUC.GetAllRatesDisplay(ctx)
	if err != nil {
		h.logger.Error("获取汇率失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := "<b>闪兑</b>\n\n"
	if !rates.USDTtoTRX.IsZero() {
		text += fmt.Sprintf("<b>100 USDT = %s TRX</b>\n", rates.USDTtoTRX.StringFixed(2))
	}
	if !rates.USDTtoCNY.IsZero() {
		text += fmt.Sprintf("<b>100 USDT = %s CNY</b>\n", rates.USDTtoCNY.StringFixed(2))
	}
	if !rates.USDTtoTON.IsZero() {
		text += fmt.Sprintf("<b>100 USDT = %s TON</b>\n", rates.USDTtoTON.StringFixed(2))
	}
	if !rates.TRXtoUSDT.IsZero() {
		text += fmt.Sprintf("<b>1000 TRX = %s USDT</b>\n", rates.TRXtoUSDT.StringFixed(2))
	}
	text += "────────────────────"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("USDT => CNY", "Exchange--usdt_to_cny"),
			tgpkg.Button("CNY => USDT", "Exchange--cny_to_usdt"),
		).
		Row(
			tgpkg.Button("USDT => TON", "Exchange--usdt_to_ton"),
		).
		Row(
			tgpkg.Button("USDT => TRX", "Exchange--usdt_to_trx"),
			tgpkg.Button("TRX => USDT", "Exchange--trx_to_usdt"),
		).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// startExchange 开始兑换流程 (选择方向后输入金额)
func (h *ExchangeHandler) startExchange(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	// 查找兑换方向
	direction := h.exchangeUC.FindDirection(action)
	if direction == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取用户余额
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	balance, err := h.exchangeUC.GetBalance(ctx, user.ID, direction.FromCurrency)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 保存兑换方向到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Exchange", "input_amount", map[string]interface{}{
		"from_currency":  direction.FromCurrency,
		"to_currency":    direction.ToCurrency,
		"min_amount":     direction.MinAmount.String(),
		"min_amount_text": direction.MinAmountText,
	})

	text := fmt.Sprintf(
		"请输入要闪兑的金额,最小为 %s\n\n"+
			"当前余额 : <code>%s</code> %s",
		direction.MinAmountText,
		balance.StringFixed(4), direction.FromCurrency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.BackButton("Exchange--dashboard")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleInputAmount 处理金额输入
func (h *ExchangeHandler) handleInputAmount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	amount, err := decimal.NewFromString(text)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "请输入有效的数字金额。")
		return
	}

	// 获取兑换方向信息
	fromCurrency := h.sessionSvc.GetDataString(ctx, telegramID, "from_currency")
	toCurrency := h.sessionSvc.GetDataString(ctx, telegramID, "to_currency")
	minAmountStr := h.sessionSvc.GetDataString(ctx, telegramID, "min_amount")
	minAmountText := h.sessionSvc.GetDataString(ctx, telegramID, "min_amount_text")

	minAmount, _ := decimal.NewFromString(minAmountStr)
	if amount.LessThan(minAmount) {
		tgpkg.SendMessage(ctx, b, chatID, fmt.Sprintf("最小兑换金额为 %s", minAmountText))
		return
	}

	// 获取用户信息
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 检查余额
	if err := h.exchangeUC.CheckBalance(ctx, user.ID, fromCurrency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "余额不足，请先充值。")
		return
	}

	// 获取汇率
	pair, err := h.exchangeUC.GetExchangePair(ctx, fromCurrency, toCurrency)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "当前兑换暂不可用，请稍后重试。")
		return
	}

	// 计算兑入金额
	toAmount, fee := h.exchangeUC.CalculateExchange(pair, amount)

	// 保存数据到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Exchange", "confirm", map[string]interface{}{
		"amount":      text,
		"to_amount":   toAmount.String(),
		"rate":        pair.Rate.String(),
		"fee":         fee.String(),
	})

	// 显示确认页面
	confirmText := fmt.Sprintf(
		"<b>闪兑确认</b>\n\n"+
			"兑出: %s %s\n"+
			"兑入: %s %s\n"+
			"汇率: 1 %s = %s %s\n"+
			"手续费: %s %s\n\n"+
			"确认兑换请点击下方按钮",
		amount.String(), fromCurrency,
		toAmount.StringFixed(4), toCurrency,
		fromCurrency, pair.Rate.StringFixed(4), toCurrency,
		fee.String(), fromCurrency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("确认", "Exchange--confirm"),
			tgpkg.Button("取消", "Exchange--cancel"),
		).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, confirmText, keyboard)
}

// handleConfirmExchange 处理确认兑换 (发起 PIN 验证)
func (h *ExchangeHandler) handleConfirmExchange(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取金额和币种
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, _ := decimal.NewFromString(amountStr)
	fromCurrency := h.sessionSvc.GetDataString(ctx, telegramID, "from_currency")

	// 检查是否需要 PIN
	needPIN, err := h.userUC.NeedPIN(ctx, user.ID, fromCurrency, amount)
	if err != nil {
		needPIN = true
	}

	if needPIN {
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Exchange", "pin_verified")
		return
	}

	// 编辑确认消息为处理中状态
	tgpkg.EditMessage(ctx, b, chatID, messageID, "⏳ 处理中...")
	h.executeExchange(ctx, b, chatID, telegramID)
}

// executeExchange 执行兑换 (PIN 验证通过后)
func (h *ExchangeHandler) executeExchange(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 从会话中读取兑换参数
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	fromCurrency := h.sessionSvc.GetDataString(ctx, telegramID, "from_currency")
	toCurrency := h.sessionSvc.GetDataString(ctx, telegramID, "to_currency")

	// 重新获取汇率 (防止汇率变动)
	pair, err := h.exchangeUC.GetExchangePair(ctx, fromCurrency, toCurrency)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "当前兑换暂不可用，请稍后重试。")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 执行兑换
	result, err := h.exchangeUC.ExecuteExchange(ctx, user.ID, fromCurrency, toCurrency, amount, pair)
	if err != nil {
		h.logger.Error("闪兑执行失败",
			zap.Uint64("user_id", user.ID),
			zap.String("pair", fromCurrency+"/"+toCurrency),
			zap.Error(err),
		)
		errMsg := "兑换失败，请稍后重试。"
		if strings.Contains(err.Error(), "余额不足") {
			errMsg = "余额不足，请先充值。"
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送成功消息
	successText := fmt.Sprintf(
		"<b>闪兑成功!</b>\n\n"+
			"兑出: %s %s\n"+
			"兑入: %s %s\n"+
			"汇率: 1 %s = %s %s",
		result.Exchange.FromAmount.String(), fromCurrency,
		result.Exchange.ToAmount.StringFixed(4), toCurrency,
		fromCurrency, result.Exchange.Rate.StringFixed(4), toCurrency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("继续兑换", "Exchange--dashboard")).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, successText, keyboard)
}

// handleCancel 取消兑换，返回闪兑菜单
func (h *ExchangeHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)
	h.showDashboard(ctx, b, chatID, messageID)
}

// RegisterExchangeHandlers 注册闪兑相关的 handler 到 router
func RegisterExchangeHandlers(router *telegram.Router, h *ExchangeHandler) {
	router.RegisterModule("Exchange", h.HandleCallback)
}
