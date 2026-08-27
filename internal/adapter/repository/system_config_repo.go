package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SystemConfigRepo 系统配置 Repository GORM 实现
// 内置内存缓存，减少数据库查询
type SystemConfigRepo struct {
	db    *gorm.DB
	cache sync.Map
}

// NewSystemConfigRepo 创建系统配置 Repository
func NewSystemConfigRepo(db *gorm.DB) *SystemConfigRepo {
	return &SystemConfigRepo{db: db}
}

// Get 获取配置值
func (r *SystemConfigRepo) Get(ctx context.Context, key string) (string, error) {
	// 先查内存缓存
	if v, ok := r.cache.Load(key); ok {
		return v.(string), nil
	}
	// 查数据库
	var cfg entity.SystemConfig
	err := r.db.WithContext(ctx).Where("config_key = ?", key).First(&cfg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	r.cache.Store(key, cfg.ConfigValue)
	return cfg.ConfigValue, nil
}

// Set 设置配置值
func (r *SystemConfigRepo) Set(ctx context.Context, key, value string) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "config_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_value", "updated_at"}),
	}).Create(&entity.SystemConfig{
		ConfigKey:   key,
		ConfigValue: value,
	}).Error
	if err == nil {
		r.cache.Store(key, value)
	}
	return err
}

// GetAll 获取所有配置
func (r *SystemConfigRepo) GetAll(ctx context.Context) ([]*entity.SystemConfig, error) {
	var configs []*entity.SystemConfig
	err := r.db.WithContext(ctx).Find(&configs).Error
	if err == nil {
		// 刷新缓存
		for _, cfg := range configs {
			r.cache.Store(cfg.ConfigKey, cfg.ConfigValue)
		}
	}
	return configs, err
}

// GetMultiple 批量获取配置
func (r *SystemConfigRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	var configs []*entity.SystemConfig
	err := r.db.WithContext(ctx).Where("config_key IN ?", keys).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(configs))
	for _, cfg := range configs {
		result[cfg.ConfigKey] = cfg.ConfigValue
		r.cache.Store(cfg.ConfigKey, cfg.ConfigValue)
	}
	return result, nil
}
