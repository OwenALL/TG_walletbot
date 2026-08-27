package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
	apperrors "github.com/OwenALL/TG_walletbot/pkg/errors"
)

// AdminService 管理员领域服务
type AdminService struct {
	adminUserRepo port.AdminUserRepository
	adminLogRepo  port.AdminLogRepository
	logger        *zap.Logger
}

// NewAdminService 创建管理员领域服务
func NewAdminService(
	adminUserRepo port.AdminUserRepository,
	adminLogRepo port.AdminLogRepository,
	logger *zap.Logger,
) *AdminService {
	return &AdminService{
		adminUserRepo: adminUserRepo,
		adminLogRepo:  adminLogRepo,
		logger:        logger,
	}
}

// Authenticate 验证管理员账号密码
func (s *AdminService) Authenticate(ctx context.Context, username, password string) (*entity.AdminUser, error) {
	admin, err := s.adminUserRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, apperrors.Wrap(err, "查询管理员失败")
	}
	if admin == nil {
		return nil, apperrors.NewUnauthorized("用户名或密码错误")
	}
	if admin.Status != 1 {
		return nil, apperrors.NewForbidden("账号已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, apperrors.NewUnauthorized("用户名或密码错误")
	}

	// 更新最后登录时间
	now := time.Now()
	admin.LastLoginAt = &now
	_ = s.adminUserRepo.Update(ctx, admin)

	s.logger.Info("管理员登录成功",
		zap.String("username", admin.Username),
		zap.Uint64("admin_id", admin.ID),
	)

	return admin, nil
}

// GetAdminByID 根据 ID 获取管理员
func (s *AdminService) GetAdminByID(ctx context.Context, id uint64) (*entity.AdminUser, error) {
	return s.adminUserRepo.FindByID(ctx, id)
}

// CreateAuditLog 创建审计日志
func (s *AdminService) CreateAuditLog(ctx context.Context, adminID uint64, action, targetType string, targetID *uint64, detail, ipAddress string) {
	log := &entity.AdminLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
	}
	if err := s.adminLogRepo.Create(ctx, log); err != nil {
		s.logger.Error("创建审计日志失败", zap.Error(err))
	}
}

// ChangePassword 修改管理员密码
// 验证旧密码正确后，对新密码进行 bcrypt 哈希并更新
func (s *AdminService) ChangePassword(ctx context.Context, adminID uint64, oldPassword, newPassword string) error {
	// 查询管理员
	admin, err := s.adminUserRepo.FindByID(ctx, adminID)
	if err != nil {
		return apperrors.Wrap(err, "查询管理员失败")
	}
	if admin == nil {
		return apperrors.NewNotFound("管理员", adminID)
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(oldPassword)); err != nil {
		return apperrors.NewBadRequest("旧密码不正确")
	}

	// 验证新密码长度
	if len(newPassword) < 6 {
		return apperrors.NewBadRequest("新密码长度不能少于 6 位")
	}

	// 哈希新密码
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(err, "密码加密失败")
	}

	// 保存
	admin.PasswordHash = newHash
	if err := s.adminUserRepo.Update(ctx, admin); err != nil {
		return apperrors.Wrap(err, "更新密码失败")
	}

	s.logger.Info("管理员修改密码成功",
		zap.Uint64("admin_id", adminID),
		zap.String("username", admin.Username),
	)

	return nil
}

// ListAuditLogs 获取审计日志列表
func (s *AdminService) ListAuditLogs(ctx context.Context, filter *port.AdminLogFilter, offset, limit int) ([]*entity.AdminLog, int64, error) {
	return s.adminLogRepo.List(ctx, filter, offset, limit)
}

// HashPassword 密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
