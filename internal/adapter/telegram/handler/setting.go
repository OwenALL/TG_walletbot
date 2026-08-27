package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/app"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// SettingHandler 设置模块 Bot Handler
type SettingHandler struct {
	userUC     *app.UserUseCase
	profileUC  *app.ProfileUseCase
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewSettingHandler 创建设置处理器
func NewSettingHandler(
	userUC *app.UserUseCase,
	profileUC *app.ProfileUseCase,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *SettingHandler {
	return &SettingHandler{
		userUC:     userUC,
		profileUC:  profileUC,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// HandleCallback 处理 Setting-- 回调
func (h *SettingHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID

	_, action := telegram.ParseCallbackData(query.Data)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	switch action {
	case "language":
		tgpkg.AnswerCallback(ctx, b, query.ID)
		h.showLanguageSelect(ctx, b, chatID, messageID)
	case "lang_zh":
		h.switchLanguage(ctx, b, query.ID, chatID, messageID, user.ID, "zh-CN")
	case "lang_en":
		h.switchLanguage(ctx, b, query.ID, chatID, messageID, user.ID, "en")
	default:
		tgpkg.AnswerCallback(ctx, b, query.ID)
		h.logger.Warn("未知设置回调", zap.String("action", action))
	}
}

// showLanguageSelect 显示语言选择界面
func (h *SettingHandler) showLanguageSelect(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	text := "请选择语言 / Please select language"

	keyboard := tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("中文", "Setting--lang_zh"),
			tgpkg.Button("English", "Setting--lang_en"),
		).
		Row(tgpkg.MainMenuButton()).
		Build()

	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// switchLanguage 切换语言并回到主菜单
func (h *SettingHandler) switchLanguage(ctx context.Context, b *bot.Bot, callbackID string, chatID int64, messageID int, userID uint64, language string) {
	// 更新语言设置
	if err := h.profileUC.UpdateLanguage(ctx, userID, language); err != nil {
		h.logger.Error("更新语言设置失败",
			zap.Uint64("user_id", userID),
			zap.String("language", language),
			zap.Error(err),
		)
		tgpkg.AnswerCallbackWithText(ctx, b, callbackID, "操作失败，请稍后重试")
		return
	}

	// Toast 提示
	var toastText string
	if language == "zh-CN" {
		toastText = "语言已切换为中文"
	} else {
		toastText = "Language changed to English"
	}
	tgpkg.AnswerCallbackWithText(ctx, b, callbackID, toastText)

	// 回到主菜单: 获取钱包概览并显示
	overview, err := h.userUC.GetWalletOverview(ctx, userID)
	if err != nil {
		h.logger.Error("获取钱包概览失败", zap.Uint64("user_id", userID), zap.Error(err))
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// RegisterSettingHandlers 注册设置模块到路由器
func RegisterSettingHandlers(router *telegram.Router, handler *SettingHandler) {
	router.RegisterModule("Setting", handler.HandleCallback)
}
