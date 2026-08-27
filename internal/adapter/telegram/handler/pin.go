package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/app"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// PIN 长度: 4 位数字
const pinLength = 4

// PINHandler PIN 相关回调处理 (内联数字键盘)
type PINHandler struct {
	userUC     *app.UserUseCase
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewPINHandler 创建 PIN 处理器
func NewPINHandler(userUC *app.UserUseCase, sessionSvc *tgpkg.SessionService, logger *zap.Logger) *PINHandler {
	return &PINHandler{
		userUC:     userUC,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理 Pin--input_pin--{value} 回调
// value: "0"-"9", "del", "exit"
func (h *PINHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID
	telegramID := query.From.ID

	// 立即响应回调，避免按钮转圈
	tgpkg.AnswerCallback(ctx, b, query.ID)

	// 解析 callback_data: Pin--input_pin--{value}
	_, action := telegram.ParseCallbackData(query.Data)
	if !strings.HasPrefix(action, "input_pin--") {
		h.logger.Warn("未知 PIN 回调", zap.String("action", action))
		return
	}

	value := strings.TrimPrefix(action, "input_pin--")

	// 读取 session
	module, step := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "PIN" {
		return
	}

	switch value {
	case "exit":
		h.handleExit(ctx, b, chatID, messageID, telegramID)
	case "del":
		h.handleDelete(ctx, b, chatID, messageID, telegramID, step)
	default:
		// 数字 0-9
		if len(value) == 1 && value[0] >= '0' && value[0] <= '9' {
			h.handleDigitInput(ctx, b, chatID, messageID, telegramID, step, value)
		}
	}
}

// HandleTextMessage 处理文本消息中的 PIN 输入
// PIN 已改为内联键盘输入，文本输入不再处理 PIN
// 返回 false 表示不是 PIN 流程
func (h *PINHandler) HandleTextMessage(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.From == nil {
		return false
	}

	telegramID := update.Message.From.ID

	// 检查会话状态
	module, _ := h.sessionSvc.GetStep(ctx, telegramID)
	if module != "PIN" {
		return false
	}

	// PIN 已改为内联键盘输入，提示用户使用键盘
	chatID := update.Message.Chat.ID
	messageID := update.Message.ID

	// 删除用户的文本消息 (安全)
	go func() {
		tgpkg.DeleteMessage(context.Background(), b, chatID, messageID)
	}()

	tgpkg.SendMessage(ctx, b, chatID, "请使用数字键盘输入支付密码")
	return true
}

// StartPINSetup 发起 PIN 设置流程 (供其他模块调用)
// messageID > 0 时使用 EditMessage，否则发送新消息
// callbackModule/callbackAction: PIN 设置完成后回到的模块和动作
func (h *PINHandler) StartPINSetup(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, callbackModule, callbackAction string) {
	// 设置 session
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "set_input", map[string]interface{}{
		"pin_digits":          "",
		"pin_first":           "",
		"pin_callback_module": callbackModule,
		"pin_callback_action": callbackAction,
	})

	text := buildPINDisplay("", "set")
	keyboard := BuildPINKeyboard()

	if messageID > 0 {
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		// 保存消息 ID 到 session，供后续编辑使用
		_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(messageID))
	} else {
		msg, err := tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
		if err == nil && msg != nil {
			_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(msg.ID))
		}
	}
}

// StartPINVerify 发起 PIN 验证流程 (供其他模块调用)
// messageID > 0 时使用 EditMessage，否则发送新消息
// callbackModule/callbackAction: PIN 验证通过后恢复到的模块和步骤
func (h *PINHandler) StartPINVerify(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, callbackModule, callbackAction string) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "verify", map[string]interface{}{
		"pin_digits":          "",
		"pin_callback_module": callbackModule,
		"pin_callback_action": callbackAction,
	})

	text := buildPINDisplay("", "verify")
	keyboard := BuildPINKeyboard()

	if messageID > 0 {
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(messageID))
	} else {
		msg, err := tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
		if err == nil && msg != nil {
			_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(msg.ID))
		}
	}
}

