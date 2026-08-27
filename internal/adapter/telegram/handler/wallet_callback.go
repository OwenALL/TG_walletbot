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

// WalletCallbackHandler 钱包模块回调路由器
// 将 Wallet-- 前缀的回调分发到 DepositHandler 或 WithdrawHandler
// 入口处检查 PIN 保护: 未设置 PIN 的用户会被引导设置 PIN
type WalletCallbackHandler struct {
	depositHandler  *DepositHandler
	withdrawHandler *WithdrawHandler
	pinHandler      *PINHandler
	userUC          *app.UserUseCase
	logger          *zap.Logger
}

// NewWalletCallbackHandler 创建钱包回调路由器
func NewWalletCallbackHandler(
	depositHandler *DepositHandler,
	withdrawHandler *WithdrawHandler,
	pinHandler *PINHandler,
	userUC *app.UserUseCase,
	logger *zap.Logger,
) *WalletCallbackHandler {
	return &WalletCallbackHandler{
		depositHandler:  depositHandler,
		withdrawHandler: withdrawHandler,
		pinHandler:      pinHandler,
		userUC:          userUC,
		logger:          logger,
	}
}

// Handle 处理 Wallet-- 回调
// 入口处检查 PIN 保护: 用户未设置 PIN 时弹出 PIN 设置键盘
func (h *WalletCallbackHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	chatID := query.Message.Message.Chat.ID
	messageID := query.Message.Message.ID
	telegramID := query.From.ID

	_, action := telegram.ParseCallbackData(query.Data)

	// 获取用户
	user := telegram.UserFromContext(ctx)
	if user == nil {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		tgpkg.EditError(ctx, b, chatID, messageID, "default")
		return
	}

	// PIN 保护检查: 充值和提币操作需要设置 PIN
	if !user.HasPIN() {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		h.pinHandler.StartPINSetup(ctx, b, chatID, messageID, telegramID, "Wallet", action)
		return
	}

	// 分发到具体处理器
	switch {
	case action == "tronlink_deposit":
		h.depositHandler.HandleCallback(ctx, b, update)
	case strings.HasPrefix(action, "withdraw"):
		h.withdrawHandler.HandleCallback(ctx, b, update)
	default:
		h.logger.Warn("未知钱包回调", zap.String("action", action))
	}
}

// RegisterDepositWithdrawHandlers 注册充值/提币模块到路由器
// 由 main.go 调用，将 Wallet 模块注册到路由器
func RegisterDepositWithdrawHandlers(router *telegram.Router, handler *WalletCallbackHandler) {
	router.RegisterModule("Wallet", handler.Handle)
}
