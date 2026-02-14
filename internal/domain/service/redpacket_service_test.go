package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/service/mocks"
)

// newTestRedPacketService 创建测试用的 RedPacketService 及其 mock 依赖
// 注意: db 传 nil，只测试不依赖事务的方法
func newTestRedPacketService() (
	*RedPacketService,
	*mocks.MockRedPacketRepository,
	*mocks.MockRedPacketClaimRepository,
	*mocks.MockRedPacketCoverRepository,
	*mocks.MockWalletRepository,
	*mocks.MockTransactionRepository,
	*mocks.MockCacheRepository,
	*mocks.MockSystemConfigRepository,
) {
	redPacketRepo := new(mocks.MockRedPacketRepository)
	claimRepo := new(mocks.MockRedPacketClaimRepository)
	coverRepo := new(mocks.MockRedPacketCoverRepository)
	walletRepo := new(mocks.MockWalletRepository)
	transactionRepo := new(mocks.MockTransactionRepository)
	cacheRepo := new(mocks.MockCacheRepository)
	systemConfigRepo := new(mocks.MockSystemConfigRepository)
	logger := zap.NewNop()

	svc := NewRedPacketService(
		nil, // db 为 nil，事务方法不测
		redPacketRepo,
		claimRepo,
		coverRepo,
		walletRepo,
		transactionRepo,
		cacheRepo,
		systemConfigRepo,
		logger,
	)

	return svc, redPacketRepo, claimRepo, coverRepo, walletRepo, transactionRepo, cacheRepo, systemConfigRepo
}

// ==================== NewRedPacketService 构造测试 ====================

func TestNewRedPacketService(t *testing.T) {
	svc, redPacketRepo, claimRepo, coverRepo, walletRepo, transactionRepo, cacheRepo, systemConfigRepo := newTestRedPacketService()

	assert.NotNil(t, svc)
	assert.Equal(t, redPacketRepo, svc.redPacketRepo)
	assert.Equal(t, claimRepo, svc.claimRepo)
	assert.Equal(t, coverRepo, svc.coverRepo)
	assert.Equal(t, walletRepo, svc.walletRepo)
	assert.Equal(t, transactionRepo, svc.transactionRepo)
	assert.Equal(t, cacheRepo, svc.cacheRepo)
	assert.Equal(t, systemConfigRepo, svc.systemConfigRepo)
	assert.NotNil(t, svc.logger)
}

// ==================== CreateRedPacket 参数校验测试 ====================

func TestRedPacketService_CreateRedPacket_金额小于等于0(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	tests := []struct {
		name   string
		amount decimal.Decimal
	}{
		{"金额为0", decimal.Zero},
		{"金额为负数", decimal.NewFromFloat(-1.5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CreateRedPacket(ctx, 1, "USDT", tt.amount, 5, entity.RedPacketTypeEqual, "")
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "红包金额必须大于 0")
		})
	}
}

