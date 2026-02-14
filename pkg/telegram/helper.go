package telegram

import (
	"bytes"
	"io"
)

// bytes2reader 将字节切片转换为 io.Reader
func bytes2reader(data []byte) io.Reader {
	return bytes.NewReader(data)
}