// StartPINChange 发起 PIN 修改流程
// messageID > 0 时使用 EditMessage，否则发送新消息
func (h *PINHandler) StartPINChange(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "change_old", map[string]interface{}{
		"pin_digits": "",
	})

	text := buildPINDisplay("", "change_old")
	keyboard := BuildPINKeyboard()

	if messageID > 0 {
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(messageID))
	} else {
		msg, err := tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
		if err == nil && msg != nil {
			_ = h.sessionSvc.SetData(ctx, telegramID, "pin_message_id", strconv.Itoa(msg.ID))
		}
	}
}

// handleExit 处理取消按钮
func (h *PINHandler) handleExit(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	// 读取原始操作信息，用于返回按钮
	callbackModule := h.sessionSvc.GetDataString(ctx, telegramID, "pin_callback_module")

	// 清除 session
	_ = h.sessionSvc.Clear(ctx, telegramID)

	text := "已取消操作"
	var keyboard *models.InlineKeyboardMarkup

	if callbackModule != "" {
		// 有原始操作，提供主菜单按钮
		keyboard = tgpkg.NewInlineKeyboard().
			Row(tgpkg.MainMenuButton()).
			Build()
	} else {
		keyboard = tgpkg.NewInlineKeyboard().
			Row(tgpkg.MainMenuButton()).
			Build()
	}

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleDelete 处理退格键
func (h *PINHandler) handleDelete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, step string) {
	digits := h.sessionSvc.GetDataString(ctx, telegramID, "pin_digits")
	if len(digits) == 0 {
		return
	}

	// 删除最后一位
	digits = digits[:len(digits)-1]
	_ = h.sessionSvc.SetData(ctx, telegramID, "pin_digits", digits)

	// 更新显示
	mode := stepToMode(step)
	text := buildPINDisplay(digits, mode)
	keyboard := BuildPINKeyboard()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleDigitInput 处理数字输入
func (h *PINHandler) handleDigitInput(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, step, digit string) {
	digits := h.sessionSvc.GetDataString(ctx, telegramID, "pin_digits")

	// 防止超过 PIN 长度
	if len(digits) >= pinLength {
		return
	}

	// 追加数字
	digits += digit
	_ = h.sessionSvc.SetData(ctx, telegramID, "pin_digits", digits)

	// 未满 4 位，更新显示
	if len(digits) < pinLength {
		mode := stepToMode(step)
		text := buildPINDisplay(digits, mode)
		keyboard := BuildPINKeyboard()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 已满 4 位，根据 step 执行对应操作
	h.onPINComplete(ctx, b, chatID, messageID, telegramID, step, digits)
}

// onPINComplete 4 位 PIN 输入完成后的分支逻辑
func (h *PINHandler) onPINComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, step, digits string) {
	switch step {
	case "set_input":
		h.onSetInputComplete(ctx, b, chatID, messageID, telegramID, digits)
	case "set_confirm":
		h.onSetConfirmComplete(ctx, b, chatID, messageID, telegramID, digits)
	case "verify":
		h.onVerifyComplete(ctx, b, chatID, messageID, telegramID, digits)
	case "change_old":
		h.onChangeOldComplete(ctx, b, chatID, messageID, telegramID, digits)
	case "change_new":
		h.onChangeNewComplete(ctx, b, chatID, messageID, telegramID, digits)
	case "change_confirm":
		h.onChangeConfirmComplete(ctx, b, chatID, messageID, telegramID, digits)
	}
}

// onSetInputComplete 设置 PIN 第一步完成: 保存并进入确认步骤
func (h *PINHandler) onSetInputComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "set_confirm", map[string]interface{}{
		"pin_first":  digits,
		"pin_digits": "",
	})

	text := buildPINDisplay("", "confirm")
	keyboard := BuildPINKeyboard()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// onSetConfirmComplete 设置 PIN 第二步完成: 对比两次输入
