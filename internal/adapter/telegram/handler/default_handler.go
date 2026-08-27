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

// DefaultMessageHandler 默认消息处理器
// 处理非命令文本消息: 根据会话状态路由到对应模块，或返回精简菜单
type DefaultMessageHandler struct {
	pinHandler       *PINHandler
	withdrawHandler  *WithdrawHandler
	transferHandler  *TransferHandler
	receiveHandler   *ReceiveHandler
	exchangeHandler  *ExchangeHandler
	profileHandler   *ProfileHandler
	financeHandler   *FinanceHandler
	merchantHandler  *MerchantHandler
	redPacketHandler *RedPacketHandler
	userUC           *app.UserUseCase
	sessionSvc       *tgpkg.SessionService
	logger           *zap.Logger
}

// NewDefaultMessageHandler 创建默认消息处理器
func NewDefaultMessageHandler(
	pinHandler *PINHandler,
	withdrawHandler *WithdrawHandler,
	transferHandler *TransferHandler,
	receiveHandler *ReceiveHandler,
	exchangeHandler *ExchangeHandler,
	profileHandler *ProfileHandler,
	financeHandler *FinanceHandler,
	merchantHandler *MerchantHandler,
	redPacketHandler *RedPacketHandler,
	userUC *app.UserUseCase,
	sessionSvc *tgpkg.SessionService,
	logger *zap.Logger,
) *DefaultMessageHandler {
	return &DefaultMessageHandler{
		pinHandler:       pinHandler,
		withdrawHandler:  withdrawHandler,
		transferHandler:  transferHandler,
		receiveHandler:   receiveHandler,
		exchangeHandler:  exchangeHandler,
		profileHandler:   profileHandler,
		financeHandler:   financeHandler,
		merchantHandler:  merchantHandler,
		redPacketHandler: redPacketHandler,
		userUC:           userUC,
		sessionSvc:       sessionSvc,
		logger:           logger,
	}
}

// Handle 处理默认文本消息
func (h *DefaultMessageHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	// 非文本消息: 仅检查红包封面上传流程
	if update.Message.Text == "" {
		telegramID := update.Message.From.ID
		module, step := h.sessionSvc.GetStep(ctx, telegramID)
		if module == "RedPacket" && step == "cover_upload" {
			if h.redPacketHandler != nil && h.redPacketHandler.HandleTextMessage(ctx, b, update) {
				return
			}
		}
		return
	}

	// 1. 先检查 PIN 流程 (优先级最高)
	// PIN 已改为内联键盘输入，但文本输入时仍需拦截并提示
	if h.pinHandler.HandleTextMessage(ctx, b, update) {
		return
	}

	// 2. 检查其他会话模块
	telegramID := update.Message.From.ID
	module, _ := h.sessionSvc.GetStep(ctx, telegramID)

	switch module {
	case "Wallet":
		// 提币多步输入 (金额/地址)
		if h.withdrawHandler != nil && h.withdrawHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Transfer":
		if h.transferHandler != nil && h.transferHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Receive":
		if h.receiveHandler != nil && h.receiveHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Exchange":
		if h.exchangeHandler != nil && h.exchangeHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Profile":
		if h.profileHandler != nil && h.profileHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Finance":
		if h.financeHandler != nil && h.financeHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "Merchant":
		if h.merchantHandler != nil && h.merchantHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	case "RedPacket":
		if h.redPacketHandler != nil && h.redPacketHandler.HandleTextMessage(ctx, b, update) {
			return
		}
		h.showDefaultMenu(ctx, b, update)
		return
	}

	// 3. 无活跃会话，返回精简菜单
	h.showDefaultMenu(ctx, b, update)
}

// showDefaultMenu 显示默认精简菜单
// 所有用户直接显示钱包主页，不再检查 PIN
func (h *DefaultMessageHandler) showDefaultMenu(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	telegramID := update.Message.From.ID

	// 清除可能存在的过期会话
	_ = h.sessionSvc.Clear(ctx, telegramID)

	user := telegram.UserFromContext(ctx)
	if user == nil {
		// 未注册用户，先注册
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

	overview, err := h.userUC.GetWalletOverview(ctx, user.ID)
	if err != nil {
		tgpkg.SendError(ctx, b, chatID, "default")
		return
	}

	text := BuildWalletHomeText(overview.User, overview.Balances)
	keyboard := BuildSimpleMenuKeyboard().Build()
	tgpkg.SendMessageWithKeyboard(ctx, b, chatID, text, keyboard)
}
