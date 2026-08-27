package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/app"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// WithdrawHandler 提币相关处理器
type WithdrawHandler struct {
	withdrawUC *app.WithdrawalUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewWithdrawHandler 创建提币处理器
func NewWithdrawHandler(
	withdrawUC *app.WithdrawalUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *WithdrawHandler {
	return &WithdrawHandler{
		withdrawUC: withdrawUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理提币相关的回调
// actions: withdraw_menu, withdraw_usdt, withdraw_trx, withdraw_cny, withdraw_confirm, withdraw_cancel
func (h *WithdrawHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
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
	case "withdraw_menu":
		h.showWithdrawMenu(ctx, b, chatID, messageID)
	case "withdraw_usdt":
		h.startWithdraw(ctx, b, chatID, messageID, telegramID, "USDT")
	case "withdraw_trx":
		h.startWithdraw(ctx, b, chatID, messageID, telegramID, "TRX")
	case "withdraw_cny":
		h.showCNYPlaceholder(ctx, b, chatID, messageID)
	case "withdraw_confirm":
		h.handleConfirm(ctx, b, chatID, messageID, telegramID)
	case "withdraw_cancel":
		h.handleCancel(ctx, b, chatID, messageID, telegramID)
	default:
		h.logger.Warn("未知提币回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理提币相关的文本输入
// 返回 true 表示已处理，false 表示不是提币流程
func (h *WithdrawHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Wallet" {
		return false
	}

	switch step {
	case "withdraw_amount":
		h.handleAmountInput(ctx, b, chatID, telegramID, text)
		return true
	case "withdraw_address":
		h.handleAddressInput(ctx, b, chatID, telegramID, text)
		return true
	default:
		return false
	}
}

// showWithdrawMenu 显示提币菜单
func (h *WithdrawHandler) showWithdrawMenu(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	text := "请选择您的提现方式\n\n<b>注意: 为了避免风险! 请勿直接提币到交易所!</b>"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("USDT(TRC20)", "Wallet--withdraw_usdt")).
		Row(tgpkg.Button("TRX", "Wallet--withdraw_trx")).
		Row(tgpkg.Button("CNY 人民币", "Wallet--withdraw_cny")).
		Row(tgpkg.BackButton("Menu--main")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// startWithdraw 开始提币流程 - 输入金额
// PIN 保护已由 WalletCallbackHandler 在入口处统一检查
func (h *WithdrawHandler) startWithdraw(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, currency string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取余额
	balance, err := h.withdrawUC.GetBalance(ctx, user.ID, currency)
	if err != nil {
		h.logger.Error("获取余额失败", zap.Uint64("user_id", user.ID), zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 设置会话状态
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Wallet", "withdraw_amount", map[string]interface{}{
		"withdraw_currency": currency,
	})

	text := fmt.Sprintf(
		"<b>请回复要提现的金额</b>\n\n"+
			"当前余额 : <code>%s</code> %s",
		balance.StringFixed(4), currency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.BackButton("Wallet--withdraw_cancel")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleAmountInput 处理金额输入
func (h *WithdrawHandler) handleAmountInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	currency := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_currency")
	if currency == "" {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 解析金额
	amount, err := decimal.NewFromString(text)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 请输入有效的数字金额")
		return
	}

	// 验证金额
	if err := h.withdrawUC.ValidateAmount(ctx, user.ID, currency, amount); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ "+err.Error())
		return
	}

	// 保存金额，进入地址输入步骤
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Wallet", "withdraw_address", map[string]interface{}{
		"withdraw_amount": amount.String(),
	})

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.CancelButton()).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, "请输入 TRC20 提币地址", keyboard)
}

// handleAddressInput 处理地址输入
func (h *WithdrawHandler) handleAddressInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 验证地址格式
	address := strings.TrimSpace(text)
	if !h.withdrawUC.ValidateAddress(address) {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 地址格式不正确，请输入有效的 TRC20 地址 (T 开头，34 位字符)")
		return
	}

	// 获取会话中的金额和币种
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_currency")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_amount")
	amount, _ := decimal.NewFromString(amountStr)

	if currency == "" || amount.IsZero() {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 构建确认信息
	confirmInfo, err := h.withdrawUC.BuildConfirmInfo(ctx, user.ID, currency, amount, address)
	if err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ "+err.Error())
		return
	}

	// 保存地址到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Wallet", "withdraw_confirm", map[string]interface{}{
		"withdraw_address":  address,
		"withdraw_need_pin": confirmInfo.NeedPIN,
	})

	// 显示确认页面
	confirmText := fmt.Sprintf(
		"<b>提币确认</b>\n\n"+
			"币种: %s (TRC20)\n"+
			"金额: %s %s\n"+
			"手续费: %s %s\n"+
			"实际到账: %s %s\n"+
			"提币地址: <code>%s</code>",
		confirmInfo.Currency,
		confirmInfo.Amount.String(), confirmInfo.Currency,
		confirmInfo.Fee.String(), confirmInfo.Currency,
		confirmInfo.ActualAmount.String(), confirmInfo.Currency,
		confirmInfo.ToAddress,
	)

	// 统一使用确认按钮，PIN 验证在 handleConfirm 中通过内联键盘完成
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("确认提币", "Wallet--withdraw_confirm")).
		Row(tgpkg.CancelButton()).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, confirmText, keyboard)
}