func (h *PINHandler) onSetConfirmComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	firstPIN := h.sessionSvc.GetDataString(ctx, telegramID, "pin_first")

	if firstPIN != digits {
		// 两次不匹配，重新开始
		_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "set_input", map[string]interface{}{
			"pin_digits": "",
			"pin_first":  "",
		})

		text := "⚠️ 两次输入不一致，请重新设置\n\n" + buildPINDisplay("", "set")
		keyboard := BuildPINKeyboard()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 获取用户
	user := telegram.UserFromContext(ctx)
	var userID uint64
	if user != nil {
		userID = user.ID
	} else {
		result, err := h.userUC.RegisterOrGet(ctx, telegramID, "", "", "", "", false)
		if err != nil {
			h.logger.Error("PIN 设置获取用户失败", zap.Error(err))
			tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
			return
		}
		userID = result.User.ID
	}

	// 保存 PIN
	if err := h.userUC.SetPIN(ctx, userID, digits); err != nil {
		h.logger.Error("PIN 设置失败", zap.Uint64("user_id", userID), zap.Error(err))
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	h.logger.Info("用户设置 PIN 完成", zap.Uint64("user_id", userID))

	// 读取原始操作回调
	callbackModule := h.sessionSvc.GetDataString(ctx, telegramID, "pin_callback_module")
	callbackAction := h.sessionSvc.GetDataString(ctx, telegramID, "pin_callback_action")

	// 清除 session
	_ = h.sessionSvc.Clear(ctx, telegramID)

	// 设置成功，提供返回按钮
	text := "✅ 支付密码设置成功!"

	var keyboard *models.InlineKeyboardMarkup
	if callbackModule != "" && callbackAction != "" {
		// 有原始操作，提供继续操作的按钮
		keyboard = tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("继续操作", callbackModule+"--"+callbackAction)).
			Row(tgpkg.MainMenuButton()).
			Build()
	} else {
		keyboard = tgpkg.NewInlineKeyboard().
			Row(tgpkg.MainMenuButton()).
			Build()
	}

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// onVerifyComplete PIN 验证完成
func (h *PINHandler) onVerifyComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	ok, remaining, err := h.userUC.VerifyPIN(ctx, user.ID, digits)
	if err != nil {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "支付密码连续错误次数过多，账户已锁定")
		return
	}

	if !ok {
		if remaining <= 0 {
			_ = h.sessionSvc.Clear(ctx, telegramID)
			tgpkg.EditMessage(ctx, b, chatID, messageID, "支付密码连续错误次数过多，账户已锁定")
			return
		}

		// 重置输入，允许重试
		_ = h.sessionSvc.SetData(ctx, telegramID, "pin_digits", "")

		var warnText string
		if remaining <= 3 {
			warnText = fmt.Sprintf("⚠️ 支付密码错误，还可尝试 %d 次\n\n", remaining)
		} else {
			warnText = "⚠️ 支付密码错误，请重试\n\n"
		}

		text := warnText + buildPINDisplay("", "verify")
		keyboard := BuildPINKeyboard()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 验证通过，恢复到原模块
	callbackModule := h.sessionSvc.GetDataString(ctx, telegramID, "pin_callback_module")
	callbackAction := h.sessionSvc.GetDataString(ctx, telegramID, "pin_callback_action")

	if callbackModule != "" {
		// 恢复到原操作模块
		_ = h.sessionSvc.SetStep(ctx, telegramID, callbackModule, callbackAction)
	} else {
		_ = h.sessionSvc.Clear(ctx, telegramID)
	}

	// 编辑为验证通过提示
	text := "✅ 验证通过"
	if callbackModule != "" && callbackAction != "" {
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.Button("继续操作", callbackModule+"--"+callbackAction)).
			Build()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
	} else {
		keyboard := tgpkg.NewInlineKeyboard().
			Row(tgpkg.MainMenuButton()).
			Build()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
	}
}

// onChangeOldComplete 修改 PIN 第一步: 验证旧密码
func (h *PINHandler) onChangeOldComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	ok, remaining, err := h.userUC.VerifyPIN(ctx, user.ID, digits)
	if err != nil {
		_ = h.sessionSvc.Clear(ctx, telegramID)
		tgpkg.EditMessage(ctx, b, chatID, messageID, "支付密码连续错误次数过多，账户已锁定")
		return
	}

	if !ok {
		if remaining <= 0 {
			_ = h.sessionSvc.Clear(ctx, telegramID)
			tgpkg.EditMessage(ctx, b, chatID, messageID, "支付密码连续错误次数过多，账户已锁定")
			return
		}

		_ = h.sessionSvc.SetData(ctx, telegramID, "pin_digits", "")

		var warnText string
		if remaining <= 3 {
			warnText = fmt.Sprintf("⚠️ 当前密码错误，还可尝试 %d 次\n\n", remaining)
		} else {
			warnText = "⚠️ 当前密码错误，请重试\n\n"
		}

		text := warnText + buildPINDisplay("", "change_old")
		keyboard := BuildPINKeyboard()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	// 旧密码验证通过，进入新密码输入
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "change_new", map[string]interface{}{
		"pin_digits": "",
	})

	text := buildPINDisplay("", "change_new")
	keyboard := BuildPINKeyboard()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// onChangeNewComplete 修改 PIN 第二步: 输入新密码
