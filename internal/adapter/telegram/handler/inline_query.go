package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/adapter/telegram"
	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// InlineQueryHandler 处理 Telegram Inline Query 及关联的转账/收款回调
type InlineQueryHandler struct {
	redPacketUC *app.RedPacketUseCase
	transferUC  *app.TransferUseCase
	userUC      *app.UserUseCase
	logger      *zap.Logger
	botUsername  string   // Bot 用户名，用于构建深度链接 URL
	botOnce     sync.Once // 确保只调用一次 GetMe
}

// NewInlineQueryHandler 创建 Inline Query 处理器
func NewInlineQueryHandler(
	redPacketUC *app.RedPacketUseCase,
	transferUC *app.TransferUseCase,
	userUC *app.UserUseCase,
	logger *zap.Logger,
) *InlineQueryHandler {
	return &InlineQueryHandler{
		redPacketUC: redPacketUC,
		transferUC:  transferUC,
		userUC:      userUC,
		logger:      logger,
	}
}

// RegisterInlineCallbackHandlers 注册 Inline 转账回调模块
// 收款 (IR) 已改为深度链接支付，由 InlinePayHandler 处理
func RegisterInlineCallbackHandlers(router *telegram.Router, h *InlineQueryHandler) {
	router.RegisterModule("IT", h.HandleInlineTransferCallback)
}

// =====================================================
// 辅助: Bot 信息 & URL 构建
// =====================================================

// getBotUsername 获取 Bot 用户名 (懒加载，仅首次调用时请求 API)
func (h *InlineQueryHandler) getBotUsername(ctx context.Context, b *bot.Bot) string {
	h.botOnce.Do(func() {
		me, err := b.GetMe(ctx)
		if err == nil && me != nil {
			h.botUsername = me.Username
		} else {
			h.logger.Warn("获取 Bot 信息失败", zap.Error(err))
		}
	})
	return h.botUsername
}

// botDeepLink 构建 Bot 深度链接 URL (https://t.me/BOT?start=PARAM)
func (h *InlineQueryHandler) botDeepLink(ctx context.Context, b *bot.Bot, startParam string) string {
	username := h.getBotUsername(ctx, b)
	if username == "" {
		return ""
	}
	if startParam != "" {
		return fmt.Sprintf("https://t.me/%s?start=%s", username, startParam)
	}
	return fmt.Sprintf("https://t.me/%s", username)
}

// buildPostActionKeyboard 构建转账/收款完成后的 "钱包 | 提现" URL 按钮键盘
// 交易完成后显示 "💰 钱包 ↗" 和 "👉 提现 ↗"
func (h *InlineQueryHandler) buildPostActionKeyboard(ctx context.Context, b *bot.Bot) *models.InlineKeyboardMarkup {
	walletURL := h.botDeepLink(ctx, b, "wallet")
	withdrawURL := h.botDeepLink(ctx, b, "withdraw")
	if walletURL == "" || withdrawURL == "" {
		return nil
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				tgpkg.URLButton("💰 钱包 ↗", walletURL),
				tgpkg.URLButton("👉 提现 ↗", withdrawURL),
			},
		},
	}
}

// currencyShort 获取币种短显示名
// USDT → "U" (参考格式), TRX → "TRX", CNY → "CNY"
func currencyShort(currency string) string {
	if currency == entity.CurrencyUSDT {
		return "U"
	}
	return currency
}

// =====================================================
// Inline Query 入口
// =====================================================

