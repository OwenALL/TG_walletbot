package repository

import (
	"context"
	"errors"
	"time"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"gorm.io/gorm"
)

// RedPacketRepo 红包 Repository GORM 实现
type RedPacketRepo struct {
	db *gorm.DB
}

// NewRedPacketRepo 创建红包 Repository
func NewRedPacketRepo(db *gorm.DB) *RedPacketRepo {
	return &RedPacketRepo{db: db}
}

// Create 创建红包
func (r *RedPacketRepo) Create(ctx context.Context, packet *entity.RedPacket) error {
	return r.db.WithContext(ctx).Create(packet).Error
}

// FindByID 根据 ID 查询
func (r *RedPacketRepo) FindByID(ctx context.Context, id uint64) (*entity.RedPacket, error) {
	var p entity.RedPacket
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// Update 更新红包
func (r *RedPacketRepo) Update(ctx context.Context, packet *entity.RedPacket) error {
	return r.db.WithContext(ctx).Save(packet).Error
}

// ListByUserID 获取用户创建的红包列表
func (r *RedPacketRepo) ListByUserID(ctx context.Context, userID uint64, status *int8, offset, limit int) ([]*entity.RedPacket, int64, error) {
	var list []*entity.RedPacket
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.RedPacket{}).Where("user_id = ?", userID)
	if status != nil {
		db = db.Where("status = ?", *status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByUserIDAndStatuses 获取用户指定多个状态的红包列表
func (r *RedPacketRepo) ListByUserIDAndStatuses(ctx context.Context, userID uint64, statuses []int8, offset, limit int) ([]*entity.RedPacket, int64, error) {
	var list []*entity.RedPacket
	var total int64
	db := r.db.WithContext(ctx).Model(&entity.RedPacket{}).
		Where("user_id = ? AND status IN (?)", userID, statuses)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListExpired 获取已过期未处理的红包
func (r *RedPacketRepo) ListExpired(ctx context.Context) ([]*entity.RedPacket, error) {
	var list []*entity.RedPacket
	err := r.db.WithContext(ctx).
		Where("status IN (?) AND expire_at < ?", []int8{
			entity.RedPacketStatusPending, entity.RedPacketStatusSent,
		}, time.Now()).
		Find(&list).Error
	return list, err
}

// RedPacketClaimRepo 红包领取记录 Repository
type RedPacketClaimRepo struct {
	db *gorm.DB
}

// NewRedPacketClaimRepo 创建红包领取记录 Repository
func NewRedPacketClaimRepo(db *gorm.DB) *RedPacketClaimRepo {
	return &RedPacketClaimRepo{db: db}
}

// Create 创建领取记录
func (r *RedPacketClaimRepo) Create(ctx context.Context, claim *entity.RedPacketClaim) error {
	return r.db.WithContext(ctx).Create(claim).Error
}

// FindByPacketAndUser 查询某用户是否已领取
func (r *RedPacketClaimRepo) FindByPacketAndUser(ctx context.Context, packetID, userID uint64) (*entity.RedPacketClaim, error) {
	var c entity.RedPacketClaim
	err := r.db.WithContext(ctx).Where("red_packet_id = ? AND user_id = ?", packetID, userID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

// ListByPacketID 获取红包的所有领取记录
func (r *RedPacketClaimRepo) ListByPacketID(ctx context.Context, packetID uint64) ([]*entity.RedPacketClaim, error) {
	var list []*entity.RedPacketClaim
	err := r.db.WithContext(ctx).Where("red_packet_id = ?", packetID).Order("id ASC").Find(&list).Error
	return list, err
}

// RedPacketCoverRepo 红包封面 Repository
type RedPacketCoverRepo struct {
	db *gorm.DB
}

// NewRedPacketCoverRepo 创建红包封面 Repository
func NewRedPacketCoverRepo(db *gorm.DB) *RedPacketCoverRepo {
	return &RedPacketCoverRepo{db: db}
}

// Create 创建封面
func (r *RedPacketCoverRepo) Create(ctx context.Context, cover *entity.RedPacketCover) error {
	return r.db.WithContext(ctx).Create(cover).Error
}

// ListByUserID 获取用户的封面列表
func (r *RedPacketCoverRepo) ListByUserID(ctx context.Context, userID uint64) ([]*entity.RedPacketCover, error) {
	var list []*entity.RedPacketCover
	err := r.db.WithContext(ctx).Where("user_id = ? AND status = 1", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// Delete 删除封面
func (r *RedPacketCoverRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&entity.RedPacketCover{}, id).Error
}
