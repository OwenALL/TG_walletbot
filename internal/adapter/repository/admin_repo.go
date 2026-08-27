package repository

import (
	"context"
	"errors"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	"gorm.io/gorm"
)

// AdminUserRepo 管理员 Repository GORM 实现
type AdminUserRepo struct {
	db *gorm.DB
}

// NewAdminUserRepo 创建管理员 Repository
func NewAdminUserRepo(db *gorm.DB) *AdminUserRepo {
	return &AdminUserRepo{db: db}
}

// Create 创建管理员
func (r *AdminUserRepo) Create(ctx context.Context, admin *entity.AdminUser) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

// FindByID 根据 ID 查询
func (r *AdminUserRepo) FindByID(ctx context.Context, id uint64) (*entity.AdminUser, error) {
	var admin entity.AdminUser
	err := r.db.WithContext(ctx).First(&admin, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &admin, err
}

// FindByUsername 根据用户名查询
func (r *AdminUserRepo) FindByUsername(ctx context.Context, username string) (*entity.AdminUser, error) {
	var admin entity.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &admin, err
}

// Update 更新管理员
func (r *AdminUserRepo) Update(ctx context.Context, admin *entity.AdminUser) error {
	return r.db.WithContext(ctx).Save(admin).Error
}

// List 获取管理员列表
func (r *AdminUserRepo) List(ctx context.Context, offset, limit int) ([]*entity.AdminUser, int64, error) {
	var list []*entity.AdminUser
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.AdminUser{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// AdminLogRepo 管理员操作日志 Repository GORM 实现
type AdminLogRepo struct {
	db *gorm.DB
}

// NewAdminLogRepo 创建管理员操作日志 Repository
func NewAdminLogRepo(db *gorm.DB) *AdminLogRepo {
	return &AdminLogRepo{db: db}
}

// Create 创建操作日志
func (r *AdminLogRepo) Create(ctx context.Context, log *entity.AdminLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// List 获取操作日志列表
func (r *AdminLogRepo) List(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
	var list []*entity.AdminLog
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.AdminLog{})

	if filter != nil {
		if filter.AdminID != nil {
			db = db.Where("admin_id = ?", *filter.AdminID)
		}
		if filter.Action != "" {
			db = db.Where("action = ?", filter.Action)
		}
		if filter.TargetType != "" {
			db = db.Where("target_type = ?", filter.TargetType)
		}
		if filter.StartTime != "" {
			db = db.Where("created_at >= ?", filter.StartTime)
		}
		if filter.EndTime != "" {
			db = db.Where("created_at <= ?", filter.EndTime)
		}
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// LEFT JOIN admin_users 填充 admin_username 字段
	if err := db.Select("admin_logs.*, admin_users.username as admin_username").
		Joins("LEFT JOIN admin_users ON admin_users.id = admin_logs.admin_id").
		Offset(offset).Limit(limit).Order("admin_logs.id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
