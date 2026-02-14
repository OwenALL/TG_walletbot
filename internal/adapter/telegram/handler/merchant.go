package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// MerchantHandler 商户模块 Bot Handler
// Callback 前缀: Shop-- (参考原版格式)
type MerchantHandler struct {
	merchantUC *app.MerchantUseCase
	userUC     *app.UserUseCase
	pinHandler *PINHandler
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewMerchantHandler 创建商户处理器
func NewMerchantHandler(
	merchantUC *app.MerchantUseCase,
	userUC *app.UserUseCase,
	pinHandler *PINHandler,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *MerchantHandler {
	return &MerchantHandler{
		merchantUC: merchantUC,
		userUC:     userUC,
		pinHandler: pinHandler,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理 Shop-- 回调
// 路由规则:
//   - Shop--dashboard              → showDashboard (商户列表)
//   - Shop--create_shop            → startCreateShop (PIN验证后创建)
//   - Shop--shop_info--{id}        → showDetail (商户详情)
//   - Shop--settings--{id}--settings_name    → promptSetName
//   - Shop--settings--{id}--settings_webhook → promptSetWebhook
//   - Shop--settings--{id}--reset_token      → refreshSecret
//   - Shop--settings--{id}--settings_logo    → promptSetLogo
//   - Shop--settings_whitelist--{id}         → showIPWhitelist
//   - Shop--toggle_shop_status--{id}         → toggleStatus
//   - Shop--add_ip--{id}                    → promptAddIP
//   - Shop--remove_ip--{id}--{ip}           → removeIP
//   - Shop--pin_verified                    → onPINVerified (PIN验证通过后)
func (h *MerchantHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
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

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	switch {
	case action == "dashboard":
		h.showDashboard(ctx, b, chatID, messageID, user)

	case action == "create_shop":
		h.startCreateShop(ctx, b, chatID, messageID, telegramID, user)

	case action == "pin_verified":
		h.onPINVerified(ctx, b, chatID, messageID, telegramID, user)

	case strings.HasPrefix(action, "shop_info--"):
		merchantID := parseUint64FromAction(action, "shop_info--")
		if merchantID == 0 {
			tgpkg.EditError(ctx, b, chatID, messageID, "default")
			return
		}
		h.showDetail(ctx, b, chatID, messageID, user, merchantID)

	case strings.HasPrefix(action, "settings--"):
		h.handleSettings(ctx, b, chatID, messageID, telegramID, user, action)

	case strings.HasPrefix(action, "settings_whitelist--"):
		merchantID := parseUint64FromAction(action, "settings_whitelist--")
		if merchantID == 0 {
			tgpkg.EditError(ctx, b, chatID, messageID, "default")
			return
		}
		h.showIPWhitelist(ctx, b, chatID, messageID, user, merchantID)

	case strings.HasPrefix(action, "toggle_shop_status--"):
		merchantID := parseUint64FromAction(action, "toggle_shop_status--")
		if merchantID == 0 {
			tgpkg.EditError(ctx, b, chatID, messageID, "default")
			return
		}
		h.toggleStatus(ctx, b, chatID, messageID, user, merchantID)

	case strings.HasPrefix(action, "add_ip--"):
		merchantID := parseUint64FromAction(action, "add_ip--")
		if merchantID == 0 {
			tgpkg.EditError(ctx, b, chatID, messageID, "default")
			return
		}
		h.promptAddIP(ctx, b, chatID, messageID, telegramID, merchantID)

	case strings.HasPrefix(action, "remove_ip--"):
		h.handleRemoveIP(ctx, b, chatID, messageID, user, action)

	default:
		h.logger.Warn("未知商户回调", zap.String("action", action))
	}
}

// HandleTextMessage 处理商户模块的文本消息输入
// 返回 true 表示已处理
func (h *MerchantHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "Merchant" {
		return false
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return true
	}

	switch step {
	case "set_name":
		h.handleNameInput(ctx, b, chatID, telegramID, user, update.Message.Text)
	case "set_webhook":
		h.handleWebhookInput(ctx, b, chatID, telegramID, user, update.Message.Text)
	case "add_ip":
		h.handleIPInput(ctx, b, chatID, telegramID, user, update.Message.Text)
	default:
		return false
	}

	return true
}

// ==================== 商户列表 ====================

// showDashboard 显示商户列表页
func (h *MerchantHandler) showDashboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User) {
	merchants, err := h.merchantUC.ListByUserID(ctx, user.ID)
	if err != nil {
		h.logger.Error("查询商户列表失败", zap.Uint64("user_id", user.ID), zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	if len(merchants) == 0 {
		// 未创建商户
		text := "还未创建商户"
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("➕ 创建商户", "Shop--create_shop")).
			Row(tgpkg.Button("🏠 主菜单", "Menu--main")).
			Build()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 已有商户: 显示列表
	text := fmt.Sprintf("%s 🤖 商户列表", user.DisplayName())
	kb := tgpkg.NewInlineKeyboard()
	for _, m := range merchants {
		btnText := fmt.Sprintf("%d - %s", m.ID, m.BusinessName)
		cb := fmt.Sprintf("Shop--shop_info--%d", m.ID)
		kb.Row(tgpkg.Button(btnText, cb))
	}
	kb.Row(tgpkg.Button("➕ 创建商户", "Shop--create_shop"))
	kb.Row(tgpkg.Button("🏠 主菜单", "Menu--main"))
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, kb.Build())
}

// ==================== 创建商户 ====================

// startCreateShop 发起创建商户流程 (需 PIN 验证)
func (h *MerchantHandler) startCreateShop(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, user *entity.User) {
	if !user.HasPIN() {
		// 未设置 PIN，先设置
		h.pinHandler.StartPINSetup(ctx, b, chatID, messageID, telegramID, "Shop", "pin_verified")
		return
	}
	// 已设置 PIN，发起验证
	h.pinHandler.StartPINVerify(ctx, b, chatID, messageID, telegramID, "Shop", "pin_verified")
}

// onPINVerified PIN 验证通过后自动创建商户
func (h *MerchantHandler) onPINVerified(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, user *entity.User) {
	// 清除 PIN session 残留
	_ = h.sessionSvc.Clear(ctx, telegramID)

	merchant, err := h.merchantUC.Create(ctx, user.ID, user.DisplayName())
	if err != nil {
		h.logger.Error("创建商户失败",
			zap.Uint64("user_id", user.ID),
			zap.Error(err),
		)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "创建商户失败，请稍后重试")
		return
	}

	h.logger.Info("商户创建成功",
		zap.Uint64("user_id", user.ID),
		zap.Uint64("merchant_id", merchant.ID),
	)

	// 创建成功，显示商户详情
	h.showDetail(ctx, b, chatID, messageID, user, merchant.ID)
}

// ==================== 商户详情 ====================

// showDetail 显示商户详情页
func (h *MerchantHandler) showDetail(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User, merchantID uint64) {
	merchant, err := h.merchantUC.GetByID(ctx, merchantID)
	if err != nil {
		h.logger.Error("查询商户详情失败", zap.Uint64("merchant_id", merchantID), zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// 验证所有权
	if merchant.UserID != user.ID {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := buildMerchantDetailText(merchant, user)
	keyboard := buildMerchantDetailKeyboard(merchant)
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// buildMerchantDetailText 构建商户详情文本
func buildMerchantDetailText(m *entity.Merchant, user *entity.User) string {
	return fmt.Sprintf(
		"🔰 <b>商户详情</b>\n\n"+
			"名称 : %s\n"+
			"管理员 : %s\n"+
			"App ID : %d\n"+
			"密钥 : <code>%s</code>\n"+
			"⚠️ 请勿将密钥透露给别人可能会造成资金安全隐患\n"+
			"回调地址 : %s\n"+
			"状态 : %s\n"+
			"费率 : %s\n"+
			"创建时间 : %s\n"+
			"更新时间 : %s",
		m.BusinessName,
		user.DisplayName(),
		m.ID,
		m.APISecret,
		m.WebhookDisplay(),
		m.StatusEmoji(),
		m.FeeRateDisplay(),
		m.CreatedAt.Format("2006-01-02 15:04:05"),
		m.UpdatedAt.Format("2006-01-02 15:04:05"),
	)
}

// buildMerchantDetailKeyboard 构建商户详情页键盘
func buildMerchantDetailKeyboard(m *entity.Merchant) *models.InlineKeyboardMarkup {
	id := m.ID
	// 切换状态按钮文案
	toggleText := "❌ 关闭商户"
	if m.IsDisabled() {
		toggleText = "✅ 开启商户"
	}

	return tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("✍️ 设置名称", fmt.Sprintf("Shop--settings--%d--settings_name", id)),
			tgpkg.Button("🪝 设置回调地址", fmt.Sprintf("Shop--settings--%d--settings_webhook", id))).
		Row(tgpkg.Button("*️⃣ 刷新密钥", fmt.Sprintf("Shop--settings--%d--reset_token", id)),
			tgpkg.Button("🌌 设置LOGO", fmt.Sprintf("Shop--settings--%d--settings_logo", id))).
		Row(tgpkg.Button("👮\u200d♀️ 设置ip白名单", fmt.Sprintf("Shop--settings_whitelist--%d", id))).
		Row(tgpkg.Button(toggleText, fmt.Sprintf("Shop--toggle_shop_status--%d", id))).
		Row(tgpkg.MainMenuButton()).
		Build()
}

// ==================== 设置操作 ====================

// handleSettings 处理 Shop--settings--{id}--{setting_type} 回调
func (h *MerchantHandler) handleSettings(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, user *entity.User, action string) {
	// 格式: settings--{id}--{setting_type}
	parts := strings.SplitN(action, "--", 3)
	if len(parts) < 3 {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	merchantID := parseUint64Str(parts[1])
	settingType := parts[2]

	if merchantID == 0 {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	switch settingType {
	case "settings_name":
		h.promptSetName(ctx, b, chatID, messageID, telegramID, merchantID)
	case "settings_webhook":
		h.promptSetWebhook(ctx, b, chatID, messageID, telegramID, merchantID)
	case "reset_token":
		h.refreshSecret(ctx, b, chatID, messageID, user, merchantID)
	case "settings_logo":
		h.promptSetLogo(ctx, b, chatID, messageID, merchantID)
	default:
		h.logger.Warn("未知商户设置类型", zap.String("setting_type", settingType))
	}
}

// promptSetName 提示输入新名称
func (h *MerchantHandler) promptSetName(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, merchantID uint64) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Merchant", "set_name", map[string]interface{}{
		"merchant_id": merchantID,
	})

	text := "OK, 向我发送新的名称."
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", fmt.Sprintf("Shop--shop_info--%d", merchantID))).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// promptSetWebhook 提示输入新回调地址
func (h *MerchantHandler) promptSetWebhook(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, merchantID uint64) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Merchant", "set_webhook", map[string]interface{}{
		"merchant_id": merchantID,
	})

	text := "OK, 向我发送新的webhook网址."
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", fmt.Sprintf("Shop--shop_info--%d", merchantID))).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// refreshSecret 刷新密钥 (直接执行，无需二次确认)
func (h *MerchantHandler) refreshSecret(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User, merchantID uint64) {
	_, err := h.merchantUC.RefreshSecret(ctx, merchantID, user.ID)
	if err != nil {
		h.logger.Error("刷新商户密钥失败",
			zap.Uint64("merchant_id", merchantID),
			zap.Error(err),
		)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "刷新密钥失败，请稍后重试")
		return
	}

	// 刷新后直接返回详情页 (显示新密钥)
	h.showDetail(ctx, b, chatID, messageID, user, merchantID)
}

// promptSetLogo 提示上传 LOGO 图片
// TODO: 图片上传需要额外的 photo handler，暂时提示功能开发中
func (h *MerchantHandler) promptSetLogo(ctx context.Context, b *bot.Bot, chatID int64, messageID int, merchantID uint64) {
	text := "LOGO 设置功能开发中，敬请期待"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", fmt.Sprintf("Shop--shop_info--%d", merchantID))).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// ==================== 切换状态 ====================

// toggleStatus 切换商户开启/关闭状态
func (h *MerchantHandler) toggleStatus(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User, merchantID uint64) {
	_, err := h.merchantUC.ToggleStatus(ctx, merchantID, user.ID)
	if err != nil {
		h.logger.Error("切换商户状态失败",
			zap.Uint64("merchant_id", merchantID),
			zap.Error(err),
		)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	// 切换后返回详情页
	h.showDetail(ctx, b, chatID, messageID, user, merchantID)
}

// ==================== IP 白名单 ====================

// showIPWhitelist 显示 IP 白名单管理页
func (h *MerchantHandler) showIPWhitelist(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User, merchantID uint64) {
	merchant, err := h.merchantUC.GetByID(ctx, merchantID)
	if err != nil || merchant.UserID != user.ID {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	ips := merchant.GetIPList()

	var textBuilder strings.Builder
	textBuilder.WriteString("👮\u200d♀️ <b>IP 白名单</b>\n\n")

	if len(ips) == 0 {
		textBuilder.WriteString("暂无 IP 白名单\n\n")
		textBuilder.WriteString("添加 IP 后，仅允许白名单中的 IP 调用 API")
	} else {
		for i, ip := range ips {
			textBuilder.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, ip))
		}
	}

	kb := tgpkg.NewInlineKeyboard()

	// 每个 IP 一个删除按钮
	for _, ip := range ips {
		kb.Row(tgpkg.Button(fmt.Sprintf("🗑 删除 %s", ip), fmt.Sprintf("Shop--remove_ip--%d--%s", merchantID, ip)))
	}

	kb.Row(tgpkg.Button("➕ 添加 IP", fmt.Sprintf("Shop--add_ip--%d", merchantID)))
	kb.Row(tgpkg.Button("↩️ 返回", fmt.Sprintf("Shop--shop_info--%d", merchantID)))

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, textBuilder.String(), kb.Build())
}

// promptAddIP 提示输入 IP 地址
func (h *MerchantHandler) promptAddIP(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, merchantID uint64) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "Merchant", "add_ip", map[string]interface{}{
		"merchant_id": merchantID,
	})

	text := "OK, 向我发送要添加的 IP 地址."
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.Button("🚫 取消", fmt.Sprintf("Shop--settings_whitelist--%d", merchantID))).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleRemoveIP 处理删除 IP 回调
func (h *MerchantHandler) handleRemoveIP(ctx context.Context, b *bot.Bot, chatID int64, messageID int, user *entity.User, action string) {
	// 格式: remove_ip--{id}--{ip}
	parts := strings.SplitN(action, "--", 3)
	if len(parts) < 3 {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	merchantID := parseUint64Str(parts[1])
	ip := parts[2]

	if merchantID == 0 || ip == "" {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	_, err := h.merchantUC.RemoveIPWhitelist(ctx, merchantID, user.ID, ip)
	if err != nil {
		h.logger.Error("删除 IP 白名单失败",
			zap.Uint64("merchant_id", merchantID),
			zap.String("ip", ip),
			zap.Error(err),
		)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "删除 IP 失败，请稍后重试")
		return
	}

	// 删除成功，刷新白名单页
	h.showIPWhitelist(ctx, b, chatID, messageID, user, merchantID)
}

// ==================== 文本输入处理 ====================

// handleNameInput 处理名称输入
func (h *MerchantHandler) handleNameInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, user *entity.User, text string) {
	merchantID := h.getMerchantIDFromSession(ctx, telegramID)
	if merchantID == 0 {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 验证名称长度
	text = strings.TrimSpace(text)
	if len(text) == 0 || len(text) > 128 {
		tgpkg.SendMessage(ctx, b, chatID, "商户名称长度应在 1-128 个字符之间，请重新输入")
		return
	}

	_, err := h.merchantUC.UpdateName(ctx, merchantID, user.ID, text)
	if err != nil {
		h.logger.Error("更新商户名称失败",
			zap.Uint64("merchant_id", merchantID),
			zap.Error(err),
		)
		tgpkg.SendMessage(ctx, b, chatID, "更新名称失败，请稍后重试")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	// 清除 session
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送新消息显示商户详情
	h.sendDetailAsNewMessage(ctx, b, chatID, user, merchantID)
}

// handleWebhookInput 处理 webhook 地址输入
func (h *MerchantHandler) handleWebhookInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, user *entity.User, text string) {
	merchantID := h.getMerchantIDFromSession(ctx, telegramID)
	if merchantID == 0 {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	text = strings.TrimSpace(text)

	_, err := h.merchantUC.UpdateWebhook(ctx, merchantID, user.ID, text)
	if err != nil {
		h.logger.Error("更新商户回调地址失败",
			zap.Uint64("merchant_id", merchantID),
			zap.Error(err),
		)
		// 提取友好错误信息
		errMsg := "更新回调地址失败，请稍后重试"
		if strings.Contains(err.Error(), "回调地址格式不正确") {
			errMsg = "回调地址格式不正确，请输入有效的 HTTP/HTTPS 地址"
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		return
	}

	// 清除 session
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送新消息显示商户详情
	h.sendDetailAsNewMessage(ctx, b, chatID, user, merchantID)
}

// handleIPInput 处理 IP 地址输入
func (h *MerchantHandler) handleIPInput(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64, user *entity.User, text string) {
	merchantID := h.getMerchantIDFromSession(ctx, telegramID)
	if merchantID == 0 {
		tgpkg.SendError(ctx, b, chatID, "default")
		_ = h.sessionSvc.Clear(ctx, telegramID)
		return
	}

	ip := strings.TrimSpace(text)

	_, err := h.merchantUC.AddIPWhitelist(ctx, merchantID, user.ID, ip)
	if err != nil {
		h.logger.Error("添加 IP 白名单失败",
			zap.Uint64("merchant_id", merchantID),
			zap.String("ip", ip),
			zap.Error(err),
		)
		errMsg := "添加 IP 失败，请稍后重试"
		if strings.Contains(err.Error(), "IP 地址格式不正确") {
			errMsg = "IP 地址格式不正确，请重新输入"
		} else if strings.Contains(err.Error(), "已在白名单中") {
			errMsg = "该 IP 已在白名单中"
		}
		tgpkg.SendMessage(ctx, b, chatID, errMsg)
		return
	}

	// 清除 session
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 发送新消息显示 IP 白名单页
	h.sendIPWhitelistAsNewMessage(ctx, b, chatID, user, merchantID)
}

// ==================== 辅助方法 ====================

// getMerchantIDFromSession 从 session 中获取当前操作的商户 ID
func (h *MerchantHandler) getMerchantIDFromSession(ctx context.Context, telegramID int64) uint64 {
	v, ok := h.sessionSvc.GetData(ctx, telegramID, "merchant_id")
	if !ok {
		return 0
	}
	switch id := v.(type) {
	case float64:
		return uint64(id)
	case uint64:
		return id
	case int64:
		return uint64(id)
	default:
		return 0
	}
}

// sendDetailAsNewMessage 以新消息形式发送商户详情 (文本输入后使用)
func (h *MerchantHandler) sendDetailAsNewMessage(ctx context.Context, b *bot.Bot, chatID int64, user *entity.User, merchantID uint64) {
	merchant, err := h.merchantUC.GetByID(ctx, merchantID)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}
	if merchant.UserID != user.ID {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	text := buildMerchantDetailText(merchant, user)
	keyboard := buildMerchantDetailKeyboard(merchant)
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// sendIPWhitelistAsNewMessage 以新消息形式发送 IP 白名单页 (文本输入后使用)
func (h *MerchantHandler) sendIPWhitelistAsNewMessage(ctx context.Context, b *bot.Bot, chatID int64, user *entity.User, merchantID uint64) {
	merchant, err := h.merchantUC.GetByID(ctx, merchantID)
	if err != nil || merchant.UserID != user.ID {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	ips := merchant.GetIPList()

	var textBuilder strings.Builder
	textBuilder.WriteString("👮\u200d♀️ <b>IP 白名单</b>\n\n")

	if len(ips) == 0 {
		textBuilder.WriteString("暂无 IP 白名单\n\n")
		textBuilder.WriteString("添加 IP 后，仅允许白名单中的 IP 调用 API")
	} else {
		for i, ip := range ips {
			textBuilder.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, ip))
		}
	}

	kb := tgpkg.NewInlineKeyboard()
	for _, ip := range ips {
		kb.Row(tgpkg.Button(fmt.Sprintf("🗑 删除 %s", ip), fmt.Sprintf("Shop--remove_ip--%d--%s", merchantID, ip)))
	}
	kb.Row(tgpkg.Button("➕ 添加 IP", fmt.Sprintf("Shop--add_ip--%d", merchantID)))
	kb.Row(tgpkg.Button("↩️ 返回", fmt.Sprintf("Shop--shop_info--%d", merchantID)))

	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, textBuilder.String(), kb.Build())
}

// parseUint64FromAction 从 action 字符串中解析 uint64 ID
// 例如 "shop_info--123" => prefix="shop_info--", 返回 123
func parseUint64FromAction(action, prefix string) uint64 {
	s := strings.TrimPrefix(action, prefix)
	// 可能还有后续 -- 分隔符，取第一段
	parts := strings.SplitN(s, "--", 2)
	return parseUint64Str(parts[0])
}

// parseUint64Str 将字符串转为 uint64
func parseUint64Str(s string) uint64 {
	var id uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}

// RegisterMerchantHandlers 注册商户模块到路由器
// 使用 Shop 前缀参考原版 callback data 格式
func RegisterMerchantHandlers(router *telegram.Router, handler *MerchantHandler) {
	router.RegisterModule("Shop", handler.HandleCallback)
}
