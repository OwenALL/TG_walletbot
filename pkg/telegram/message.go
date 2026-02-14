// Package telegram 提供 Telegram Bot 消息操作的统一封装
// 所有 Bot 消息发送/编辑/删除操作必须通过此包调用，禁止直接调用原始 API
package telegram

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/pkg/logger"
)

// SendMessage 发送纯文本消息 (HTML 格式)
func SendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) (*models.Message, error) {
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.L().Error("SendMessage 失败",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return msg, err
}

// SendMessageWithKeyboard 发送带内联键盘的消息 (HTML 格式)
func SendMessageWithKeyboard(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard *models.InlineKeyboardMarkup) (*models.Message, error) {
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		logger.L().Error("SendMessageWithKeyboard 失败",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return msg, err
}

// EditMessage 编辑消息文本 (HTML 格式)
func EditMessage(ctx context.Context, b *bot.Bot, chatID int64, messageID int, text string) (*models.Message, error) {
	msg, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.L().Error("EditMessage 失败",
			zap.Int64("chat_id", chatID),
			zap.Int("message_id", messageID),
			zap.Error(err),
		)
	}
	return msg, err
}

// EditMessageWithKeyboard 编辑消息文本并更新内联键盘 (HTML 格式)
func EditMessageWithKeyboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int, text string, keyboard *models.InlineKeyboardMarkup) (*models.Message, error) {
	msg, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		logger.L().Error("EditMessageWithKeyboard 失败",
			zap.Int64("chat_id", chatID),
			zap.Int("message_id", messageID),
			zap.Error(err),
		)
	}
	return msg, err
}

// EditInlineMessage 编辑 inline 模式消息文本 (HTML 格式)
// 用于编辑通过 inline query 发送的消息，使用 InlineMessageID 而非 ChatID+MessageID
func EditInlineMessage(ctx context.Context, b *bot.Bot, inlineMessageID string, text string) {
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		InlineMessageID: inlineMessageID,
		Text:            text,
		ParseMode:       models.ParseModeHTML,
	})
	if err != nil {
		logger.L().Error("EditInlineMessage 失败",
			zap.String("inline_message_id", inlineMessageID),
			zap.Error(err),
		)
	}
}

// EditInlineMessageWithKeyboard 编辑 inline 模式消息文本并更新键盘 (HTML 格式)
func EditInlineMessageWithKeyboard(ctx context.Context, b *bot.Bot, inlineMessageID string, text string, keyboard *models.InlineKeyboardMarkup) {
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		InlineMessageID: inlineMessageID,
		Text:            text,
		ParseMode:       models.ParseModeHTML,
		ReplyMarkup:     keyboard,
	})
	if err != nil {
		logger.L().Error("EditInlineMessageWithKeyboard 失败",
			zap.String("inline_message_id", inlineMessageID),
			zap.Error(err),
		)
	}
}

// DeleteMessage 删除消息
func DeleteMessage(ctx context.Context, b *bot.Bot, chatID int64, messageID int) (bool, error) {
	ok, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	if err != nil {
		logger.L().Warn("DeleteMessage 失败",
			zap.Int64("chat_id", chatID),
			zap.Int("message_id", messageID),
			zap.Error(err),
		)
	}
	return ok, err
}

// AnswerCallback 响应内联按钮回调 (无提示)
func AnswerCallback(ctx context.Context, b *bot.Bot, callbackID string) (bool, error) {
	return b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
	})
}

// AnswerCallbackWithText 响应内联按钮回调 (Toast 提示)
func AnswerCallbackWithText(ctx context.Context, b *bot.Bot, callbackID string, text string) (bool, error) {
	return b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}

// AnswerCallbackWithAlert 响应内联按钮回调 (弹窗提示)
func AnswerCallbackWithAlert(ctx context.Context, b *bot.Bot, callbackID string, text string) (bool, error) {
	return b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       true,
	})
}

// SendPhoto 发送图片消息
func SendPhoto(ctx context.Context, b *bot.Bot, chatID int64, photo *models.InputFileString, caption string, keyboard *models.InlineKeyboardMarkup) (*models.Message, error) {
	params := &bot.SendPhotoParams{
		ChatID:    chatID,
		Photo:     photo,
		Caption:   caption,
		ParseMode: models.ParseModeHTML,
	}
	if keyboard != nil {
		params.ReplyMarkup = keyboard
	}
	msg, err := b.SendPhoto(ctx, params)
	if err != nil {
		logger.L().Error("SendPhoto 失败",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return msg, err
}

// SendPhotoBytes 发送图片消息 (字节流)
func SendPhotoBytes(ctx context.Context, b *bot.Bot, chatID int64, photoData []byte, filename string, caption string, keyboard *models.InlineKeyboardMarkup) (*models.Message, error) {
	params := &bot.SendPhotoParams{
		ChatID:    chatID,
		Photo:     &models.InputFileUpload{Filename: filename, Data: bytes2reader(photoData)},
		Caption:   caption,
		ParseMode: models.ParseModeHTML,
	}
	if keyboard != nil {
		params.ReplyMarkup = keyboard
	}
	msg, err := b.SendPhoto(ctx, params)
	if err != nil {
		logger.L().Error("SendPhotoBytes 失败",
			zap.Int64("chat_id", chatID),
			zap.Error(err),
		)
	}
	return msg, err
}

// SendError 发送友好错误提示
func SendError(ctx context.Context, b *bot.Bot, chatID int64, errCode string) {
	msg, ok := userFriendlyErrors[errCode]
	if !ok {
		msg = userFriendlyErrors["default"]
	}
	_, _ = SendMessage(ctx, b, chatID, msg)
}

// EditError 编辑消息为友好错误提示
func EditError(ctx context.Context, b *bot.Bot, chatID int64, messageID int, errCode string) {
	msg, ok := userFriendlyErrors[errCode]
	if !ok {
		msg = userFriendlyErrors["default"]
	}
	_, _ = EditMessage(ctx, b, chatID, messageID, msg)
}

// 用户友好错误提示映射
var userFriendlyErrors = map[string]string{
	"insufficient_balance": "余额不足，请先充值",
	"invalid_address":      "地址格式不正确，请检查后重试",
	"network_error":        "网络繁忙，请稍后重试",
	"rate_limit":           "操作太频繁，请稍后再试",
	"pin_invalid":          "支付密码错误，请重试",
	"pin_locked":           "支付密码连续错误次数过多，账户已锁定",
	"user_frozen":          "您的账户已被冻结，如有疑问请联系客服",
	"maintenance":          "系统维护中，请稍后再试",
	"default":              "操作失败，请稍后重试",
}
