package tron

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateQRCode_Success 测试正常生成二维码
func TestGenerateQRCode_Success(t *testing.T) {
	tests := []struct {
		name    string
		content string
		size    int
	}{
		{
			name:    "TRON地址二维码",
			content: "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			size:    256,
		},
		{
			name:    "URL二维码",
			content: "https://tronscan.org/#/address/TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW",
			size:    512,
		},
		{
			name:    "短文本",
			content: "hello",
			size:    128,
		},
		{
			name:    "最小尺寸",
			content: "test",
			size:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := GenerateQRCode(tt.content, tt.size)
			require.NoError(t, err, "生成二维码不应出错")
			assert.NotEmpty(t, data, "二维码数据不应为空")

			// 验证 PNG 文件头 (前 8 字节为 PNG 签名)
			pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
			assert.True(t, len(data) >= 8, "二维码数据应至少 8 字节")
			assert.Equal(t, pngSignature, data[:8], "二维码数据应为合法 PNG 格式")
		})
	}
}

// TestGenerateQRCode_EmptyContent 测试空内容应返回错误
func TestGenerateQRCode_EmptyContent(t *testing.T) {
	data, err := GenerateQRCode("", 256)
	// go-qrcode 库不支持空内容编码，应返回错误
	assert.Error(t, err, "空内容应返回错误")
	assert.Nil(t, data, "错误时不应返回数据")
}

// TestGenerateQRCode_DifferentSizesProduceDifferentOutput 测试不同尺寸产生不同大小的输出
func TestGenerateQRCode_DifferentSizesProduceDifferentOutput(t *testing.T) {
	small, err := GenerateQRCode("test", 64)
	require.NoError(t, err)

	large, err := GenerateQRCode("test", 512)
	require.NoError(t, err)

	// 更大尺寸的二维码数据量应该更大
	assert.Greater(t, len(large), len(small), "512px 二维码应比 64px 二维码数据量更大")
}
