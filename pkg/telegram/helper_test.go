package telegram

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBytes2Reader_NormalData 测试正常字节数据转换
func TestBytes2Reader_NormalData(t *testing.T) {
	input := []byte("hello world")
	reader := bytes2reader(input)

	require.NotNil(t, reader)

	// 读取全部内容验证
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, input, data)
}

// TestBytes2Reader_EmptyData 测试空字节数据
func TestBytes2Reader_EmptyData(t *testing.T) {
	reader := bytes2reader([]byte{})

	require.NotNil(t, reader)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Empty(t, data)
}

// TestBytes2Reader_BinaryData 测试二进制数据
func TestBytes2Reader_BinaryData(t *testing.T) {
	input := []byte{0x00, 0xFF, 0x89, 0x50, 0x4E, 0x47}
	reader := bytes2reader(input)

	require.NotNil(t, reader)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, input, data)
}

// TestBytes2Reader_LargeData 测试大量数据
func TestBytes2Reader_LargeData(t *testing.T) {
	// 1MB 数据
	size := 1024 * 1024
	input := make([]byte, size)
	for i := range input {
		input[i] = byte(i % 256)
	}

	reader := bytes2reader(input)
	require.NotNil(t, reader)

	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Len(t, data, size)
	assert.Equal(t, input, data)
}

// TestBytes2Reader_ImplementsIOReader 测试返回值实现 io.Reader 接口
func TestBytes2Reader_ImplementsIOReader(t *testing.T) {
	reader := bytes2reader([]byte("test"))

	// 确保返回值满足 io.Reader 接口
	var _ io.Reader = reader

	buf := make([]byte, 4)
	n, err := reader.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, []byte("test"), buf)
}

// TestBytes2Reader_MultipleReads 测试多次读取
func TestBytes2Reader_MultipleReads(t *testing.T) {
	reader := bytes2reader([]byte("abcdef"))

	// 第一次读 3 字节
	buf1 := make([]byte, 3)
	n, err := reader.Read(buf1)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("abc"), buf1)

	// 第二次读剩余字节
	buf2 := make([]byte, 3)
	n, err = reader.Read(buf2)
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("def"), buf2)

	// 第三次读应返回 EOF
	buf3 := make([]byte, 1)
	n, err = reader.Read(buf3)
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)
}
