package handler

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/adapter/telegram"
	"github.com/OwenALL/TG_walletbot/internal/app"
	tgpkg "github.com/OwenALL/TG_walletbot/pkg/telegram"
)

// StartHandler /start 命令处理器
type StartHandler struct {
	userUC          *app.UserUseCase
	sessionSvc      *tgpkg.SessionService
	inlinePayHandler *InlinePayHandler
	logger          *zap.Logger
}

// NewStartHandler 创建 /start 处理器
func NewStartHandler(userUC *app.UserUseCase, sessionSvc *tgpkg.SessionService, inlinePayHandler *InlinePayHandler, logger *zap.Logger) *StartHandler {
	return &StartHandler{
		userUC:          userUC,
		sessionSvc:      sessionSvc,
		inlinePayHandler: inlinePayHandler,
		logger:          logger,
	}
}

// Handle 处理 /start 命令
// 支持深度链接分发:
//   - /start pay--xxx → Inline 收款支付流程
//   - /start wallet   → 钱包主页
//   - /start withdraw → 提现 (预留)
//   - /start (无参数) → 钱包主页
func (h *StartHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID
	tgUser := update.Message.From

	// 注册或获取用户
	result, err := h.userUC.RegisterOrGet(
		ctx,
		tgUser.ID,
		tgUser.Username,
		tgUser.FirstName,
		tgUser.LastName,
		tgUser.LanguageCode,
		tgUser.IsPremium,
	)
	if err != nil {
		h.logger.Error("用户注册/获取失败",
			zap.Int64("telegram_id", tgUser.ID),
			zap.Error(err),
		)
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	// 清除可能存在的过期会话
	_ = h.sessionSvc.Clear(ctx, tgUser.ID)

	// 解析深度链接参数: /start <param>
	startParam := extractStartParam(update.Message.Text)

	// 分发深度链接
	if strings.HasPrefix(startParam, "pay--") {
		// Inline 收款支付: /start pay--<base64url(tgID amount currency)>
		h.inlinePayHandler.HandleDeepLink(ctx, b, chatID, tgUser.ID, result.User, startParam)
		return
	}

	// 默认: 显示钱包主页
	h.showWalletHome(ctx, b, chatID, result.User.ID)
}

// extractStartParam 从 /start 命令文本中提取深度链接参数
// "/start pay--abc" → "pay--abc"
// "/start" → ""
func extractStartParam(text string) string {
	parts := strings.SplitN(strings.TrimSpace(text), " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// showWalletHome 显示钱包主页 (发送新消息)
func (h *StartHandler) showWalletHome(ctx context.Context, b *bot.Bot, chatID int64, userID uint64) {
	overview, err := h.userUC.GetWalletOverview(ctx, userID)
	if err != nil {
		h.logger.Error("获取钱包概览失败", zap.Uint64("user_id", userID), zap.Error(err))
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildMainMenuKeyboard().Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}

// SimpleMenuHandler 处理 /settings /menu /wallet /balance 等命令
// 显示精简菜单
type SimpleMenuHandler struct {
	userUC    *app.UserUseCase
	sessionSvc *tgpkg.SessionService
	logger    *zap.Logger
}

// NewSimpleMenuHandler 创建精简菜单处理器
func NewSimpleMenuHandler(userUC *app.UserUseCase, sessionSvc *tgpkg.SessionService, logger *zap.Logger) *SimpleMenuHandler {
	return &SimpleMenuHandler{
		userUC:    userUC,
		sessionSvc: sessionSvc,
		logger:    logger,
	}
}

// Handle 处理精简菜单命令
// 所有用户直接显示钱包主页，不再检查 PIN
func (h *SimpleMenuHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	chatID := update.Message.Chat.ID
	user := telegram.UserFromContext(ctx)

	// 未注册用户交给注册逻辑
	if user == nil {
		tgUser := update.Message.From
		result, err := h.userUC.RegisterOrGet(
			ctx, tgUser.ID, tgUser.Username, tgUser.FirstName, tgUser.LastName, tgUser.LanguageCode, tgUser.IsPremium,
		)
		if err != nil {
			tgpkg.SendError(ctx, b, chatID, "default")
			return
		}
		user = result.User
	}

	// 清除会话状态
	_ = h.sessionSvc.Clear(ctx, update.Message.From.ID)

	// 显示钱包主页 + 精简菜单
	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildSimpleMenuKeyboard().Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}