// Handle 处理 Inline Query 请求
// 支持格式:
//   - 空查询: 列出用户待发送红包
//   - redpacket <id>: 发送指定红包到群聊
//   - <amount>: 转账 (如 @bot 10)
//   - -<amount>: 收款 (如 @bot -10)
func (h *InlineQueryHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.InlineQuery == nil {
		return
	}

	query := update.InlineQuery
	telegramID := query.From.ID
	queryText := strings.TrimSpace(query.Query)

	// 去除零宽空格
	queryText = strings.ReplaceAll(queryText, "\u200b", "")
	queryText = strings.TrimSpace(queryText)

	h.logger.Debug("收到 Inline Query",
		zap.Int64("user_id", telegramID),
		zap.String("query", queryText),
	)

	// 权限验证: 用户必须已注册
	user, err := h.redPacketUC.GetUserByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.answerWithIntroCard(ctx, b, query.ID)
		return
	}

	// 权限验证: 账户必须活跃
	if !user.IsActive() {
		h.answerWithError(ctx, b, query.ID, "您的账户已被冻结，无法使用此功能")
		return
	}

	// 1. 红包发送: redpacket <id>
	if strings.HasPrefix(queryText, "redpacket ") {
		packetIDStr := strings.TrimSpace(strings.TrimPrefix(queryText, "redpacket "))
		h.handleRedPacketSend(ctx, b, query, user, packetIDStr)
		return
	}

	// 2. 收款: -<amount> (负数前缀)
	if strings.HasPrefix(queryText, "-") {
		amountStr := strings.TrimSpace(strings.TrimPrefix(queryText, "-"))
		if amountStr != "" {
			if amount, parseErr := decimal.NewFromString(amountStr); parseErr == nil && amount.GreaterThan(decimal.Zero) {
				h.handleReceiveRequest(ctx, b, query, user, amount)
				return
			}
		}
	}

	// 3. 转账: <amount> (纯正数)
	if queryText != "" {
		if amount, parseErr := decimal.NewFromString(queryText); parseErr == nil && amount.GreaterThan(decimal.Zero) {
			h.handleTransferRequest(ctx, b, query, user, amount)
			return
		}
	}

	// 4. 空查询: 列出用户待发送红包
	if queryText == "" {
		h.handleEmptyQuery(ctx, b, query, user)
		return
	}

	// 无法识别的查询
	h.answerWithError(ctx, b, query.ID, "输入数字转账，负数收款，或 redpacket <编号> 发红包")
}

// =====================================================
// 红包 Inline Query
// =====================================================

// handleRedPacketSend 处理红包发送 Inline Query (redpacket <id>)
func (h *InlineQueryHandler) handleRedPacketSend(ctx context.Context, b *bot.Bot, query *models.InlineQuery, user *entity.User, packetIDStr string) {
	packetID, err := strconv.ParseUint(packetIDStr, 10, 64)
	if err != nil {
		h.answerWithError(ctx, b, query.ID, "无效的红包编号")
		return
	}

	packet, err := h.redPacketUC.GetRedPacketByID(ctx, packetID)
	if err != nil || packet == nil {
		h.answerWithError(ctx, b, query.ID, "红包不存在")
		return
	}

	// 验证红包属于当前用户
	if packet.UserID != user.ID {
		h.answerWithError(ctx, b, query.ID, "无权发送此红包")
		return
	}

	// 验证红包状态
	if packet.Status != entity.RedPacketStatusPending {
		h.answerWithError(ctx, b, query.ID, "红包已发送或已过期")
		return
	}

	senderName := user.DisplayName()
	typeName := app.RedPacketTypeName(packet.Type)

	// 构建红包消息
	messageText := fmt.Sprintf(
		"🧧 <b>%s 的红包</b>\n\n"+
			"%s %s | %s\n"+
			"0/%d 个已领取",
		senderName, packet.Currency, packet.TotalAmount.String(), typeName,
		packet.TotalCount,
	)

	if packet.Remark != "" {
		messageText += fmt.Sprintf("\n💬 %s", packet.Remark)
	}

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         "🧧 领取红包",
					CallbackData: fmt.Sprintf("RedPacket--claim_%d", packet.ID),
				},
			},
		},
	}

	result := []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:    fmt.Sprintf("rp_%d", packet.ID),
			Title: fmt.Sprintf("🧧 发送红包 - %s %s", packet.Currency, packet.TotalAmount.String()),
			Description: fmt.Sprintf("%s | %d 个 | %s",
				packet.Currency, packet.TotalCount, typeName),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: messageText,
				ParseMode:   models.ParseModeHTML,
			},
			ReplyMarkup: keyboard,
		},
	}

	_, err = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       result,
		CacheTime:     0,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}

// =====================================================
// 转账 Inline Query (@bot <amount>)
// =====================================================