// handleConfirm 处理确认提币
// 需要 PIN 时弹出内联数字键盘验证，不需要 PIN 时直接执行
func (h *WithdrawHandler) handleConfirm(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 验证会话状态
	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Wallet" || step != "withdraw_confirm" {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 检查是否需要 PIN 验证
	needPINStr := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_need_pin")
	needPIN := needPINStr == "true"

	if needPIN {
		// 弹出 PIN 内联数字键盘，验证通过后回到 Wallet/withdraw_confirm 重新确认
		// 注意: PIN 验证通过后会恢复 session 到 Wallet/withdraw_confirm
		// 此时 withdraw_need_pin 改为已验证状态，避免循环
		_ = h.sessionSvc.SetData(ctx, telegramID, "withdraw_need_pin", "verified")
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Wallet", "withdraw_confirm")
		return
	}

	// 编辑消息为处理中
	tgpkg.EditMessage(ctx, b, chatID, messageID, "处理中...")

	h.executeWithdraw(ctx, b, chatID, telegramID, user.ID)
}

// executeWithdraw 执行提币
func (h *WithdrawHandler) executeWithdraw(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, userID uint64) {
	// 从会话获取参数
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_currency")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_amount")
	address := h.sessionSvc.GetDataString(ctx, telegramID, "withdraw_address")

	amount, _ := decimal.NewFromString(amountStr)

	if currency == "" || amount.IsZero() || address == "" {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 提交提币
	withdrawal, err := h.withdrawUC.SubmitWithdrawal(ctx, userID, currency, address, amount)
	if err != nil {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ "+err.Error())
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送提币成功消息
	text := fmt.Sprintf(
		"提币申请已提交!\n\n"+
			"金额: %s %s\n"+
			"手续费: %s %s\n"+
			"预计到账: 10-30 分钟\n\n"+
			"提示: 提币正在处理中, 请耐心等待",
		withdrawal.Amount.String(), withdrawal.Currency,
		withdrawal.Fee.String(), withdrawal.Currency,
	)

	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// handleCancel 取消提币
func (h *WithdrawHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 返回主菜单
	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// showCNYPlaceholder 显示 CNY 提现占位 (Phase 2)
func (h *WithdrawHandler) showCNYPlaceholder(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	text := "CNY 人民币提现功能即将上线，敬请期待!"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.BackButton("Wallet--withdraw_menu")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}