func (h *PINHandler) onChangeNewComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "change_confirm", map[string]interface{}{
		"pin_first":  digits,
		"pin_digits": "",
	})

	text := buildPINDisplay("", "change_confirm")
	keyboard := BuildPINKeyboard()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// onChangeConfirmComplete 修改 PIN 第三步: 确认新密码
func (h *PINHandler) onChangeConfirmComplete(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64, digits string) {
	firstPIN := h.sessionSvc.GetDataString(ctx, telegramID, "pin_first")

	if firstPIN != digits {
		// 两次不匹配，重新输入新密码
		_ = h.sessionSvc.SetStepWithData(ctx, telegramID, "PIN", "change_new", map[string]interface{}{
			"pin_digits": "",
			"pin_first":  "",
		})

		text := "⚠️ 两次输入不一致，请重新设置\n\n" + buildPINDisplay("", "change_new")
		keyboard := BuildPINKeyboard()
		tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
		return
	}

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	if err := h.userUC.SetPIN(ctx, user.ID, digits); err != nil {
		h.logger.Error("PIN 修改失败", zap.Uint64("user_id", user.ID), zap.Error(err))
		tgpkg.EditMessage(ctx, b, chatID, messageID, "操作失败，请稍后重试")
		return
	}

	_ = h.sessionSvc.Clear(ctx, telegramID)

	h.logger.Info("用户修改 PIN 完成", zap.Uint64("user_id", user.ID))

	text := "✅ 支付密码修改成功!"
	keyboard := tgpkg.NewInlineKeyboard().
		Row(tgpkg.MainMenuButton()).
		Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// --- 键盘和显示构建 ---

// BuildPINKeyboard 构建 4x3 数字键盘
func BuildPINKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				pinButton("1"), pinButton("2"), pinButton("3"),
			},
			{
				pinButton("4"), pinButton("5"), pinButton("6"),
			},
			{
				pinButton("7"), pinButton("8"), pinButton("9"),
			},
			{
				{Text: "取消", CallbackData: "Pin--input_pin--exit"},
				pinButton("0"),
				{Text: "\u232b", CallbackData: "Pin--input_pin--del"},
			},
		},
	}
}

// pinButton 创建数字按钮
func pinButton(digit string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         digit,
		CallbackData: "Pin--input_pin--" + digit,
	}
}

// buildPINDisplay 构建 PIN 输入状态显示文本
// entered: 已输入的数字字符串 (如 "12")
// mode: "set" / "confirm" / "verify" / "change_old" / "change_new" / "change_confirm"
func buildPINDisplay(entered string, mode string) string {
	title := pinModeTitle(mode)

	// 构建可视化: * 代表已输入, _ 代表未输入
	var display strings.Builder
	for i := 0; i < pinLength; i++ {
		if i > 0 {
			display.WriteString(" ")
		}
		if i < len(entered) {
			display.WriteString("*")
		} else {
			display.WriteString("_")
		}
	}

	return fmt.Sprintf("%s\n────────────────────\n\n\U0001f511 %s", title, display.String())
}

// pinModeTitle 获取 PIN 模式标题
func pinModeTitle(mode string) string {
	switch mode {
	case "set":
		return "\U0001f530 设置支付密码"
	case "confirm":
		return "\U0001f530 确认支付密码"
	case "verify":
		return "\U0001f512 请输入支付密码"
	case "change_old":
		return "\U0001f512 请输入当前支付密码"
	case "change_new":
		return "\U0001f530 请输入新支付密码"
	case "change_confirm":
		return "\U0001f530 确认新支付密码"
	default:
		return "\U0001f512 请输入支付密码"
	}
}

// stepToMode 将 session step 转换为显示 mode
func stepToMode(step string) string {
	switch step {
	case "set_input":
		return "set"
	case "set_confirm":
		return "confirm"
	case "verify":
		return "verify"
	case "change_old":
		return "change_old"
	case "change_new":
		return "change_new"
	case "change_confirm":
		return "change_confirm"
	default:
		return "verify"
	}
}
