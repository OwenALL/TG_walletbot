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

	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/app"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// FinanceHandler 余额宝模块 Bot Handler
type FinanceHandler struct {
	financeUC  *app.FinanceUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewFinanceHandler 创建余额宝处理器
func NewFinanceHandler(
	financeUC *app.FinanceUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *FinanceHandler {
	return &FinanceHandler{
		financeUC:  financeUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理 Finance-- 回调
func (h *FinanceHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	switch {
	case action == "dashboard":
		h.showDashboard(ctx, b, chatID, messageID, telegramID)
	case action == "invest":
		h.promptInvestAmount(ctx, b, chatID, messageID, telegramID)
	case action == "confirm_invest":
		h.handleConfirmInvest(ctx, b, chatID, messageID, telegramID)
	case action == "pin_verified":
		h.doInvest(ctx, b, chatID, telegramID)
	case strings.HasPrefix(action, "withdraw_"):
		h.handleWithdraw(ctx, b, chatID, messageID, telegramID, action)
	case action == "my_investments":
		h.showMyInvestments(ctx, b, chatID, messageID, telegramID)
	default:
		h.logger.Warn("未知余额宝回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理余额宝模块的文本消息输入
// 返回 true 表示已处理
func (h *FinanceHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Finance" {
		return false
	}

	switch step {
	case "invest_amount":
		h.handleInputAmount(ctx, b, chatID, telegramID, update.Message.Text)
	case "pin_verified":
		h.doInvest(ctx, b, chatID, telegramID)
	default:
		return false
	}

	return true
}

// showDashboard 显示余额宝面板 (产品介绍 + 持仓信息)
func (h *FinanceHandler) showDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	// 清除可能存在的旧会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	dashboard, err := h.financeUC.GetDashboardData(ctx, user.ID)
	if err != nil {
		h.logger.Error("获取余额宝面板数据失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := fmt.Sprintf(
		"<b>余额宝</b>\n\n"+
			"\U0001f4b0 买入金额 : <b> %s 至 %s USDT</b>\n"+
			"\U0001f4c8 当前年化利率 : <b>%s%%</b> (此利率可能会基于风险规避策略变动)\n\n"+
			"<b>到账方式：</b>\n"+
			"每日收益自动发放至钱包余额账户。\n\n"+
			"<b>取出规则：</b>\n"+
			"随时可以取出至钱包余额账户。\n\n"+
			"<b>派息规则:</b>\n"+
			"购买后24小时开始派息。\n\n"+
			"<b>收益计算:</b>\n"+
			"当日收益=存入资产*年化收益率/365",
		dashboard.MinInvest.String(),
		dashboard.MaxInvest.String(),
		dashboard.AnnualRate.String(),
	)

	// 如果有持仓，追加持仓统计
	if len(dashboard.Investments) > 0 {
		text += fmt.Sprintf(
			"\n\n────────────────────\n"+
				"\U0001f4ca 我的持仓: <code>%s</code> USDT\n"+
				"\U0001f4c8 累计收益: <code>%s</code> USDT\n"+
				"\U0001f4b0 昨日收益: <code>%s</code> USDT",
			dashboard.TotalAmount.StringFixed(4),
			dashboard.TotalProfit.StringFixed(4),
			dashboard.YesterdayProfit.StringFixed(4),
		)
	}

	// 构建键盘
	kb := tgpkg.NewInlineKeyboard()
	if len(dashboard.Investments) > 0 {
		kb.Row(
			tgpkg.Button("\U0001f4c8 投资", "Finance--invest"),
			tgpkg.Button("\U0001f4cb 我的持仓", "Finance--my_investments"),
		)
	} else {
		kb.Row(tgpkg.Button("\U0001f4c8 投资", "Finance--invest"))
	}
	kb.Row(tgpkg.MainMenuButton())
	keyboard := kb.Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// promptInvestAmount 提示用户输入投资金额
func (h *FinanceHandler) promptInvestAmount(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取 USDT 余额
	balance, err := h.financeUC.GetUSDTBalance(ctx, user.ID)
	if err != nil {
		h.logger.Error("获取 USDT 余额失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 设置会话状态: 等待输入金额
	_ = h.sessionSvc.SetStep(ctx, telegramID, "Finance", "invest_amount")

	text := fmt.Sprintf(
		"\U0001f4b0 请输入您要投资的金额\n\n"+
			"当前余额: <code>%s</code> USDT",
		balance.StringFixed(4),
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.BackButton("Finance--dashboard")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleInputAmount 处理用户输入的投资金额
func (h *FinanceHandler) handleInputAmount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	amount, err := decimal.NewFromString(text)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.SendMessage(ctx, b, chatID, "请输入有效的数字金额。")
		return
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 获取配置进行预校验
	cfg, err := h.financeUC.GetFinanceConfig(ctx)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	if amount.LessThan(cfg.MinInvest) {
		tgpkg.SendMessage(ctx, b, chatID, fmt.Sprintf("最低投资金额为 %s USDT", cfg.MinInvest.String()))
		return
	}
	if amount.GreaterThan(cfg.MaxInvest) {
		tgpkg.SendMessage(ctx, b, chatID, fmt.Sprintf("最高投资金额为 %s USDT", cfg.MaxInvest.String()))
		return
	}

	// 检查余额
	balance, err := h.financeUC.GetUSDTBalance(ctx, user.ID)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}
	if balance.LessThan(amount) {
		tgpkg.SendMessage(ctx, b, chatID, "余额不足，请先充值。")
		return
	}

	// 计算预计日收益
	dailyProfit := amount.Mul(cfg.AnnualRate).Div(decimal.NewFromInt(100)).Div(decimal.NewFromInt(365))

	// 保存金额到会话，进入确认步骤
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Finance", "confirm_invest", map[string]interface{}{
		"invest_amount": amount.String(),
		"annual_rate":   cfg.AnnualRate.String(),
		"daily_profit":  dailyProfit.StringFixed(8),
	})

	confirmText := fmt.Sprintf(
		"<b>投资确认</b>\n\n"+
			"金额: <code>%s</code> USDT\n"+
			"年化利率: %s%%\n"+
			"预计日收益: <code>%s</code> USDT\n\n"+
			"确认投资请点击下方按钮",
		amount.String(),
		cfg.AnnualRate.String(),
		dailyProfit.StringFixed(8),
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("确认投资", "Finance--confirm_invest"),
			tgpkg.Button("取消", "Finance--dashboard"),
		).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, confirmText, keyboard)
}

// handleConfirmInvest 处理确认投资 (发起 PIN 验证)
func (h *FinanceHandler) handleConfirmInvest(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 获取投资金额
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "invest_amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 检查是否需要 PIN
	needPIN, err := h.userUC.NeedPIN(ctx, user.ID, "USDT", amount)
	if err != nil {
		needPIN = true
	}

	if needPIN {
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Finance", "pin_verified")
		return
	}

	// 不需要 PIN，直接执行
	tgpkg.EditMessage(ctx, b, chatID, messageID, "\u23f3 处理中...")
	h.doInvest(ctx, b, chatID, telegramID)
}

// doInvest 执行投资 (PIN 验证通过后或不需要 PIN 时调用)
func (h *FinanceHandler) doInvest(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 从会话读取投资金额
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "invest_amount")
	amount, err := decimal.NewFromString(amountStr)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 执行买入
	investment, err := h.financeUC.Invest(ctx, user.ID, amount)
	if err != nil {
		h.logger.Error("余额宝买入执行失败",
			zap.Uint64("user_id", user.ID),
			zap.String("amount", amountStr),
			zap.Error(err),
		)
		errMsg := "投资失败，请稍后重试。"
		if strings.Contains(err.Error(), "余额不足") {
			errMsg = "余额不足，请先充值。"
		} else if strings.Contains(err.Error(), "最低") || strings.Contains(err.Error(), "最高") {
			errMsg = err.Error()
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送成功消息
	dailyProfit := investment.DailyProfit()
	successText := fmt.Sprintf(
		"<b>投资成功!</b>\n\n"+
			"投资金额: <code>%s</code> USDT\n"+
			"年化利率: %s%%\n"+
			"预计日收益: <code>%s</code> USDT\n"+
			"开始计息: 24小时后",
		investment.Amount.String(),
		investment.AnnualRate.String(),
		dailyProfit.StringFixed(8),
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("继续投资", "Finance--invest")).
		Row(tgpkg.Button("查看持仓", "Finance--my_investments")).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, successText, keyboard)
}

// handleWithdraw 处理取出投资
func (h *FinanceHandler) handleWithdraw(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, action string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 解析投资 ID: withdraw_{id}
	idStr := strings.TrimPrefix(action, "withdraw_")
	investmentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.logger.Warn("无效的投资 ID", zap.String("id_str", idStr))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 编辑为处理中
	tgpkg.EditMessage(ctx, b, chatID, messageID, "\u23f3 处理中...")

	// 执行取出
	err = h.financeUC.Withdraw(ctx, investmentID, user.ID)
	if err != nil {
		h.logger.Error("余额宝取出失败",
			zap.Uint64("user_id", user.ID),
			zap.Uint64("investment_id", investmentID),
			zap.Error(err),
		)
		errMsg := "取出失败，请稍后重试。"
		if strings.Contains(err.Error(), "已取出") {
			errMsg = "该投资已取出。"
		} else if strings.Contains(err.Error(), "无权") {
			errMsg = "无权操作该投资记录。"
		}
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, errMsg,
			tgpkg.NewInlineKeyboard().
				Row(tgpkg.BackButton("Finance--dashboard")).
				Build(),
		)
		return
	}

	// 取出成功
	successText := "<b>取出成功!</b>\n\n本金已退回至您的 USDT 钱包余额。"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("查看持仓", "Finance--my_investments")).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, successText, keyboard)
}

// showMyInvestments 显示持仓列表
func (h *FinanceHandler) showMyInvestments(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	investments, err := h.financeUC.GetUserInvestments(ctx, user.ID)
	if err != nil {
		h.logger.Error("获取持仓列表失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	if len(investments) == 0 {
		text := "\U0001f4cb <b>我的持仓</b>\n\n暂无持仓记录"
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("\U0001f4c8 去投资", "Finance--invest")).
			Row(tgpkg.BackButton("Finance--dashboard")).
			Build()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 构建持仓列表文本
	text := "\U0001f4cb <b>我的持仓</b>\n"
	kb := tgpkg.NewInlineKeyboard()

	for _, inv := range investments {
		text += fmt.Sprintf(
			"\n#%d | <code>%s</code> USDT | 年化 %s%%\n"+
				"  累计收益: <code>%s</code> USDT\n"+
				"────────────────────",
			inv.ID,
			inv.Amount.StringFixed(2),
			inv.AnnualRate.String(),
			inv.TotalProfit.StringFixed(4),
		)

		// 每笔投资一个取出按钮
		kb.Row(tgpkg.Button(
			fmt.Sprintf("\U0001f4e4 取出 #%d (%s USDT)", inv.ID, inv.Amount.StringFixed(2)),
			fmt.Sprintf("Finance--withdraw_%d", inv.ID),
		))
	}

	kb.Row(tgpkg.BackButton("Finance--dashboard"))
	keyboard := kb.Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// RegisterFinanceHandlers 注册余额宝相关的 handler 到 router
func RegisterFinanceHandlers(router *telegram.Router, h *FinanceHandler) {
	router.RegisterModule("Finance", h.HandleCallback)
}
