package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/config"
	"go.uber.org/zap"
)

// NewRedis 创建 Redis 客户端连接
func NewRedis(cfg *config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接测试失败: %w", err)
	}

	logger.Info("Redis 连接成功",
		zap.String("addr", cfg.Addr),
		zap.Int("db", cfg.DB),
	)

	return client, nil
}
