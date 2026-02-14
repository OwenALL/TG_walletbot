package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// ==================== NewContainer 测试 ====================

func TestNewContainer(t *testing.T) {
	logger := zap.NewNop()

	c := NewContainer(nil, nil, logger)

	assert.NotNil(t, c)
	assert.Nil(t, c.DB)
	assert.Nil(t, c.Redis)
	assert.Equal(t, logger, c.Logger)
}
