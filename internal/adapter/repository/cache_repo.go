package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheRepo 缓存 Repository Redis 实现
type CacheRepo struct {
	rdb *redis.Client
}

// NewCacheRepo 创建缓存 Repository
func NewCacheRepo(rdb *redis.Client) *CacheRepo {
	return &CacheRepo{rdb: rdb}
}

// Get 获取缓存值
func (r *CacheRepo) Get(ctx context.Context, key string) (string, error) {
	val, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Set 设置缓存值
func (r *CacheRepo) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.rdb.Set(ctx, key, fmt.Sprintf("%v", value), expiration).Err()
}

// Delete 删除缓存
func (r *CacheRepo) Delete(ctx context.Context, key string) error {
	return r.rdb.Del(ctx, key).Err()
}

// Exists 检查缓存是否存在
func (r *CacheRepo) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

// Incr 自增计数器
func (r *CacheRepo) Incr(ctx context.Context, key string) (int64, error) {
	return r.rdb.Incr(ctx, key).Result()
}

// SetExpire 设置过期时间
func (r *CacheRepo) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	return r.rdb.Expire(ctx, key, expiration).Err()
}

// SetNX 仅在 key 不存在时设置 (用于分布式锁)
func (r *CacheRepo) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, key, value, expiration).Result()
}

// HSet Hash 类型设置
func (r *CacheRepo) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.rdb.HSet(ctx, key, values...).Err()
}

// HGet Hash 类型获取
func (r *CacheRepo) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := r.rdb.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// HGetAll Hash 类型获取全部
func (r *CacheRepo) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.rdb.HGetAll(ctx, key).Result()
}