func TestRedPacketService_CreateRedPacket_个数校验(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	tests := []struct {
		name     string
		count    int
		wantErr  string
	}{
		{"个数为0", 0, "红包个数必须大于 0"},
		{"个数为负数", -1, "红包个数必须大于 0"},
		{"个数超过100", 101, "红包个数最多 100 个"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.CreateRedPacket(ctx, 1, "USDT", decimal.NewFromInt(100), tt.count, entity.RedPacketTypeEqual, "")
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRedPacketService_CreateRedPacket_均分红包每份低于0点01(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	// 0.05 / 10 = 0.005 < 0.01
	result, err := svc.CreateRedPacket(ctx, 1, "USDT", decimal.NewFromFloat(0.05), 10, entity.RedPacketTypeEqual, "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "每个红包金额不能低于 0.01")
}

func TestRedPacketService_CreateRedPacket_拼手气红包金额不足分配(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	// 0.05 < 0.01 * 10 = 0.1
	result, err := svc.CreateRedPacket(ctx, 1, "USDT", decimal.NewFromFloat(0.05), 10, entity.RedPacketTypeRandom, "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "红包金额不足以分配给所有人")
}

func TestRedPacketService_CreateRedPacket_均分红包参数校验通过边界(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	// 0.10 / 10 = 0.01 => 刚好等于 0.01，应该通过参数校验
	// 但由于 db 为 nil，事务会 panic，使用 recover 捕获以验证参数校验通过
	// 这里验证不返回参数校验错误即可
	func() {
		defer func() {
			// 预期: 由于 db 为 nil 导致 panic，说明参数校验已通过
			r := recover()
			assert.NotNil(t, r, "参数校验通过后由于 db 为 nil 应该 panic")
		}()
		_, _ = svc.CreateRedPacket(ctx, 1, "USDT", decimal.NewFromFloat(0.10), 10, entity.RedPacketTypeEqual, "")
	}()
}

// ==================== getExpireHours 测试 ====================

func TestRedPacketService_getExpireHours_正常配置(t *testing.T) {
	svc, _, _, _, _, _, _, systemConfigRepo := newTestRedPacketService()
	ctx := context.Background()

	systemConfigRepo.On("Get", ctx, entity.ConfigRedpacketExpireHours).Return("48", nil)

	hours := svc.getExpireHours(ctx)

	assert.Equal(t, 48, hours)
	systemConfigRepo.AssertExpectations(t)
}

func TestRedPacketService_getExpireHours_配置不存在使用默认值(t *testing.T) {
	svc, _, _, _, _, _, _, systemConfigRepo := newTestRedPacketService()
	ctx := context.Background()

	systemConfigRepo.On("Get", ctx, entity.ConfigRedpacketExpireHours).Return("", nil)

	hours := svc.getExpireHours(ctx)

	assert.Equal(t, 24, hours)
}

func TestRedPacketService_getExpireHours_查询失败使用默认值(t *testing.T) {
	svc, _, _, _, _, _, _, systemConfigRepo := newTestRedPacketService()
	ctx := context.Background()

	systemConfigRepo.On("Get", ctx, entity.ConfigRedpacketExpireHours).Return("", errors.New("redis error"))

	hours := svc.getExpireHours(ctx)

	assert.Equal(t, 24, hours)
}

func TestRedPacketService_getExpireHours_非法数值使用默认值(t *testing.T) {
	tests := []struct {
		name string
		val  string
	}{
		{"非数字", "abc"},
		{"小于等于0", "0"},
		{"负数", "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, _, _, _, systemConfigRepo := newTestRedPacketService()
			ctx := context.Background()

			systemConfigRepo.On("Get", ctx, entity.ConfigRedpacketExpireHours).Return(tt.val, nil)

			hours := svc.getExpireHours(ctx)
			assert.Equal(t, 24, hours)
		})
	}
}

// ==================== calculateClaimAmount 测试 ====================

func TestRedPacketService_calculateClaimAmount_最后一人拿全部(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    3,
		ClaimedCount:  2,
		ClaimedAmount: decimal.NewFromFloat(6.5),
		Type:          entity.RedPacketTypeRandom,
	}

	amount := svc.calculateClaimAmount(packet)

	// 剩余 10 - 6.5 = 3.5，最后一人全部拿走
	assert.True(t, amount.Equal(decimal.NewFromFloat(3.5)))
}

func TestRedPacketService_calculateClaimAmount_均分类型(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    3,
		ClaimedCount:  0,
		ClaimedAmount: decimal.Zero,
		Type:          entity.RedPacketTypeEqual,
	}

	amount := svc.calculateClaimAmount(packet)

	// 10 / 3 = 3.33333333 (截断到8位)
	expected := decimal.NewFromFloat(10).Div(decimal.NewFromInt(3)).Truncate(8)
	assert.True(t, amount.Equal(expected))
}

func TestRedPacketService_calculateClaimAmount_均分类型_可整除(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    5,
		ClaimedCount:  0,
		ClaimedAmount: decimal.Zero,
		Type:          entity.RedPacketTypeEqual,
	}

	amount := svc.calculateClaimAmount(packet)

	// 10 / 5 = 2.0
	assert.True(t, amount.Equal(decimal.NewFromInt(2)))
}

func TestRedPacketService_calculateClaimAmount_拼手气类型范围合法(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    5,
		ClaimedCount:  0,
		ClaimedAmount: decimal.Zero,
		Type:          entity.RedPacketTypeRandom,
	}

	// 多次执行，验证随机金额在合法范围内
	for i := 0; i < 100; i++ {
		amount := svc.calculateClaimAmount(packet)

		// 最少 0.01
		assert.True(t, amount.GreaterThanOrEqual(decimal.NewFromFloat(0.01)),
			"金额应 >= 0.01, 实际: %s", amount.String())

		// 最多不超过总额 (二倍均值法上限不会超过剩余总额)
		assert.True(t, amount.LessThanOrEqual(packet.TotalAmount),
			"金额应 <= 总额, 实际: %s", amount.String())
	}
}