// handleTransferRequest 处理转账 Inline Query
// 格式: "🪙 转账给你 X.XX U" + "🐲 收款" 按钮
func (h *InlineQueryHandler) handleTransferRequest(ctx context.Context, b *bot.Bot, query *models.InlineQuery, user *entity.User, amount decimal.Decimal) {
	tgID := query.From.ID

	currencies := []string{entity.CurrencyUSDT, entity.CurrencyTRX, entity.CurrencyCNY}
	var results []models.InlineQueryResult

	for _, currency := range currencies {
		shortCurrency := currencyShort(currency)

		// 验证金额是否符合最低限额
		if err := h.transferUC.ValidateTransferAmount(currency, amount); err != nil {
			continue
		}

		// 检查余额
		if err := h.transferUC.CheckBalance(ctx, user.ID, currency, amount); err != nil {
			// 余额不足: 仍显示但标注
			results = append(results, &models.InlineQueryResultArticle{
				ID:          fmt.Sprintf("transfer_%s_%s", currency, amount.String()),
				Title:       fmt.Sprintf("💰 转账 %s %s (余额不足)", amount.String(), shortCurrency),
				Description: "余额不足，无法发送此转账",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "❌ 余额不足，无法完成转账。",
					ParseMode:   models.ParseModeHTML,
				},
			})
			continue
		}

		// "🪙 转账给你 X.XX U\n请在24小时内领取"
		messageText := fmt.Sprintf(
			"🪙 转账给你 %s %s\n请在24小时内领取",
			amount.String(), shortCurrency,
		)

		// 回调数据: IT--claim_<senderTgID>_<amount>_<currency>
		callbackData := fmt.Sprintf("IT--claim_%d_%s_%s", tgID, amount.String(), currency)

		// "🐲 收款" 按钮
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         "🐲 收款",
						CallbackData: callbackData,
					},
				},
			},
		}

		results = append(results, &models.InlineQueryResultArticle{
			ID:    fmt.Sprintf("transfer_%s_%s", currency, amount.String()),
			Title: fmt.Sprintf("💰 转账 %s %s", amount.String(), shortCurrency),
			Description: fmt.Sprintf(
				"⚠️ 您正在向对方转账 %s 并立刻生效",
				shortCurrency,
			),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: messageText,
				ParseMode:   models.ParseModeHTML,
			},
			ReplyMarkup: keyboard,
		})
	}

	if len(results) == 0 {
		h.answerWithError(ctx, b, query.ID, "金额无效或所有币种余额不足")
		return
	}

	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}

// =====================================================
// 收款 Inline Query (@bot -<amount>)
// =====================================================

// handleReceiveRequest 处理收款 Inline Query
// 格式: "[name] 向你收款 X.XX U" + "🐲 立即支付" 按钮
func (h *InlineQueryHandler) handleReceiveRequest(ctx context.Context, b *bot.Bot, query *models.InlineQuery, user *entity.User, amount decimal.Decimal) {
	requesterName := user.DisplayName()
	tgID := query.From.ID

	currencies := []string{entity.CurrencyUSDT, entity.CurrencyTRX, entity.CurrencyCNY}
	var results []models.InlineQueryResult

	for _, currency := range currencies {
		shortCurrency := currencyShort(currency)

		// 验证金额
		if err := h.transferUC.ValidateTransferAmount(currency, amount); err != nil {
			continue
		}

		// "[name] 向你收款 X.XX U"
		messageText := fmt.Sprintf(
			"%s 向你收款 %s %s",
			requesterName, amount.String(), shortCurrency,
		)

		// 构建深度链接 URL: 点击后跳转到 Bot 私聊，进入 PIN 验证 + 支付流程
		payParam := EncodePayLink(tgID, amount, currency)
		payURL := h.botDeepLink(ctx, b, payParam)
		if payURL == "" {
			h.logger.Warn("Bot 用户名未获取，跳过收款结果", zap.String("currency", currency))
			continue
		}

		// "🐲 立即支付" URL 按钮 → 跳转 Bot 私聊
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					tgpkg.URLButton("🐲 立即支付", payURL),
				},
			},
		}

		results = append(results, &models.InlineQueryResultArticle{
			ID:          fmt.Sprintf("receive_%s_%s", currency, amount.String()),
			Title:       fmt.Sprintf("💸 收款 %s %s", amount.String(), shortCurrency),
			Description: fmt.Sprintf("向对方发送 %s %s 收款请求", amount.String(), shortCurrency),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: messageText,
				ParseMode:   models.ParseModeHTML,
			},
			ReplyMarkup: keyboard,
		})
	}

	if len(results) == 0 {
		h.answerWithError(ctx, b, query.ID, "金额无效")
		return
	}

	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}

