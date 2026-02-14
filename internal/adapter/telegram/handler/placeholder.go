// Package handler 提供 Telegram Bot 的各模块处理器
package handler

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// PlaceholderHandler 占位处理器
// 用于尚未开发的模块 (RedPacket、Service、Finance、Setting 等)
// 点击按钮后提示"功能即将上线"，而非静默失败
type PlaceholderHandler struct {
	logger *zap.Logger
}

// NewPlaceholderHandler 创建占位处理器
func NewPlaceholderHandler(logger *zap.Logger) *PlaceholderHandler {
	return &PlaceholderHandler{logger: logger}
}

// Handle 处理未开发模块的回调请求
func (h *PlaceholderHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	query := update.CallbackQuery

	h.logger.Info("用户点击未上线功能",
		zap.Int64("user_id", query.From.ID),
		zap.String("callback_data", query.Data),
	)

	// 弹窗提示 (ShowAlert=true，用户必须点确定关闭)
	tgpkg.AnswerCallbackWithAlert(ctx, b, query.ID, "🚧 该功能即将上线，敬请期待！")
}