func TestRedPacketService_calculateClaimAmount_拼手气类型_已领取部分(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    5,
		ClaimedCount:  3,
		ClaimedAmount: decimal.NewFromFloat(7),
		Type:          entity.RedPacketTypeRandom,
	}

	// 剩余 3 USDT，剩余 2 人
	for i := 0; i < 50; i++ {
		amount := svc.calculateClaimAmount(packet)

		remaining := packet.RemainingAmount()
		assert.True(t, amount.GreaterThanOrEqual(decimal.NewFromFloat(0.01)),
			"金额应 >= 0.01, 实际: %s", amount.String())
		assert.True(t, amount.LessThanOrEqual(remaining),
			"金额应 <= 剩余 %s, 实际: %s", remaining.String(), amount.String())
	}
}

func TestRedPacketService_calculateClaimAmount_默认类型(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	// 使用一个非 1 非 2 的类型值，触发 default 分支
	packet := &entity.RedPacket{
		TotalAmount:   decimal.NewFromFloat(10),
		TotalCount:    4,
		ClaimedCount:  0,
		ClaimedAmount: decimal.Zero,
		Type:          99,
	}

	amount := svc.calculateClaimAmount(packet)

	// default: remaining / remainingCount = 10 / 4 = 2.5
	expected := decimal.NewFromFloat(10).Div(decimal.NewFromInt(4)).Truncate(8)
	assert.True(t, amount.Equal(expected))
}

// ==================== randomAmount 测试 ====================

func TestRedPacketService_randomAmount_边界值_最小情况(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	// 剩余 0.02，剩余 2 人 => 每人最少 0.01, 最多 0.01
	// 二倍均值: avg = 0.02 / 2 = 0.01, max = 0.01 * 2 = 0.02
	// maxAllowed = 0.02 - 0.01*(2-1) = 0.01
	// 所以 max = 0.01, 只能返回 0.01
	amount := svc.randomAmount(decimal.NewFromFloat(0.02), 2)
	assert.True(t, amount.Equal(decimal.NewFromFloat(0.01)))
}

func TestRedPacketService_randomAmount_只有1分钱(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	// 剩余 0.01，1 人
	// 注意: remainingCount 对 randomAmount 来说是传入的参数
	// 如果只剩 0.01, 1人 => avg=0.01, max=0.02, maxAllowed=0.01
	amount := svc.randomAmount(decimal.NewFromFloat(0.01), 1)
	assert.True(t, amount.GreaterThanOrEqual(decimal.NewFromFloat(0.01)))
}

func TestRedPacketService_randomAmount_多次调用结果在范围内(t *testing.T) {
	svc, _, _, _, _, _, _, _ := newTestRedPacketService()

	remaining := decimal.NewFromFloat(5.0)
	remainingCount := 3

	minAmount := decimal.NewFromFloat(0.01)

	for i := 0; i < 100; i++ {
		amount := svc.randomAmount(remaining, remainingCount)

		assert.True(t, amount.GreaterThanOrEqual(minAmount),
			"金额应 >= 0.01, 实际: %s", amount.String())

		// 确保不超过 remaining
		assert.True(t, amount.LessThanOrEqual(remaining),
			"金额应 <= %s, 实际: %s", remaining.String(), amount.String())
	}
}

// ==================== GetUserRedPackets 测试 ====================

