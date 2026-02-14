package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- AdminTransactionHandler 测试 ---
// 注意: AdminTransactionHandler.List 和 .Deposits 内部调用
// AdminUseCase.ListTransactions / ListDeposits，这些方法使用 uc.db 直接查询，
// 在无真实 DB 的单元测试环境中会 panic。完整测试需在集成测试中覆盖。
// 此处仅验证 Handler 构造。

func TestAdminTransactionHandler_New(t *testing.T) {
	mockUC := newMockAdminUseCase()
	handler := NewAdminTransactionHandler(mockUC.toAdminUseCase(), testLogger())
	assert.NotNil(t, handler)
}
