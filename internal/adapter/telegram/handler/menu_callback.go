package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// MenuCallbackHandler 菜单回调处理器
// 处理 Menu-- 前缀的所有回调
type MenuCallbackHandler struct {
	userUC     *app.UserUseCase
	sessionSvc *tgpkg.SessionService
	logger     *zap.Logger
}

// NewMenuCallbackHandler 创建菜单回调处理器
func NewMenuCallbackHandler(userUC *app.UserUseCase, sessionSvc *tgpkg.SessionService, logger *zap.Logger) *MenuCallbackHandler {
	return &MenuCallbackHandler{
		userUC:     userUC,
		sessionSvc: sessionSvc,
		logger:     logger,
	}
}

// Handle 处理 Menu-- 回调
func (h *MenuCallbackHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID

	// 立即响应回调
	tgpkg.AnswerCallback(ctx, b, query.ID)

	_, action := telegram.ParseCallbackData(query.Data)

	switch action {
	case "main":
		h.showMainMenu(ctx, b, chatID, messageID, query.From.ID)
	case "simple":
		h.showSimpleMenu(ctx, b, chatID, messageID, query.From.ID)
	case "cancel":
		h.handleCancel(ctx, b, chatID, messageID, query.From.ID)
	case "back":
		h.showMainMenu(ctx, b, chatID, messageID, query.From.ID)
	default:
		h.logger.Warn("未知菜单回调", zap.String("action", action))
	}
}

// showMainMenu 显示完整主菜单 (编辑原消息)
func (h *MenuCallbackHandler) showMainMenu(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	// 清除会话状态
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// showSimpleMenu 显示精简菜单 (编辑原消息)
func (h *MenuCallbackHandler) showSimpleMenu(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildSimpleMenuKeyboard().Build()
	tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, keyboard)
}

// handleCancel 取消当前操作，返回主菜单
func (h *MenuCallbackHandler) handleCancel(ctx context.Context, b *bot.Bot, chatID int64, messageID int, telegramID int64) {
	_ = h.sessionSvc.Clear(ctx, telegramID)
	h.showMainMenu(ctx, b, chatID, messageID, telegramID)
}
