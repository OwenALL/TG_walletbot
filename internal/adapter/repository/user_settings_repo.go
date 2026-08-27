package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserSettingsRepo 用户配置 Repository GORM 实现
type UserSettingsRepo struct {
	db *gorm.DB
}

// NewUserSettingsRepo 创建用户配置 Repository
func NewUserSettingsRepo(db *gorm.DB) *UserSettingsRepo {
	return &UserSettingsRepo{db: db}
}

// FindByUserID 根据用户 ID 查询
func (r *UserSettingsRepo) FindByUserID(ctx context.Context, userID uint64) (*entity.UserSettings, error) {
	var settings entity.UserSettings
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &settings, err
}

// Upsert 创建或更新用户配置
func (r *UserSettingsRepo) Upsert(ctx context.Context, settings *entity.UserSettings) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"language", "small_free_usdt", "small_free_trx", "notification_enabled", "updated_at"}),
	}).Create(settings).Error
}

// Update 更新配置
func (r *UserSettingsRepo) Update(ctx context.Context, settings *entity.UserSettings) error {
	return r.db.WithContext(ctx).Save(settings).Error
}
