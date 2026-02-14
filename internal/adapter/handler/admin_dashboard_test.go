package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- DashboardHandler 测试 ---
// DashboardHandler.Stats 内部调用 AdminUseCase.GetDashboardStats，
// 该方法大量使用 uc.db 直接查询，在无真实 DB 的单元测试环境中会 panic。
// 因此 Dashboard 的完整测试需在集成测试中完成。
// 此处仅验证 Handler 构造器和基本属性。

func TestDashboardHandler_New(t *testing.T) {
	mockUC := newMockAdminUseCase()
	handler := NewDashboardHandler(mockUC.toAdminUseCase(), testLogger())
	assert.NotNil(t, handler)
}
