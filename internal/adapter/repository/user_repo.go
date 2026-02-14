// Package repository 提供 port 接口的 GORM 实现
package repository

import (
	"context"
	"errors"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// UserRepo 用户 Repository GORM 实现
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户 Repository
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create 创建用户
func (r *UserRepo) Create(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID 根据 ID 查询用户
func (r *UserRepo) FindByID(ctx context.Context, id uint64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// FindByTelegramID 根据 Telegram ID 查询用户
func (r *UserRepo) FindByTelegramID(ctx context.Context, telegramID int64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// FindByUsername 根据用户名查询用户
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// Update 更新用户信息
func (r *UserRepo) Update(ctx context.Context, user *entity.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// List 获取用户列表 (分页)
func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.User{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// CountAll 获取总用户数
func (r *UserRepo) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.User{}).Count(&count).Error
	return count, err
}