func TestRedPacketService_GetUserRedPackets_成功(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	expectedList := []*entity.RedPacket{
		{ID: 1, UserID: 1, Status: entity.RedPacketStatusPending},
		{ID: 2, UserID: 1, Status: entity.RedPacketStatusPending},
	}
	status := int8(entity.RedPacketStatusPending)
	redPacketRepo.On("ListByUserID", ctx, uint64(1), &status, 0, 50).Return(expectedList, int64(2), nil)

	result, err := svc.GetUserRedPackets(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedList, result)
	redPacketRepo.AssertExpectations(t)
}

func TestRedPacketService_GetUserRedPackets_空列表(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	status := int8(entity.RedPacketStatusPending)
	redPacketRepo.On("ListByUserID", ctx, uint64(1), &status, 0, 50).Return([]*entity.RedPacket{}, int64(0), nil)

	result, err := svc.GetUserRedPackets(ctx, 1)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestRedPacketService_GetUserRedPackets_查询失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	status := int8(entity.RedPacketStatusPending)
	redPacketRepo.On("ListByUserID", ctx, uint64(1), &status, 0, 50).Return(([]*entity.RedPacket)(nil), int64(0), errors.New("db error"))

	result, err := svc.GetUserRedPackets(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询用户红包失败")
}

// ==================== GetRedPacketByID 测试 ====================

func TestRedPacketService_GetRedPacketByID_成功(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	expected := &entity.RedPacket{ID: 1, UserID: 1, TotalAmount: decimal.NewFromInt(100)}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(expected, nil)

	result, err := svc.GetRedPacketByID(ctx, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestRedPacketService_GetRedPacketByID_不存在(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.GetRedPacketByID(ctx, 999)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRedPacketService_GetRedPacketByID_查询失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetRedPacketByID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== UpdateSentStatus 测试 ====================

func TestRedPacketService_UpdateSentStatus_成功(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	packet := &entity.RedPacket{
		ID:     1,
		UserID: 1,
		Status: entity.RedPacketStatusPending,
	}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)
	redPacketRepo.On("Update", ctx, mock.AnythingOfType("*entity.RedPacket")).Return(nil)

	err := svc.UpdateSentStatus(ctx, 1, -1001234567890, 42)

	assert.NoError(t, err)
	assert.Equal(t, int8(entity.RedPacketStatusSent), packet.Status)
	assert.NotNil(t, packet.ChatID)
	assert.Equal(t, int64(-1001234567890), *packet.ChatID)
	assert.NotNil(t, packet.MessageID)
	assert.Equal(t, int64(42), *packet.MessageID)
	redPacketRepo.AssertExpectations(t)
}

func TestRedPacketService_UpdateSentStatus_红包不存在(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	err := svc.UpdateSentStatus(ctx, 999, -100, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestRedPacketService_UpdateSentStatus_状态不正确(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	tests := []struct {
		name   string
		status int8
	}{
		{"已发送", entity.RedPacketStatusSent},
		{"已领完", entity.RedPacketStatusClaimed},
		{"已过期", entity.RedPacketStatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redPacketRepoLocal := new(mocks.MockRedPacketRepository)
			svc.redPacketRepo = redPacketRepoLocal

			packet := &entity.RedPacket{ID: 1, UserID: 1, Status: tt.status}
			redPacketRepoLocal.On("FindByID", ctx, uint64(1)).Return(packet, nil)

			err := svc.UpdateSentStatus(ctx, 1, -100, 1)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "红包状态不正确")
		})
	}

	// 恢复原始 repo
	svc.redPacketRepo = redPacketRepo
}

func TestRedPacketService_UpdateSentStatus_查询失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	err := svc.UpdateSentStatus(ctx, 1, -100, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询红包失败")
}

func TestRedPacketService_UpdateSentStatus_更新失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	packet := &entity.RedPacket{ID: 1, UserID: 1, Status: entity.RedPacketStatusPending}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)
	redPacketRepo.On("Update", ctx, mock.AnythingOfType("*entity.RedPacket")).Return(errors.New("db error"))

	err := svc.UpdateSentStatus(ctx, 1, -100, 1)

	assert.Error(t, err)
}

// ==================== GetClaimsByPacketID 测试 ====================

func TestRedPacketService_GetClaimsByPacketID_成功(t *testing.T) {
	svc, _, claimRepo, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	expectedClaims := []*entity.RedPacketClaim{
		{ID: 1, RedPacketID: 1, UserID: 10, Amount: decimal.NewFromFloat(3.5)},
		{ID: 2, RedPacketID: 1, UserID: 20, Amount: decimal.NewFromFloat(6.5)},
	}
	claimRepo.On("ListByPacketID", ctx, uint64(1)).Return(expectedClaims, nil)

	result, err := svc.GetClaimsByPacketID(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedClaims, result)
	claimRepo.AssertExpectations(t)
}

func TestRedPacketService_GetClaimsByPacketID_空列表(t *testing.T) {
	svc, _, claimRepo, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	claimRepo.On("ListByPacketID", ctx, uint64(1)).Return([]*entity.RedPacketClaim{}, nil)

	result, err := svc.GetClaimsByPacketID(ctx, 1)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestRedPacketService_GetClaimsByPacketID_查询失败(t *testing.T) {
	svc, _, claimRepo, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	claimRepo.On("ListByPacketID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetClaimsByPacketID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== GetCoversByUserID 测试 ====================

func TestRedPacketService_GetCoversByUserID_成功(t *testing.T) {
	svc, _, _, coverRepo, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	expectedCovers := []*entity.RedPacketCover{
		{ID: 1, UserID: 1, FileID: "file_1", FileType: "image"},
		{ID: 2, UserID: 1, FileID: "file_2", FileType: "gif"},
	}
	coverRepo.On("ListByUserID", ctx, uint64(1)).Return(expectedCovers, nil)

	result, err := svc.GetCoversByUserID(ctx, 1)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expectedCovers, result)
	coverRepo.AssertExpectations(t)
}

func TestRedPacketService_GetCoversByUserID_空列表(t *testing.T) {
	svc, _, _, coverRepo, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	coverRepo.On("ListByUserID", ctx, uint64(1)).Return([]*entity.RedPacketCover{}, nil)

	result, err := svc.GetCoversByUserID(ctx, 1)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestRedPacketService_GetCoversByUserID_查询失败(t *testing.T) {
	svc, _, _, coverRepo, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	coverRepo.On("ListByUserID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.GetCoversByUserID(ctx, 1)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ==================== AddCover 测试 ====================

func TestRedPacketService_AddCover_成功(t *testing.T) {
	svc, _, _, coverRepo, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	coverRepo.On("Create", ctx, mock.AnythingOfType("*entity.RedPacketCover")).Return(nil)

	err := svc.AddCover(ctx, 1, "file_abc123", "image")

	assert.NoError(t, err)

	// 验证传入参数
	coverRepo.AssertCalled(t, "Create", ctx, mock.MatchedBy(func(cover *entity.RedPacketCover) bool {
		return cover.UserID == 1 &&
			cover.FileID == "file_abc123" &&
			cover.FileType == "image" &&
			cover.Status == 1 &&
			!cover.CreatedAt.IsZero()
	}))
}

func TestRedPacketService_AddCover_创建失败(t *testing.T) {
	svc, _, _, coverRepo, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	coverRepo.On("Create", ctx, mock.AnythingOfType("*entity.RedPacketCover")).Return(errors.New("db error"))

	err := svc.AddCover(ctx, 1, "file_abc", "gif")

	assert.Error(t, err)
}

// ==================== ClaimRedPacket 领取前校验测试 ====================

func TestRedPacketService_ClaimRedPacket_Redis加锁失败(t *testing.T) {
	svc, _, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(false, errors.New("redis error"))

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "系统繁忙")
}

func TestRedPacketService_ClaimRedPacket_获取锁失败_并发中(t *testing.T) {
	svc, _, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(false, nil)

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "红包领取中，请稍后重试")
}

func TestRedPacketService_ClaimRedPacket_红包不存在(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:999", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:999").Return(nil)
	redPacketRepo.On("FindByID", ctx, uint64(999)).Return(nil, nil)

	result, err := svc.ClaimRedPacket(ctx, 999, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")
}

func TestRedPacketService_ClaimRedPacket_红包查询失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(nil, errors.New("db error"))

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询红包失败")
}

func TestRedPacketService_ClaimRedPacket_红包状态不可领取(t *testing.T) {
	tests := []struct {
		name   string
		status int8
	}{
		// Pending 现在允许领取 (inline 模式发送后首次领取自动升级为 Sent)
		{"已领完", entity.RedPacketStatusClaimed},
		{"已过期", entity.RedPacketStatusExpired},
		{"已关闭", entity.RedPacketStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, redPacketRepo, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
			ctx := context.Background()

			cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
			cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)

			packet := &entity.RedPacket{
				ID:       1,
				Status:   tt.status,
				ExpireAt: time.Now().Add(24 * time.Hour),
			}
			redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)

			result, err := svc.ClaimRedPacket(ctx, 1, 10)

			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "红包不可领取")
		})
	}
}