// =====================================================
// 空查询: 列出用户待发送红包
// =====================================================

// handleEmptyQuery 处理空查询: 列出用户待发送红包
func (h *InlineQueryHandler) handleEmptyQuery(ctx context.Context, b *bot.Bot, query *models.InlineQuery, user *entity.User) {
	packets, err := h.redPacketUC.GetUserPendingPackets(ctx, user.ID)
	if err != nil {
		h.logger.Error("查询用户红包失败", zap.Error(err))
		h.answerWithIntroCard(ctx, b, query.ID)
		return
	}

	var results []models.InlineQueryResult

	for _, packet := range packets {
		typeName := app.RedPacketTypeName(packet.Type)
		senderName := user.DisplayName()

		messageText := fmt.Sprintf(
			"🧧 <b>%s 的红包</b>\n\n"+
				"%s %s | %s\n"+
				"0/%d 个已领取",
			senderName, packet.Currency, packet.TotalAmount.String(), typeName,
			packet.TotalCount,
		)

		if packet.Remark != "" {
			messageText += fmt.Sprintf("\n💬 %s", packet.Remark)
		}

		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text:         "🧧 领取红包",
						CallbackData: fmt.Sprintf("RedPacket--claim_%d", packet.ID),
					},
				},
			},
		}

		results = append(results, &models.InlineQueryResultArticle{
			ID:    fmt.Sprintf("rp_%d", packet.ID),
			Title: fmt.Sprintf("🧧 红包 #%d - %s %s", packet.ID, packet.Currency, packet.TotalAmount.String()),
			Description: fmt.Sprintf("%d 个 | %s",
				packet.TotalCount, typeName),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: messageText,
				ParseMode:   models.ParseModeHTML,
			},
			ReplyMarkup: keyboard,
		})
	}

	// 没有待发送红包时显示介绍卡片
	if len(results) == 0 {
		results = append(results, &models.InlineQueryResultArticle{
			ID:          "intro",
			Title:       "WalletBot: Telegram 匿名钱包",
			Description: "暂无待发送的红包。输入金额转账，负数收款",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "WalletBot - Telegram 匿名钱包\n\n" +
					"<b>使用方法:</b>\n" +
					"• <code>@botname 10</code> — 转账 10 USDT\n" +
					"• <code>@botname -10</code> — 收款 10 USDT\n" +
					"• <code>@botname redpacket 123</code> — 发送红包",
				ParseMode: models.ParseModeHTML,
			},
		})
	}

	_, err = b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}

// =====================================================
// Inline 转账回调 (IT--)
// =====================================================

