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
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// RedPacketHandler 红包模块 Bot Handler
type RedPacketHandler struct {
	redPacketUC *app.RedPacketUseCase
	userUC      *app.UserUseCase
	pinHandler  *PINHandler
	sessionSvc  *tgpkg.SessionService
	logger      *zap.Logger
}

// NewRedPacketHandler 创建红包处理器
func NewRedPacketHandler(
	redPacketUC *app.RedPacketUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *RedPacketHandler {
	return &RedPacketHandler{
		redPacketUC: redPacketUC,
		userUC:      userUC,
		pinHandler:  pinHandler,
		sessionSvc:  sessionSvc,
		logger:      logger,
	}
}

// HandleCallback 处理 RedPacket-- 回调
func (h *RedPacketHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	telegramID := query.From.ID

	_, action := telegram.ParseCallbackData(query.Data)

	// ---- 群聊领取 (可能来自 inline 消息) ----
	// inline 消息没有 query.Message，只有 query.InlineMessageID
	// 必须在获取 chatID/messageID 之前处理
	if strings.HasPrefix(action, "claim_") {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		packetIDStr := strings.TrimPrefix(action, "claim_")

		if query.InlineMessageID != "" {
			// inline 模式发送的红包: 使用 InlineMessageID 编辑
			h.handleClaimInline(ctx, b, query.InlineMessageID, telegramID, packetIDStr)
		} else {
			// 普通消息中的红包
			var chatID int64
			var messageID int
			if query.Message.Message != nil {
				chatID = query.Message.Message.Chat.ID
				messageID = query.Message.Message.ID
			} else if query.Message.InaccessibleMessage != nil {
				chatID = query.Message.InaccessibleMessage.Chat.ID
				messageID = query.Message.InaccessibleMessage.MessageID
			} else {
				return
			}
			h.handleClaim(ctx, b, chatID, messageID, telegramID, packetIDStr)
		}
		return
	}

	// 立即响应回调
	tgpkg.AnswerCallback(ctx, b, query.ID)

	// 以下操作均为私聊场景，安全获取 chatID / messageID
	var chatID int64
	var messageID int
	if query.Message.Message != nil {
		chatID = query.Message.Message.Chat.ID
		messageID = query.Message.Message.ID
	} else if query.Message.InaccessibleMessage != nil {
		chatID = query.Message.InaccessibleMessage.Chat.ID
		messageID = query.Message.InaccessibleMessage.MessageID
	} else {
		return
	}

	switch {
	// ---- 面板 & 标签 ----
	case action == "dashboard" || action == "tab_active":
		h.showDashboard(ctx, b, chatID, messageID, telegramID, "active")
	case action == "tab_finished":
		h.showDashboard(ctx, b, chatID, messageID, telegramID, "finished")

	// ---- 创建流程 ----
	case action == "add" || action == "another":
		h.showCurrencySelect(ctx, b, chatID, messageID)
	case strings.HasPrefix(action, "currency_"):
		currency := strings.TrimPrefix(action, "currency_")
		h.promptAmount(ctx, b, chatID, messageID, telegramID, currency)
	case action == "confirm":
		h.handleConfirm(ctx, b, chatID, messageID, telegramID)
	case action == "pin_verified":
		h.doCreate(ctx, b, chatID, telegramID)

	// ---- 红包详情 & 设置 ----
	case strings.HasPrefix(action, "detail_"):
		idStr := strings.TrimPrefix(action, "detail_")
		h.showRedPacketDetail(ctx, b, chatID, messageID, telegramID, idStr)
	case strings.HasPrefix(action, "send_"):
		idStr := strings.TrimPrefix(action, "send_")
		h.showSendTip(ctx, b, chatID, messageID, telegramID, idStr)
	case strings.HasPrefix(action, "set_remark_"):
		idStr := strings.TrimPrefix(action, "set_remark_")
		h.promptRemark(ctx, b, chatID, messageID, telegramID, idStr)
	case strings.HasPrefix(action, "toggle_captcha_"):
		idStr := strings.TrimPrefix(action, "toggle_captcha_")
		h.toggleField(ctx, b, chatID, messageID, telegramID, idStr, "is_captcha")
	case strings.HasPrefix(action, "toggle_premium_"):
		idStr := strings.TrimPrefix(action, "toggle_premium_")
		h.toggleField(ctx, b, chatID, messageID, telegramID, idStr, "is_premium_only")
	case strings.HasPrefix(action, "toggle_private_"):
		idStr := strings.TrimPrefix(action, "toggle_private_")
		h.toggleField(ctx, b, chatID, messageID, telegramID, idStr, "is_private")
	case strings.HasPrefix(action, "close_"):
		idStr := strings.TrimPrefix(action, "close_")
		h.closeRedPacket(ctx, b, chatID, messageID, telegramID, idStr)

	// ---- 封面管理 ----
	case action == "cover":
		h.showCoverList(ctx, b, chatID, messageID, telegramID)
	case action == "cover_add":
		h.promptCoverUpload(ctx, b, chatID, messageID, telegramID)
	case strings.HasPrefix(action, "cover_select_"):
		coverID := strings.TrimPrefix(action, "cover_select_")
		h.handleCoverSelect(ctx, b, chatID, messageID, telegramID, coverID)

	// ---- 通用 ----
	case action == "cancel":
		h.handleCancel(ctx, b, chatID, messageID, telegramID)

	default:
		h.logger.Warn("未知红包回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理红包模块的文本消息输入
// 返回 true 表示已处理
func (h *RedPacketHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "RedPacket" {
		return false
	}

	switch step {
	case "amount":
		h.handleInputAmount(ctx, b, chatID, telegramID, update.Message.Text)
	case "count":
		h.handleInputCount(ctx, b, chatID, telegramID, update.Message.Text)
	case "remark":
		h.handleInputRemark(ctx, b, chatID, telegramID, update.Message.Text)
	case "cover_upload":
		h.handleCoverUpload(ctx, b, chatID, telegramID, update)
	default:
		return false
	}

	return true
}

// =====================================================
// 红包面板
// =====================================================

// showDashboard 显示红包面板 (支持 进行中/已结束 标签切换)
func (h *RedPacketHandler) showDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, tab string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	var packets []*entity.RedPacket
	var err error

	if tab == "finished" {
		packets, err = h.redPacketUC.GetUserFinishedPackets(ctx, user.ID)
	} else {
		packets, err = h.redPacketUC.GetUserActivePackets(ctx, user.ID)
	}
	if err != nil {
		h.logger.Error("查询用户红包失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := "从下面的列表中选择一个红包 :\n"
	if len(packets) > 0 {
		text += "\n"
		for _, p := range packets {
			typeName := app.RedPacketTypeName(p.Type)
			statusText := redPacketStatusText(p.Status)
			text += fmt.Sprintf("<code>#%d</code> | %s %s | %d个 | %s | %s\n",
				p.ID, p.Currency, p.TotalAmount.String(), p.TotalCount, typeName, statusText)
		}
	} else {
		if tab == "finished" {
			text += "\n暂无已结束的红包"
		} else {
			text += "\n暂无进行中的红包"
		}
	}

	kb := tgpkg.NewInlineKeyboard()

	// 每个红包一行详情按钮
	for _, p := range packets {
		if p.Status == entity.RedPacketStatusPending {
			kb.Row(tgpkg.Button(
				fmt.Sprintf("🧧 #%d (%s %s)", p.ID, p.Currency, p.TotalAmount.String()),
				fmt.Sprintf("RedPacket--detail_%d", p.ID),
			))
		} else {
			kb.Row(tgpkg.Button(
				fmt.Sprintf("#%d (%s %s) %s", p.ID, p.Currency, p.TotalAmount.String(), redPacketStatusText(p.Status)),
				fmt.Sprintf("RedPacket--detail_%d", p.ID),
			))
		}
	}

	// 标签切换按钮
	if tab == "finished" {
		kb.Row(
			tgpkg.Button("← 进行中", "RedPacket--tab_active"),
			tgpkg.Button("✓ 已结束", "RedPacket--tab_finished"),
		)
	} else {
		kb.Row(
			tgpkg.Button("✓ 进行中", "RedPacket--tab_active"),
			tgpkg.Button("已结束 →", "RedPacket--tab_finished"),
		)
	}

	// 功能按钮
	kb.Row(
		tgpkg.Button("➕ 添加", "RedPacket--add"),
		tgpkg.Button("🖼 设置封面", "RedPacket--cover"),
	)
	kb.Row(tgpkg.Button("🔙 返回", "Menu--main"))

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, kb.Build())
}

// =====================================================
// 创建红包流程
// =====================================================

// showCurrencySelect 显示币种选择
func (h *RedPacketHandler) showCurrencySelect(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	text := "<b>🧧 请选择货币</b>"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("USDT", "RedPacket--currency_USDT"),
			tgpkg.Button("TRX", "RedPacket--currency_TRX"),
		).
		Row(
			tgpkg.Button("CNY", "RedPacket--currency_CNY"),
		).
		Row(tgpkg.Button("🚫 取消", "RedPacket--cancel")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// promptAmount 提示输入金额
func (h *RedPacketHandler) promptAmount(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, currency string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	balance, err := h.redPacketUC.GetBalance(ctx, user.ID, currency)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 保存币种到会话
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "RedPacket", "amount", map[string]interface{}{
		"currency": currency,
	})

	text := fmt.Sprintf(
		"🔸 请回复你要发送的总金额(%s)? 例如: <code>8.88</code>\n\n"+
			"当前余额 : <code>%s</code> %s",
		currency, balance.StringFixed(4), currency,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", "RedPacket--cancel")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleInputAmount 处理金额输入
func (h *RedPacketHandler) handleInputAmount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	amount, err := decimal.NewFromString(text)
	if err != nil || amount.LessThanOrEqual(decimal.Zero) {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 请输入有效的金额，例如: <code>8.88</code>")
		return
	}

	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")
	if currency == "" {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 检查余额
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	balance, err := h.redPacketUC.GetBalance(ctx, user.ID, currency)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}
	if balance.LessThan(amount) {
		tgpkg.SendMessage(ctx, b, chatID, fmt.Sprintf("⚠️ 余额不足，当前余额: <code>%s</code> %s", balance.StringFixed(4), currency))
		return
	}

	// 保存金额到会话，进入输入个数步骤
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "RedPacket", "count", map[string]interface{}{
		"amount": amount.String(),
	})

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", "RedPacket--cancel")).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, "🔸 请回复你要发送的红包数量", keyboard)
}

// handleInputCount 处理个数输入 → 直接进入确认 (默认拼手气)
func (h *RedPacketHandler) handleInputCount(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	count, err := strconv.Atoi(text)
	if err != nil || count <= 0 {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 请输入有效的正整数")
		return
	}
	if count > 100 {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 红包个数最多 100 个")
		return
	}

	// 保存个数到会话，默认拼手气类型，直接进入确认
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "RedPacket", "confirm", map[string]interface{}{
		"count": strconv.Itoa(count),
		"type":  strconv.Itoa(int(entity.RedPacketTypeRandom)),
	})

	// 读取已有数据
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")

	confirmText := fmt.Sprintf(
		"<b>🧧 红包确认</b>\n\n"+
			"币种: %s\n"+
			"总金额: %s\n"+
			"个数: %d\n"+
			"类型: 拼手气红包",
		currency, amountStr, count,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("✅ 确认", "RedPacket--confirm"),
			tgpkg.Button("🚫 取消", "RedPacket--cancel"),
		).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, confirmText, keyboard)
}

// handleConfirm 处理确认创建 (发起 PIN 验证)
func (h *RedPacketHandler) handleConfirm(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	amount, _ := decimal.NewFromString(amountStr)
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")

	// 检查是否需要 PIN
	needPIN, err := h.userUC.NeedPIN(ctx, user.ID, currency, amount)
	if err != nil {
		needPIN = true
	}

	if needPIN {
		h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "RedPacket", "pin_verified")
		return
	}

	// 不需要 PIN，直接创建
	tgpkg.EditMessage(ctx, b, chatID, messageID, "处理中...")
	h.doCreate(ctx, b, chatID, telegramID)
}

// doCreate 执行创建红包 (PIN 验证通过后)
func (h *RedPacketHandler) doCreate(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 从会话中读取参数
	currency := h.sessionSvc.GetDataString(ctx, telegramID, "currency")
	amountStr := h.sessionSvc.GetDataString(ctx, telegramID, "amount")
	countStr := h.sessionSvc.GetDataString(ctx, telegramID, "count")
	typeStr := h.sessionSvc.GetDataString(ctx, telegramID, "type")
	coverFileID := h.sessionSvc.GetDataString(ctx, telegramID, "cover_file_id")

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}
	count, err := strconv.Atoi(countStr)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}
	packetType, err := strconv.Atoi(typeStr)
	if err != nil {
		packetType = int(entity.RedPacketTypeRandom) // 默认拼手气
	}

	// 创建红包
	packet, err := h.redPacketUC.CreateRedPacket(ctx, user.ID, currency, amount, count, int8(packetType), coverFileID)
	if err != nil {
		h.logger.Error("创建红包失败",
			zap.Uint64("user_id", user.ID),
			zap.Error(err),
		)
		errMsg := "❌ 创建红包失败，请稍后重试"
		if strings.Contains(err.Error(), "余额不足") {
			errMsg = "⚠️ 余额不足，请先充值"
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 显示红包详情页面
	h.sendRedPacketDetail(ctx, b, chatID, packet)
}

// =====================================================
// 红包详情页面
// =====================================================

// showRedPacketDetail 显示红包详情 (编辑已有消息)
func (h *RedPacketHandler) showRedPacketDetail(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, idStr string) {
	packetID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	packet, err := h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if err != nil || packet == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := h.buildDetailText(packet)
	keyboard := h.buildDetailKeyboard(packet)

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// sendRedPacketDetail 发送红包详情 (创建后发送新消息)
func (h *RedPacketHandler) sendRedPacketDetail(ctx context.Context, b *bot.Bot, chatID int64, packet *entity.RedPacket) {
	text := h.buildDetailText(packet)
	keyboard := h.buildDetailKeyboard(packet)

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// buildDetailText 构建红包详情文案
func (h *RedPacketHandler) buildDetailText(packet *entity.RedPacket) string {
	typeName := app.RedPacketTypeName(packet.Type)
	statusText := redPacketStatusText(packet.Status)

	text := fmt.Sprintf(
		"🧧 <b>红包信息</b>\n\n"+
			"编号: <code>#%d</code>\n"+
			"币种: %s\n"+
			"金额: %s\n"+
			"数量: %d\n"+
			"类型: %s\n"+
			"状态: %s",
		packet.ID, packet.Currency, packet.TotalAmount.String(),
		packet.TotalCount, typeName, statusText,
	)

	// 显示附加信息
	if packet.Remark != "" {
		text += fmt.Sprintf("\n备注: %s", packet.Remark)
	}

	// 功能标记
	var tags []string
	if packet.IsCaptcha {
		tags = append(tags, "🔐 验证码")
	}
	if packet.IsPremiumOnly {
		tags = append(tags, "💎 会员专属")
	}
	if packet.IsPrivate {
		tags = append(tags, "🤫 隐私")
	}
	if len(tags) > 0 {
		text += "\n特殊: " + strings.Join(tags, " | ")
	}

	// 已发送后显示领取进度
	if packet.Status == entity.RedPacketStatusSent || packet.Status == entity.RedPacketStatusClaimed {
		text += fmt.Sprintf("\n\n领取进度: %d/%d", packet.ClaimedCount, packet.TotalCount)
	}

	return text
}

// buildDetailKeyboard 构建红包详情键盘
func (h *RedPacketHandler) buildDetailKeyboard(packet *entity.RedPacket) *models.InlineKeyboardMarkup {
	kb := tgpkg.NewInlineKeyboard()

	if packet.Status == entity.RedPacketStatusPending {
		// 待发送: 显示完整操作按钮
		kb.Row(
			tgpkg.Button("👉 再发一个", "RedPacket--another"),
			tgpkg.SwitchInlineButton("🧧 发送红包", fmt.Sprintf("redpacket %d", packet.ID)),
		)
		kb.Row(tgpkg.Button("📝 设置备注", fmt.Sprintf("RedPacket--set_remark_%d", packet.ID)))

		// 验证码 + 会员
		captchaText := "🔐 设置为验证码红包"
		if packet.IsCaptcha {
			captchaText = "🔐 ✅ 验证码红包"
		}
		premiumText := "💎 设置会员红包"
		if packet.IsPremiumOnly {
			premiumText = "💎 ✅ 会员红包"
		}
		kb.Row(
			tgpkg.Button(captchaText, fmt.Sprintf("RedPacket--toggle_captcha_%d", packet.ID)),
			tgpkg.Button(premiumText, fmt.Sprintf("RedPacket--toggle_premium_%d", packet.ID)),
		)

		// 隐私红包
		privateText := "🤫 设置隐私红包"
		if packet.IsPrivate {
			privateText = "🤫 ✅ 隐私红包"
		}
		kb.Row(tgpkg.Button(privateText, fmt.Sprintf("RedPacket--toggle_private_%d", packet.ID)))

		// 关闭 (退款)
		kb.Row(tgpkg.Button("🔴 关闭", fmt.Sprintf("RedPacket--close_%d", packet.ID)))
	} else {
		// 已发送/已领完/已过期/已关闭: 仅返回按钮
		kb.Row(tgpkg.Button("🔙 返回列表", "RedPacket--dashboard"))
	}

	return kb.Build()
}

// =====================================================
// 红包设置操作
// =====================================================

// promptRemark 提示输入备注
func (h *RedPacketHandler) promptRemark(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, idStr string) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "RedPacket", "remark", map[string]interface{}{
		"remark_packet_id": idStr,
	})

	text := "📝 请输入红包备注 (最多 50 字)"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", fmt.Sprintf("RedPacket--detail_%s", idStr))).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleInputRemark 处理备注文本输入
func (h *RedPacketHandler) handleInputRemark(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 备注不能为空")
		return
	}
	// 限制长度
	runeText := []rune(text)
	if len(runeText) > 50 {
		text = string(runeText[:50])
	}

	idStr := h.sessionSvc.GetDataString(ctx, telegramID, "remark_packet_id")
	if idStr == "" {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	packetID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 验证所有权
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	packet, err := h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if err != nil || packet == nil || packet.UserID != user.ID {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 红包不存在或无权操作")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 更新备注
	if err := h.redPacketUC.UpdateRedPacketFields(ctx, packetID, map[string]interface{}{
		"remark": text,
	}); err != nil {
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 设置备注失败")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 重新加载并显示详情
	packet, _ = h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if packet != nil {
		h.sendRedPacketDetail(ctx, b, chatID, packet)
	} else {
		tgpkg.SendMessage(ctx, b, chatID, "✅ 备注已设置")
	}
}

// toggleField 切换红包布尔字段 (is_captcha / is_premium_only / is_private)
func (h *RedPacketHandler) toggleField(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, idStr string, fieldName string) {
	packetID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	packet, err := h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if err != nil || packet == nil || packet.UserID != user.ID {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 计算新值
	var newVal bool
	switch fieldName {
	case "is_captcha":
		newVal = !packet.IsCaptcha
	case "is_premium_only":
		newVal = !packet.IsPremiumOnly
	case "is_private":
		newVal = !packet.IsPrivate
	default:
		return
	}

	if err := h.redPacketUC.UpdateRedPacketFields(ctx, packetID, map[string]interface{}{
		fieldName: newVal,
	}); err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 重新加载并刷新详情
	packet, _ = h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if packet == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := h.buildDetailText(packet)
	keyboard := h.buildDetailKeyboard(packet)
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// closeRedPacket 关闭红包 (退款)
func (h *RedPacketHandler) closeRedPacket(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, idStr string) {
	packetID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	if err := h.redPacketUC.CancelRedPacket(ctx, packetID, user.ID); err != nil {
		h.logger.Error("关闭红包失败", zap.Error(err))
		errMsg := "❌ 关闭红包失败"
		if strings.Contains(err.Error(), "只能关闭") {
			errMsg = "⚠️ 只能关闭待发送的红包"
		}
		tgpkg.EditMessage(ctx, b, chatID, messageID, errMsg)
		return
	}

	text := "✅ 红包已关闭，金额已退回钱包"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🔙 返回列表", "RedPacket--dashboard")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// =====================================================
// 发送到群聊
// =====================================================

// showSendTip 显示发送提示
func (h *RedPacketHandler) showSendTip(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, packetIDStr string) {
	packetID, err := strconv.ParseUint(packetIDStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	packet, err := h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if err != nil || packet == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := fmt.Sprintf(
		"🧧 <b>发送红包</b>\n\n"+
			"红包编号: <code>#%d</code>\n"+
			"币种: %s\n"+
			"总金额: %s\n"+
			"个数: %d\n\n"+
			"点击下方按钮选择群聊发送",
		packet.ID, packet.Currency, packet.TotalAmount.String(), packet.TotalCount,
	)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.SwitchInlineButton(
			"🧧 选择群聊发送",
			fmt.Sprintf("redpacket %d", packet.ID),
		)).
		Row(tgpkg.Button("🔙 返回", fmt.Sprintf("RedPacket--detail_%d", packet.ID))).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// =====================================================
// 红包领取 (群聊内)
// =====================================================

// handleClaim 处理红包领取
func (h *RedPacketHandler) handleClaim(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, packetIDStr string) {
	packetID, err := strconv.ParseUint(packetIDStr, 10, 64)
	if err != nil {
		return
	}

	// 获取领取者用户信息
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.AnswerCallbackWithAlert(ctx, b, "", "请先使用 /start 注册")
		return
	}

	// 领取红包
	result, err := h.redPacketUC.ClaimRedPacket(ctx, packetID, user.ID)
	if err != nil {
		errMsg := "领取失败"
		if strings.Contains(err.Error(), "已经领取") {
			errMsg = "你已经领取过了"
		} else if strings.Contains(err.Error(), "已领完") {
			errMsg = "红包已领完"
		} else if strings.Contains(err.Error(), "已过期") {
			errMsg = "红包已过期"
		} else if strings.Contains(err.Error(), "不可领取") {
			errMsg = "红包不可领取"
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		return
	}

	// 更新群聊中红包消息的显示
	h.updateRedPacketMessage(ctx, b, chatID, messageID, result.Packet)
}

// updateRedPacketMessage 更新群聊中红包消息显示
func (h *RedPacketHandler) updateRedPacketMessage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, packet *entity.RedPacket) {
	// 查询发送者信息
	sender, err := h.redPacketUC.GetUserByID(ctx, packet.UserID)
	senderName := "匿名用户"
	if err == nil && sender != nil {
		senderName = sender.DisplayName()
	}

	typeName := app.RedPacketTypeName(packet.Type)
	text := fmt.Sprintf(
		"🧧 <b>%s 的红包</b>\n\n"+
			"%s %s | %s\n"+
			"%d/%d 个已领取",
		senderName, packet.Currency, packet.TotalAmount.String(), typeName,
		packet.ClaimedCount, packet.TotalCount,
	)

	// 添加备注
	if packet.Remark != "" {
		text += fmt.Sprintf("\n💬 %s", packet.Remark)
	}

	if packet.IsFullyClaimed() || packet.Status == entity.RedPacketStatusClaimed {
		// 已领完，显示领取记录
		claims, err := h.redPacketUC.GetClaimsByPacketID(ctx, packet.ID)
		if err == nil && len(claims) > 0 && !packet.IsPrivate {
			text += "\n\n<b>领取记录:</b>\n"
			for _, c := range claims {
				claimUser, err := h.redPacketUC.GetUserByID(ctx, c.UserID)
				claimName := "用户"
				if err == nil && claimUser != nil {
					claimName = claimUser.DisplayName()
				}
				text += fmt.Sprintf("%s: %s %s\n", claimName, c.Amount.String(), packet.Currency)
			}
		}

		// 已领完，不显示领取按钮
		tgpkg.EditMessage(ctx, b, chatID, messageID, text)
	} else {
		// 未领完，继续显示领取按钮
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("🧧 领取红包", fmt.Sprintf("RedPacket--claim_%d", packet.ID))).
			Build()

		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
	}
}

// handleClaimInline 处理 inline 模式红包领取 (使用 InlineMessageID)
func (h *RedPacketHandler) handleClaimInline(ctx context.Context, b *bot.Bot, inlineMessageID string, telegramID int64, packetIDStr string) {
	packetID, err := strconv.ParseUint(packetIDStr, 10, 64)
	if err != nil {
		return
	}

	// 获取领取者用户信息
	user := telegram.UserFromContext(ctx)
	if user == nil {
		return
	}

	// 领取红包
	result, err := h.redPacketUC.ClaimRedPacket(ctx, packetID, user.ID)
	if err != nil {
		h.logger.Debug("inline 红包领取失败",
			zap.String("inline_message_id", inlineMessageID),
			zap.Error(err),
		)
		// inline 消息无法直接向聊天发送错误消息 (不知 chatID)
		// 已在 HandleCallback 中 AnswerCallback，用户不会看到转圈
		return
	}

	// 更新 inline 消息显示
	h.updateRedPacketMessageInline(ctx, b, inlineMessageID, result.Packet)
}

// updateRedPacketMessageInline 更新 inline 模式红包消息 (使用 InlineMessageID)
func (h *RedPacketHandler) updateRedPacketMessageInline(ctx context.Context, b *bot.Bot, inlineMessageID string, packet *entity.RedPacket) {
	// 查询发送者信息
	sender, err := h.redPacketUC.GetUserByID(ctx, packet.UserID)
	senderName := "匿名用户"
	if err == nil && sender != nil {
		senderName = sender.DisplayName()
	}

	typeName := app.RedPacketTypeName(packet.Type)
	text := fmt.Sprintf(
		"🧧 <b>%s 的红包</b>\n\n"+
			"%s %s | %s\n"+
			"%d/%d 个已领取",
		senderName, packet.Currency, packet.TotalAmount.String(), typeName,
		packet.ClaimedCount, packet.TotalCount,
	)

	if packet.Remark != "" {
		text += fmt.Sprintf("\n💬 %s", packet.Remark)
	}

	if packet.IsFullyClaimed() || packet.Status == entity.RedPacketStatusClaimed {
		// 已领完，显示领取记录
		claims, claimErr := h.redPacketUC.GetClaimsByPacketID(ctx, packet.ID)
		if claimErr == nil && len(claims) > 0 && !packet.IsPrivate {
			text += "\n\n<b>领取记录:</b>\n"
			for _, c := range claims {
				claimUser, userErr := h.redPacketUC.GetUserByID(ctx, c.UserID)
				claimName := "用户"
				if userErr == nil && claimUser != nil {
					claimName = claimUser.DisplayName()
				}
				text += fmt.Sprintf("%s: %s %s\n", claimName, c.Amount.String(), packet.Currency)
			}
		}
		tgpkg.EditInlineMessage(ctx, b, inlineMessageID, text)
	} else {
		// 未领完，继续显示领取按钮
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("🧧 领取红包", fmt.Sprintf("RedPacket--claim_%d", packet.ID))).
			Build()
		tgpkg.EditInlineMessageWithKeyboard(ctx, b, inlineMessageID, text, keyboard)
	}
}

// =====================================================
// 封面管理
// =====================================================

// showCoverList 显示封面列表
func (h *RedPacketHandler) showCoverList(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	covers, err := h.redPacketUC.GetCoversByUser(ctx, user.ID)
	if err != nil {
		h.logger.Error("查询封面列表失败", zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := "🧧 红包封面\n\n"
	if len(covers) == 0 {
		text += "暂无封面，请先添加"
	} else {
		text += fmt.Sprintf("共 %d 个封面", len(covers))
	}

	kb := tgpkg.NewInlineKeyboard()

	for _, c := range covers {
		label := fmt.Sprintf("#%d (%s)", c.ID, c.FileType)
		kb.Row(tgpkg.Button(label, fmt.Sprintf("RedPacket--cover_select_%d", c.ID)))
	}

	kb.Row(tgpkg.Button("➕ 添加", "RedPacket--cover_add"))
	kb.Row(tgpkg.Button("↩️ 返回", "RedPacket--dashboard"))

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, kb.Build())
}

// promptCoverUpload 提示上传封面
func (h *RedPacketHandler) promptCoverUpload(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.SetStep(ctx, telegramID, "RedPacket", "cover_upload")

	text := "请发送你要设置的封面,支持格式: 图片、GIF、视频\n\n" +
		"<b>⚠️ 禁止枪支，军火、恐怖、血腥暴力、色情、毒品 等违规内容</b>"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("↩️ 返回", "RedPacket--cover")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleCoverUpload 处理封面上传
func (h *RedPacketHandler) handleCoverUpload(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, update *models.Update) {
	msg := update.Message
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	var fileID string
	var fileType string

	switch {
	case msg.Photo != nil && len(msg.Photo) > 0:
		// 取最大尺寸的图片
		fileID = msg.Photo[len(msg.Photo)-1].FileID
		fileType = "image"
	case msg.Animation != nil:
		fileID = msg.Animation.FileID
		fileType = "gif"
	case msg.Video != nil:
		fileID = msg.Video.FileID
		fileType = "video"
	default:
		tgpkg.SendMessage(ctx, b, chatID, "⚠️ 请发送图片、GIF 或视频")
		return
	}

	// 保存封面
	if err := h.redPacketUC.AddCover(ctx, user.ID, fileID, fileType); err != nil {
		h.logger.Error("保存封面失败", zap.Error(err))
		tgpkg.SendMessage(ctx, b, chatID, "❌ 保存封面失败，请稍后重试")
		return
	}

	_ = h.sessionSvc.Clear(ctx, telegramID)

	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("查看封面列表", "RedPacket--cover")).
		Row(tgpkg.Button("🔙 返回", "RedPacket--dashboard")).
		Build()

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, "✅ 封面保存成功!", keyboard)
}

// handleCoverSelect 选择封面
func (h *RedPacketHandler) handleCoverSelect(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, coverIDStr string) {
	coverID, err := strconv.ParseUint(coverIDStr, 10, 64)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 验证封面属于该用户
	covers, err := h.redPacketUC.GetCoversByUser(ctx, user.ID)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	var selectedCover *struct {
		FileID   string
		FileType string
	}
	for _, c := range covers {
		if c.ID == coverID {
			selectedCover = &struct {
				FileID   string
				FileType string
			}{FileID: c.FileID, FileType: c.FileType}
			break
		}
	}
	if selectedCover == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 保存到会话
	_ = h.sessionSvc.SetData(ctx, telegramID, "cover_file_id", selectedCover.FileID)

	text := fmt.Sprintf("✅ 已选择封面 #%d (%s)\n\n下次创建红包时将使用此封面", coverID, selectedCover.FileType)
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🔙 返回", "RedPacket--dashboard")).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// =====================================================
// 通用
// =====================================================

// handleCancel 取消操作，返回红包面板
func (h *RedPacketHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)
	h.showDashboard(ctx, b, chatID, messageID, telegramID, "active")
}

// =====================================================
// 工具函数
// =====================================================

// redPacketStatusText 红包状态中文
func redPacketStatusText(status int8) string {
	switch status {
	case entity.RedPacketStatusPending:
		return "待发送"
	case entity.RedPacketStatusSent:
		return "已发送"
	case entity.RedPacketStatusClaimed:
		return "已领完"
	case entity.RedPacketStatusExpired:
		return "已过期"
	case entity.RedPacketStatusCancelled:
		return "已关闭"
	default:
		return "未知"
	}
}

// RegisterRedPacketHandlers 注册红包相关的 handler 到 router
func RegisterRedPacketHandlers(router *telegram.Router, h *RedPacketHandler) {
	router.RegisterModule("RedPacket", h.HandleCallback)
}