func TestRedPacketService_ClaimRedPacket_红包已过期(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)

	packet := &entity.RedPacket{
		ID:         1,
		Status:     entity.RedPacketStatusSent,
		TotalCount: 5,
		ExpireAt:   time.Now().Add(-1 * time.Hour), // 已过期
	}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "红包已过期")
}

func TestRedPacketService_ClaimRedPacket_红包已领完(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)

	packet := &entity.RedPacket{
		ID:           1,
		Status:       entity.RedPacketStatusSent,
		TotalCount:   3,
		ClaimedCount: 3, // 已领完
		ExpireAt:     time.Now().Add(24 * time.Hour),
	}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "红包已领完")
}

func TestRedPacketService_ClaimRedPacket_已领取过(t *testing.T) {
	svc, redPacketRepo, claimRepo, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)

	packet := &entity.RedPacket{
		ID:           1,
		Status:       entity.RedPacketStatusSent,
		TotalCount:   5,
		ClaimedCount: 2,
		ExpireAt:     time.Now().Add(24 * time.Hour),
	}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)

	existingClaim := &entity.RedPacketClaim{ID: 1, RedPacketID: 1, UserID: 10}
	claimRepo.On("FindByPacketAndUser", ctx, uint64(1), uint64(10)).Return(existingClaim, nil)

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "你已经领取过了")
}

