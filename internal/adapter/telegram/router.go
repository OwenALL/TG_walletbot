// Package telegram 提供 Telegram Bot 适配器
// 负责回调路由注册和中间件管理
package telegram

import (
	"context"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"
)

// HandlerFunc Bot 处理函数类型
type HandlerFunc func(ctx context.Context, b *bot.Bot, update *models.Update)

// Router 基于 callback_data 前缀的路由器
type Router struct {
	logger   *zap.Logger
	handlers map[string]HandlerFunc // module 前缀 -> handler
}

// NewRouter 创建路由器
func NewRouter(logger *zap.Logger) *Router {
	return &Router{
		logger:   logger,
		handlers: make(map[string]HandlerFunc),
	}
}

// RegisterModule 注册模块处理器
// module: 模块前缀 (如 "Wallet", "Transfer", "Exchange")
// handler: 该模块的回调处理函数
func (r *Router) RegisterModule(module string, handler HandlerFunc) {
	r.handlers[module] = handler
	r.logger.Info("注册 Bot 模块",
		zap.String("module", module),
	)
}

// HandleCallback 回调路由分发
// callback_data 格式: Module--action 或 Module--action_param
func (r *Router) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	module, _ := ParseCallbackData(data)

	handler, ok := r.handlers[module]
	if !ok {
		r.logger.Warn("未注册的回调模块",
			zap.String("module", module),
			zap.String("callback_data", data),
		)
		// 响应回调避免 Telegram 客户端一直转圈
		if update.CallbackQuery.ID != "" {
			_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "功能暂不可用",
			})
		}
		return
	}

	handler(ctx, b, update)
}

// ParseCallbackData 解析 callback_data 为模块和动作
// 格式: Module--action 或 Module--action_param
// 返回: module, action
func ParseCallbackData(data string) (module, action string) {
	parts := strings.SplitN(data, "--", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return data, ""
}

// RegisterHandlers 在 Bot 实例上注册所有处理器
// middlewares: 中间件链，同时应用于命令处理器和回调处理器
// 注意: 默认处理器 (defaultHandler) 需要通过 bot.WithDefaultHandler 选项在创建 Bot 时注入，
// 不在此方法中注册
func (r *Router) RegisterHandlers(b *bot.Bot, commandHandlers map[string]HandlerFunc, middlewares ...Middleware) {
	// 注册命令处理器
	for cmd, handler := range commandHandlers {
		h := handler // 闭包捕获
		b.RegisterHandler(bot.HandlerTypeMessageText, "/"+cmd, bot.MatchTypeExact, func(ctx context.Context, b *bot.Bot, update *models.Update) {
			h(ctx, b, update)
		})
	}

	// 将回调分发函数包裹中间件链 (Recovery → Logging → Maintenance → RateLimit → Auth)
	wrappedCallback := HandlerFunc(r.HandleCallback)
	if len(middlewares) > 0 {
		wrappedCallback = ApplyMiddleware(r.HandleCallback, middlewares...)
	}

	// 注册回调处理器 (按模块前缀，经过完整中间件链)
	for module := range r.handlers {
		prefix := module + "--"
		b.RegisterHandler(bot.HandlerTypeCallbackQueryData, prefix, bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *models.Update) {
			wrappedCallback(ctx, b, update)
		})
	}
}

// DefaultHandler 返回默认消息处理函数，用于传入 bot.WithDefaultHandler 选项
func (r *Router) DefaultHandler(handler HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		handler(ctx, b, update)
	}
}
