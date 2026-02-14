package port

import (
	"context"
	"time"
)

// CacheRepository 缓存端口
type CacheRepository interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (string, error)
	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// Incr 自增计数器
	Incr(ctx context.Context, key string) (int64, error)
	// SetExpire 设置过期时间
	SetExpire(ctx context.Context, key string, expiration time.Duration) error
	// SetNX 仅在 key 不存在时设置 (用于分布式锁)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	// HSet Hash 类型设置
	HSet(ctx context.Context, key string, values ...interface{}) error
	// HGet Hash 类型获取
	HGet(ctx context.Context, key, field string) (string, error)
	// HGetAll Hash 类型获取全部
	HGetAll(ctx context.Context, key string) (map[string]string, error)
}