func TestRedPacketService_ClaimRedPacket_查询领取记录失败(t *testing.T) {
	svc, redPacketRepo, claimRepo, _, _, _, cacheRepo, _ := newTestRedPacketService()
	ctx := context.Background()

	cacheRepo.On("SetNX", ctx, "redpacket:lock:1", "1", 5*time.Second).Return(true, nil)
	cacheRepo.On("Delete", ctx, "redpacket:lock:1").Return(nil)

	packet := &entity.RedPacket{
		ID:           1,
		Status:       entity.RedPacketStatusSent,
		TotalCount:   5,
		ClaimedCount: 2,
		ExpireAt:     time.Now().Add(24 * time.Hour),
	}
	redPacketRepo.On("FindByID", ctx, uint64(1)).Return(packet, nil)
	claimRepo.On("FindByPacketAndUser", ctx, uint64(1), uint64(10)).Return(nil, errors.New("db error"))

	result, err := svc.ClaimRedPacket(ctx, 1, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "查询领取记录失败")
}

// ==================== ExpireRedPackets 测试 ====================

func TestRedPacketService_ExpireRedPackets_无过期红包(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("ListExpired", ctx).Return([]*entity.RedPacket{}, nil)

	err := svc.ExpireRedPackets(ctx)

	assert.NoError(t, err)
	redPacketRepo.AssertExpectations(t)
}

func TestRedPacketService_ExpireRedPackets_查询过期红包失败(t *testing.T) {
	svc, redPacketRepo, _, _, _, _, _, _ := newTestRedPacketService()
	ctx := context.Background()

	redPacketRepo.On("ListExpired", ctx).Return(nil, errors.New("db error"))

	err := svc.ExpireRedPackets(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "查询过期红包失败")
}