// HandleInlineTransferCallback 处理 IT-- (Inline Transfer) 回调
// 当有人点击 [🐲 收款] 按钮时触发
func (h *InlineQueryHandler) HandleInlineTransferCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery
	_, action := telegram.ParseCallbackData(query.Data)

	// action 格式: claim_<senderTgID>_<amount>_<currency>
	if !strings.HasPrefix(action, "claim_") {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		return
	}

	parts := strings.SplitN(strings.TrimPrefix(action, "claim_"), "_", 3)
	if len(parts) != 3 {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		return
	}

	senderTgID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		return
	}
	amount, err := decimal.NewFromString(parts[1])
	if err != nil {
		tgpkg.AnswerCallback(ctx, b, query.ID)
		return
	}
	currency := parts[2]
	clickerTgID := query.From.ID
	inlineMessageID := query.InlineMessageID

	// 不能领取自己的转账
	if clickerTgID == senderTgID {
		tgpkg.AnswerCallbackWithAlert(ctx, b, query.ID, "不能领取自己的转账")
		return
	}

	// 获取领取者信息 (由 Auth 中间件注入 context)
	clicker := telegram.UserFromContext(ctx)
	if clicker == nil {
		tgpkg.AnswerCallbackWithAlert(ctx, b, query.ID, "请先私聊 Bot 发送 /start 注册")
		return
	}

	// 获取发送者信息
	sender, err := h.transferUC.FindRecipientByTelegramID(ctx, senderTgID)
	if err != nil || sender == nil {
		tgpkg.AnswerCallbackWithAlert(ctx, b, query.ID, "转账发起人不存在")
		return
	}

	// 执行转账 (发送者 → 领取者)
	result, err := h.transferUC.ExecuteTransfer(ctx, sender.ID, clicker.ID, currency, amount)
	if err != nil {
		errMsg := "转账失败，请稍后重试"
		if strings.Contains(err.Error(), "余额不足") || strings.Contains(err.Error(), "insufficient") {
			errMsg = "发起人余额不足"
		}
		h.logger.Error("Inline 转账执行失败",
			zap.Int64("sender_tg_id", senderTgID),
			zap.Uint64("clicker_id", clicker.ID),
			zap.Error(err),
		)
		tgpkg.AnswerCallbackWithAlert(ctx, b, query.ID, errMsg)
		return
	}

	// 响应回调
	tgpkg.AnswerCallbackWithText(ctx, b, query.ID, "✅ 领取成功!")

	// 已领取格式: "🔵 [name] 已领取 [amount] [currency]"
	clickerName := clicker.DisplayName()
	text := fmt.Sprintf("🔵 %s 已领取 %s %s", clickerName, amount.String(), currency)

	// 构建 "钱包 | 提现" URL 按钮
	postKeyboard := h.buildPostActionKeyboard(ctx, b)

	if inlineMessageID != "" {
		if postKeyboard != nil {
			tgpkg.EditInlineMessageWithKeyboard(ctx, b, inlineMessageID, text, postKeyboard)
		} else {
			tgpkg.EditInlineMessage(ctx, b, inlineMessageID, text)
		}
	} else if query.Message.Message != nil {
		chatID := query.Message.Message.Chat.ID
		messageID := query.Message.Message.ID
		if postKeyboard != nil {
			tgpkg.EditMessageWithKeyboard(ctx, b, chatID, messageID, text, postKeyboard)
		} else {
			tgpkg.EditMessage(ctx, b, chatID, messageID, text)
		}
	}

	// 异步通知发送者
	go func() {
		notifyText := fmt.Sprintf(
			"✅ 你的转账已被领取\n\n"+
				"领取人: %s\n"+
				"金额: %s %s\n"+
				"余额: %s %s",
			clickerName,
			amount.String(), currency,
			result.SenderNewBalance.StringFixed(4), currency,
		)
		tgpkg.SendMessage(context.Background(), b, senderTgID, notifyText)
	}()
}

// =====================================================
// 辅助方法
// =====================================================

// answerWithIntroCard 返回 Bot 介绍卡片 (未注册用户)
func (h *InlineQueryHandler) answerWithIntroCard(ctx context.Context, b *bot.Bot, queryID string) {
	results := []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          "intro",
			Title:       "WalletBot: Telegram 匿名钱包",
			Description: "安全便捷的加密货币钱包",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "WalletBot - Telegram 匿名钱包\n\n安全便捷的加密货币钱包，支持 USDT/TRX/CNY",
				ParseMode:   models.ParseModeHTML,
			},
		},
	}

	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: queryID,
		Results:       results,
		CacheTime:     300,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}

// answerWithError 返回错误提示卡片
func (h *InlineQueryHandler) answerWithError(ctx context.Context, b *bot.Bot, queryID string, errMsg string) {
	results := []models.InlineQueryResult{
		&models.InlineQueryResultArticle{
			ID:          "error",
			Title:       "⚠️ " + errMsg,
			Description: errMsg,
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: errMsg,
			},
		},
	}

	_, err := b.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: queryID,
		Results:       results,
		CacheTime:     0,
		IsPersonal:    true,
	})
	if err != nil {
		h.logger.Error("AnswerInlineQuery 失败", zap.Error(err))
	}
}
