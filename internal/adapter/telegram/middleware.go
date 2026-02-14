package telegram

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// Middleware Bot 中间件类型
type Middleware func(next HandlerFunc) HandlerFunc

// ApplyMiddleware 应用中间件链 (从右到左)
func ApplyMiddleware(handler HandlerFunc, middlewares ...Middleware) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// RecoveryMiddleware 捕获 panic，防止 Bot 崩溃
func RecoveryMiddleware(logger *zap.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Bot panic 恢复",
						zap.Any("error", r),
						zap.String("stack", string(debug.Stack())),
					)
					// 尝试向用户发送友好错误
					chatID := extractChatID(update)
					if chatID != 0 {
						tgpkg.SendError(ctx, b, chatID, "default")
					}
				}
			}()
			next(ctx, b, update)
		}
	}
}

// LoggingMiddleware 记录所有请求日志
func LoggingMiddleware(logger *zap.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			start := time.Now()

			userID := extractUserID(update)
			action := extractAction(update)

			logger.Info("Bot 请求",
				zap.Int64("user_id", userID),
				zap.String("action", action),
			)

			next(ctx, b, update)

			logger.Info("Bot 响应完成",
				zap.Int64("user_id", userID),
				zap.String("action", action),
				zap.Duration("duration", time.Since(start)),
			)
		}
	}
}

// MaintenanceMiddleware 维护模式开关
func MaintenanceMiddleware(configRepo port.SystemConfigRepository) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			val, err := configRepo.Get(ctx, entity.ConfigMaintenanceMode)
			if err == nil && val == "true" {
				chatID := extractChatID(update)
				if chatID != 0 {
					tgpkg.SendError(ctx, b, chatID, "maintenance")
				}
				return
			}
			next(ctx, b, update)
		}
	}
}

// RateLimitMiddleware 基于用户的频率限制
func RateLimitMiddleware(rdb *redis.Client, maxRequests int, window time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			userID := extractUserID(update)
			if userID == 0 {
				next(ctx, b, update)
				return
			}

			key := fmt.Sprintf("ratelimit:%d", userID)
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				next(ctx, b, update)
				return
			}

			// 第一次请求时设置过期时间
			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			if count > int64(maxRequests) {
				chatID := extractChatID(update)
				if chatID != 0 {
					tgpkg.SendError(ctx, b, chatID, "rate_limit")
				}
				return
			}

			next(ctx, b, update)
		}
	}
}

// AuthMiddleware 用户认证/自动注册
// 将用户信息注入 context
func AuthMiddleware(userRepo port.UserRepository) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			tgUser := extractTGUser(update)
			if tgUser == nil {
				next(ctx, b, update)
				return
			}

			// 查找或创建用户
			user, err := userRepo.FindByTelegramID(ctx, tgUser.ID)
			if err != nil {
				next(ctx, b, update)
				return
			}

			// 用户不存在，标记为新用户
			if user == nil {
				ctx = context.WithValue(ctx, ctxKeyIsNewUser, true)
				ctx = context.WithValue(ctx, ctxKeyTGUser, tgUser)
				next(ctx, b, update)
				return
			}

			// 检查用户状态
			if !user.IsActive() {
				chatID := extractChatID(update)
				if chatID != 0 {
					tgpkg.SendError(ctx, b, chatID, "user_frozen")
				}
				return
			}

			// 更新用户信息 (username/名字可能变化)
			needUpdate := false
			if user.Username != tgUser.Username {
				user.Username = tgUser.Username
				needUpdate = true
			}
			if user.FirstName != tgUser.FirstName {
				user.FirstName = tgUser.FirstName
				needUpdate = true
			}
			if user.LastName != tgUser.LastName {
				user.LastName = tgUser.LastName
				needUpdate = true
			}
			if needUpdate {
				_ = userRepo.Update(ctx, user)
			}

			ctx = context.WithValue(ctx, ctxKeyUser, user)
			ctx = context.WithValue(ctx, ctxKeyIsNewUser, false)
			next(ctx, b, update)
		}
	}
}

// --- Context Key 定义 ---

type contextKey string

const (
	ctxKeyUser      contextKey = "user"
	ctxKeyIsNewUser contextKey = "is_new_user"
	ctxKeyTGUser    contextKey = "tg_user"
)

// UserFromContext 从 context 获取用户实体
func UserFromContext(ctx context.Context) *entity.User {
	u, _ := ctx.Value(ctxKeyUser).(*entity.User)
	return u
}

// IsNewUserFromContext 判断是否为新用户
func IsNewUserFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(ctxKeyIsNewUser).(bool)
	return v
}

// TGUserFromContext 从 context 获取 Telegram 用户信息
func TGUserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(ctxKeyTGUser).(*models.User)
	return u
}

// --- 辅助函数 ---

// extractChatID 从 update 中提取 chat ID
func extractChatID(update *models.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message.Message != nil {
		return update.CallbackQuery.Message.Message.Chat.ID
	}
	return 0
}

// extractUserID 从 update 中提取用户 ID
func extractUserID(update *models.Update) int64 {
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID
	}
	if update.CallbackQuery != nil && update.CallbackQuery.From.ID != 0 {
		return update.CallbackQuery.From.ID
	}
	return 0
}

// extractAction 从 update 中提取操作描述
func extractAction(update *models.Update) string {
	if update.Message != nil {
		if update.Message.Text != "" {
			return "text:" + update.Message.Text
		}
		return "message"
	}
	if update.CallbackQuery != nil {
		return "callback:" + update.CallbackQuery.Data
	}
	return "unknown"
}

// extractTGUser 从 update 中提取 Telegram User
func extractTGUser(update *models.Update) *models.User {
	if update.Message != nil && update.Message.From != nil {
		return update.Message.From
	}
	if update.CallbackQuery != nil {
		return &update.CallbackQuery.From
	}
	return nil
}
