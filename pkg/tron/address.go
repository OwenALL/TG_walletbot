// Package tron 提供 TRON 区块链地址相关工具
package tron

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
)

// Base58 字符集
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// ValidateAddress 验证 TRON 地址格式
// 规则: T 开头, 34 字符, Base58Check 编码
func ValidateAddress(address string) bool {
	if len(address) != 34 {
		return false
	}
	if !strings.HasPrefix(address, "T") {
		return false
	}

	// Base58 解码
	decoded := base58Decode(address)
	if decoded == nil || len(decoded) < 4 {
		return false
	}

	// 分离 payload 和校验和
	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	// 计算双 SHA256 校验和
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])

	// 比较校验和
	for i := 0; i < 4; i++ {
		if checksum[i] != hash2[i] {
			return false
		}
	}

	// 验证地址前缀 (TRON 主网地址字节前缀为 0x41)
	if payload[0] != 0x41 {
		return false
	}

	return true
}

// FormatAddress 格式化显示地址 (截断中间部分)
func FormatAddress(address string) string {
	if len(address) <= 12 {
		return address
	}
	return fmt.Sprintf("%s...%s", address[:6], address[len(address)-6:])
}

// base58Decode 将 Base58 编码字符串解码为字节数组
func base58Decode(input string) []byte {
	result := big.NewInt(0)
	for _, c := range input {
		idx := strings.IndexRune(base58Alphabet, c)
		if idx < 0 {
			return nil
		}
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(idx)))
	}

	// 计算前导零个数
	leadingZeros := 0
	for _, c := range input {
		if c != '1' {
			break
		}
		leadingZeros++
	}

	decoded := result.Bytes()
	// 添加前导零字节
	output := make([]byte, leadingZeros+len(decoded))
	copy(output[leadingZeros:], decoded)

	return output
}
