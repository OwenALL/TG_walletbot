package tron

import (
	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRCode 为给定内容生成 QR Code PNG 字节数据
// size: 图片像素尺寸 (如 256)
func GenerateQRCode(content string, size int) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, size)
}
