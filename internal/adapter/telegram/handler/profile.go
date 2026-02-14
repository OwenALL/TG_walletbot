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

// ProfileHandler 个人中心回调处理器
type ProfileHandler struct {
	profileUC  *app.ProfileUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewProfileHandler 创建个人中心处理器
func NewProfileHandler(
	profileUC *app.ProfileUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *ProfileHandler {
	return &ProfileHandler{
		profileUC:  profileUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// Handle 处理 Profile-- 前缀的回调
func (h *ProfileHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID

	// 立即响应回调
	tgpkg.AnswerCallback(ctx, b, query.ID)

	_, action := telegram.ParseCallbackData(query.Data)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 个人中心入口 PIN 保护: 未设置 PIN 时引导设置
	if action == "dashboard" && !user.HasPIN() {
		h.pinHandler.StartPINSetup(ctx, b, chatID, messageID, user.TelegramID, "Profile", "dashboard")
		return
	}

	switch {
	case action == "dashboard":
		h.showDashboard(ctx, b, chatID, messageID, user)
	case action == "bills":
		h.showBills(ctx, b, chatID, messageID, user.ID, "ALL", 1)
	case strings.HasPrefix(action, "bills_filter_"):
		// bills_filter_USDT, bills_filter_TRX, bills_filter_CNY, bills_filter_ALL
		currency := strings.TrimPrefix(action, "bills_filter_")
		h.showBills(ctx, b, chatID, messageID, user.ID, currency, 1)
	case strings.HasPrefix(action, "bills_page_"):
		// bills_page_USDT_2
		h.handleBillsPage(ctx, b, chatID, messageID, user.ID, strings.TrimPrefix(action, "bills_page_"))
	case action == "withdraw_history":
		h.showWithdrawalHistory(ctx, b, chatID, messageID, user.ID, 1)
	case strings.HasPrefix(action, "wh_page_"):
		// wh_page_2
		page, _ := strconv.Atoi(strings.TrimPrefix(action, "wh_page_"))
		if page < 1 {
			page = 1
		}
		h.showWithdrawalHistory(ctx, b, chatID, messageID, user.ID, page)
	case action == "small_free":
		h.showSmallFree(ctx, b, chatID, messageID, user.ID)
	case action == "adjust_usdt":
		h.startAdjustLimit(ctx, b, chatID, user.TelegramID, "USDT")
	case action == "adjust_trx":
		h.startAdjustLimit(ctx, b, chatID, user.TelegramID, "TRX")
	case action == "google_auth":
		tgpkg.EditMessage(ctx, b, chatID, messageID, "Google 验证器功能开发中，敬请期待")
	case action == "backup_account":
		tgpkg.EditMessage(ctx, b, chatID, messageID, "备用账号功能开发中，敬请期待")
	case action == "cny_history":
		tgpkg.EditMessage(ctx, b, chatID, messageID, "CNY 提现历史功能开发中，敬请期待")
	default:
		h.logger.Warn("未知 Profile 回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理个人中心文本输入 (小额免密额度调整)
// 返回 true 表示已处理
func (h *ProfileHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Profile" {
		return false
	}

	switch step {
	case "adjust_limit_input":
		h.handleAdjustInput(ctx, b, chatID, telegramID, text)
		return true
	case "adjust_limit_pin":
		// PIN 验证由 PINHandler 处理，这里处理 PIN 验证通过后的回调
		h.handleAdjustAfterPIN(ctx, b, chatID, telegramID)
		return true
	}

	return false
}

// showDashboard 显示个人中心主页
func (h *ProfileHandler) showDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User) {
	profile, err := h.profileUC.GetProfileDashboard(ctx, user.ID)
	if err != nil {
		h.logger.Error("获取个人中心数据失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	usdt := formatBalance(profile.Balances[entity.CurrencyUSDT])
	trx := formatBalance(profile.Balances[entity.CurrencyTRX])
	cny := formatBalance(profile.Balances[entity.CurrencyCNY])

	text := fmt.Sprintf(
		"<b>个人中心</b>\n\n"+
			"昵称: <a href=\"tg://user?id=%d\">%s</a>\n"+
			"ID: <code>%d</code>\n"+
			"用户名: @%s\n"+
			"注册时间: %s\n\n"+
			"<b>资产概览</b>\n"+
			"USDT: <code>%s</code>\n"+
			"TRX: <code>%s</code>\n"+
			"CNY: <code>%s</code>",
		user.TelegramID, user.DisplayName(),
		user.TelegramID,
		usernameOrDefault(user.Username),
		user.CreatedAt.Format("2006-01-02 15:04"),
		usdt, trx, cny,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🔐 绑定 Google 验证器", "Profile--google_auth")).
		Row(tgpkg.Button("🧾 我的账单", "Profile--bills"), tgpkg.Button("📄 提币历史", "Profile--withdraw_history")).
		Row(tgpkg.Button("💳 小额免密", "Profile--small_free"), tgpkg.Button("👤 备用账号", "Profile--backup_account")).
		Row(tgpkg.Button("💸 CNY 提现历史", "Profile--cny_history")).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// showBills 显示账单列表
func (h *ProfileHandler) showBills(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID uint64, currency string, page int) {
	bills, err := h.profileUC.GetBills(ctx, userID, currency, page)
	if err != nil {
		h.logger.Error("获取账单列表失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	var textBuilder strings.Builder
	textBuilder.WriteString("<b>我的账单</b>\n")
	textBuilder.WriteString(fmt.Sprintf("筛选: %s | 第 %d/%d 页\n", currency, bills.Page, maxInt(bills.TotalPages, 1)))
	textBuilder.WriteString("────────────────────\n")

	if len(bills.Items) == 0 {
		textBuilder.WriteString("\n暂无交易记录")
	} else {
		for _, tx := range bills.Items {
			icon := txTypeIcon(tx.Type)
			typeName := txTypeName(tx.Type)
			sign := "+"
			if !tx.IsIncome() {
				sign = "-"
			}
			textBuilder.WriteString(fmt.Sprintf(
				"\n%s <b>%s</b> %s%s %s\n   余额: %s | %s",
				icon, typeName, sign, tx.Amount.StringFixed(4), tx.Currency,
				tx.BalanceAfter.StringFixed(4),
				tx.CreatedAt.Format("01-02 15:04"),
			))
		}
	}

	// 构建键盘: 筛选行 + 分页行 + 返回
	kb := tgpkg.NewInlineKeyboard()

	// 币种筛选行
	filterButtons := []models.InlineKeyboardButton{
		filterButton("ALL", currency),
		filterButton("USDT", currency),
		filterButton("TRX", currency),
		filterButton("CNY", currency),
	}
	kb.Row(filterButtons...)

	// 分页行
	if bills.TotalPages > 1 {
		var pageButtons []models.InlineKeyboardButton
		if page > 1 {
			pageButtons = append(pageButtons, tgpkg.Button("上一页", fmt.Sprintf("Profile--bills_page_%s_%d", currency, page-1)))
		}
		pageButtons = append(pageButtons, tgpkg.Button(fmt.Sprintf("%d/%d", page, bills.TotalPages), "Profile--noop"))
		if page < bills.TotalPages {
			pageButtons = append(pageButtons, tgpkg.Button("下一页", fmt.Sprintf("Profile--bills_page_%s_%d", currency, page+1)))
		}
		kb.Row(pageButtons...)
	}

	kb.Row(tgpkg.Button("返回个人中心", "Profile--dashboard"))

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, textBuilder.String(), kb.Build())
}

// handleBillsPage 处理账单分页回调
// 格式: USDT_2 或 ALL_3
func (h *ProfileHandler) handleBillsPage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID uint64, params string) {
	parts := strings.Split(params, "_")
	if len(parts) != 2 {
		return
	}
	currency := parts[0]
	page, err := strconv.Atoi(parts[1])
	if err != nil || page < 1 {
		page = 1
	}
	h.showBills(ctx, b, chatID, messageID, userID, currency, page)
}

// showWithdrawalHistory 显示提币历史
func (h *ProfileHandler) showWithdrawalHistory(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID uint64, page int) {
	history, err := h.profileUC.GetWithdrawalHistory(ctx, userID, page)
	if err != nil {
		h.logger.Error("获取提币历史失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	var textBuilder strings.Builder
	textBuilder.WriteString("<b>提币历史</b>\n")
	textBuilder.WriteString(fmt.Sprintf("第 %d/%d 页\n", history.Page, maxInt(history.TotalPages, 1)))
	textBuilder.WriteString("────────────────────\n")

	if len(history.Items) == 0 {
		textBuilder.WriteString("\n暂无提币记录")
	} else {
		for _, w := range history.Items {
			statusIcon := withdrawalStatusIcon(w.Status)
			statusText := withdrawalStatusText(w.Status)
			addrShort := truncateAddress(w.ToAddress)
			txHashShort := truncateTxHash(w.TxHash)

			textBuilder.WriteString(fmt.Sprintf(
				"\n%s <b>%s</b> %s %s\n"+
					"   地址: <code>%s</code>\n"+
					"   TxHash: <code>%s</code>\n"+
					"   状态: %s | %s",
				statusIcon, w.Currency, w.Amount.StringFixed(4), statusText,
				addrShort,
				txHashShort,
				statusText,
				w.CreatedAt.Format("01-02 15:04"),
			))
		}
	}

	kb := tgpkg.NewInlineKeyboard()

	// 分页行
	if history.TotalPages > 1 {
		var pageButtons []models.InlineKeyboardButton
		if page > 1 {
			pageButtons = append(pageButtons, tgpkg.Button("上一页", fmt.Sprintf("Profile--wh_page_%d", page-1)))
		}
		pageButtons = append(pageButtons, tgpkg.Button(fmt.Sprintf("%d/%d", page, history.TotalPages), "Profile--noop"))
		if page < history.TotalPages {
			pageButtons = append(pageButtons, tgpkg.Button("下一页", fmt.Sprintf("Profile--wh_page_%d", page+1)))
		}
		kb.Row(pageButtons...)
	}

	kb.Row(tgpkg.Button("返回个人中心", "Profile--dashboard"))

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, textBuilder.String(), kb.Build())
}

// showSmallFree 显示小额免密设置
func (h *ProfileHandler) showSmallFree(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID uint64) {
	info, err := h.profileUC.GetSmallFreeInfo(ctx, userID)
	if err != nil {
		h.logger.Error("获取小额免密配置失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	usdtStatus := "已开启"
	if info.USDTLimit.IsZero() {
		usdtStatus = "已关闭"
	}
	trxStatus := "已开启"
	if info.TRXLimit.IsZero() {
		trxStatus = "已关闭"
	}

	text := fmt.Sprintf(
		"<b>小额免密设置</b>\n\n"+
			"当日累计转账/提币不超过设定额度时，无需输入支付密码。\n"+
			"设为 0 即关闭免密功能。\n\n"+
			"<b>USDT 免密额度:</b> <code>%s</code> (%s)\n"+
			"<b>TRX 免密额度:</b> <code>%s</code> (%s)",
		info.USDTLimit.StringFixed(2), usdtStatus,
		info.TRXLimit.StringFixed(2), trxStatus,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("调整 USDT 额度", "Profile--adjust_usdt"),
			tgpkg.Button("调整 TRX 额度", "Profile--adjust_trx"),
		).
		Row(tgpkg.Button("返回个人中心", "Profile--dashboard")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// startAdjustLimit 开始调整小额免密额度
func (h *ProfileHandler) startAdjustLimit(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, currency string) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Profile", "adjust_limit_input", map[string]interface{}{
		"adjust_currency": currency,
	})

	text := fmt.Sprintf(
		"请输入新的 <b>%s</b> 小额免密额度:\n\n"+
			"输入 0 表示关闭免密功能\n"+
			"输入金额 (如 200) 表示每日累计不超过此额度免验证",
		currency,
	)
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("取消", "Menu--cancel")).
		Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// handleAdjustInput 处理免密额度输入
func (h *ProfileHandler) handleAdjustInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	amount, err := decimal.NewFromString(strings.TrimSpace(text))
	if err != nil || amount.IsNegative() {
		tgpkg.SendMessage(ctx, b, chatID, "请输入有效的金额数字 (大于等于 0):")
		return
	}

	// 上限检查
	maxLimit := decimal.NewFromInt(100000)
	if amount.GreaterThan(maxLimit) {
		tgpkg.SendMessage(ctx, b, chatID, fmt.Sprintf("额度不能超过 %s，请重新输入:", maxLimit.String()))
		return
	}

	// 保存金额到 session，发起 PIN 验证
	_ = h.sessionSvc.SetData(ctx, telegramID, "adjust_amount", amount.String())

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 发起 PIN 验证 (文本输入上下文，无回调消息 ID，传 0 发送新消息)
	h.pinHandler.StartPINVerify(ctx, b, chatID, 0, telegramID, "Profile", "adjust_limit_pin")
}

// handleAdjustAfterPIN 处理 PIN 验证通过后的额度更新
func (h *ProfileHandler) handleAdjustAfterPIN(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	currency := h.sessionSvc.GetDataString(ctx, telegramID, "adjust_currency")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "adjust_amount")

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	if err := h.profileUC.UpdateSmallFreeLimit(ctx, user.ID, currency, amount); err != nil {
		h.logger.Error("更新小额免密额度失败", zap.Error(err))
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	_ = h.sessionSvc.Clear(ctx, telegramID)

	statusText := fmt.Sprintf("%s %s", amount.StringFixed(2), currency)
	if amount.IsZero() {
		statusText = "已关闭"
	}

	text := fmt.Sprintf("设置成功!\n\n%s 小额免密额度: <code>%s</code>", currency, statusText)
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("返回小额免密", "Profile--small_free")).
		Row(tgpkg.MainMenuButton()).
		Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// RegisterProfileHandlers 注册 Profile 模块到路由器
func RegisterProfileHandlers(router *telegram.Router, handler *ProfileHandler) {
	router.RegisterModule("Profile", handler.Handle)
}

// --- 辅助函数 ---

// filterButton 构建筛选按钮 (当前选中加标记)
func filterButton(currency, activeCurrency string) models.InlineKeyboardButton {
	text := currency
	if currency == activeCurrency {
		text = "[" + currency + "]"
	}
	return tgpkg.Button(text, "Profile--bills_filter_"+currency)
}

// txTypeIcon 交易类型图标
func txTypeIcon(txType string) string {
	switch txType {
	case entity.TxTypeDeposit:
		return "+"
	case entity.TxTypeWithdraw:
		return "-"
	case entity.TxTypeTransferIn:
		return "+"
	case entity.TxTypeTransferOut:
		return "-"
	case entity.TxTypeExchangeIn:
		return "+"
	case entity.TxTypeExchangeOut:
		return "-"
	case entity.TxTypeRedpacketSend:
		return "-"
	case entity.TxTypeRedpacketRecv:
		return "+"
	case entity.TxTypeRedpacketRefund:
		return "+"
	default:
		return " "
	}
}

// txTypeName 交易类型名称
func txTypeName(txType string) string {
	switch txType {
	case entity.TxTypeDeposit:
		return "充值"
	case entity.TxTypeWithdraw:
		return "提币"
	case entity.TxTypeTransferIn:
		return "转入"
	case entity.TxTypeTransferOut:
		return "转出"
	case entity.TxTypeExchangeIn:
		return "兑入"
	case entity.TxTypeExchangeOut:
		return "兑出"
	case entity.TxTypeRedpacketSend:
		return "发红包"
	case entity.TxTypeRedpacketRecv:
		return "领红包"
	case entity.TxTypeRedpacketRefund:
		return "红包退回"
	case entity.TxTypeFinanceIn:
		return "买入理财"
	case entity.TxTypeFinanceOut:
		return "取出理财"
	case entity.TxTypeFinanceProfit:
		return "理财收益"
	default:
		return "其他"
	}
}

// withdrawalStatusIcon 提币状态图标
func withdrawalStatusIcon(status int8) string {
	switch status {
	case entity.WithdrawalStatusPending:
		return "..."
	case entity.WithdrawalStatusProcessing:
		return ">>>"
	case entity.WithdrawalStatusCompleted:
		return "OK"
	case entity.WithdrawalStatusRejected:
		return "X"
	case entity.WithdrawalStatusFailed:
		return "!"
	default:
		return "?"
	}
}

// withdrawalStatusText 提币状态文本
func withdrawalStatusText(status int8) string {
	switch status {
	case entity.WithdrawalStatusPending:
		return "待审核"
	case entity.WithdrawalStatusProcessing:
		return "处理中"
	case entity.WithdrawalStatusCompleted:
		return "已完成"
	case entity.WithdrawalStatusRejected:
		return "已拒绝"
	case entity.WithdrawalStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// truncateAddress 截断地址显示
func truncateAddress(address string) string {
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "..." + address[len(address)-6:]
}

// truncateTxHash 截断交易哈希显示
func truncateTxHash(hash string) string {
	if hash == "" {
		return "-"
	}
	if len(hash) <= 16 {
		return hash
	}
	return hash[:8] + "..." + hash[len(hash)-8:]
}

// usernameOrDefault 返回用户名或默认值
func usernameOrDefault(username string) string {
	if username == "" {
		return "未设置"
	}
	return username
}

// maxInt 返回两个 int 中的较大值
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
