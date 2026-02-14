package tron

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKeyGenerator_GenerateAddress_地址格式正确 验证生成的 TRON 地址以 "T" 开头且长度为 34 字符
func TestKeyGenerator_GenerateAddress_地址格式正确(t *testing.T) {
	kg := NewKeyGenerator()

	address, _, err := kg.GenerateAddress()
	require.NoError(t, err, "生成地址不应返回错误")

	assert.True(t, len(address) == 34, "TRON 地址长度应为 34 字符，实际长度: %d", len(address))
	assert.Equal(t, "T", string(address[0]), "TRON 地址应以 'T' 开头")
}

// TestKeyGenerator_GenerateAddress_私钥格式正确 验证私钥为 64 字符的 hex 字符串
func TestKeyGenerator_GenerateAddress_私钥格式正确(t *testing.T) {
	kg := NewKeyGenerator()

	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err, "生成地址不应返回错误")

	assert.Equal(t, 64, len(privKeyHex), "私钥 hex 长度应为 64 字符，实际长度: %d", len(privKeyHex))

	// 验证是有效的 hex 字符串
	_, err = hex.DecodeString(privKeyHex)
	assert.NoError(t, err, "私钥应为有效的 hex 字符串")
}

// TestKeyGenerator_GenerateAddress_多次生成不重复 验证连续生成的 10 个地址互不重复
func TestKeyGenerator_GenerateAddress_多次生成不重复(t *testing.T) {
	kg := NewKeyGenerator()
	count := 10

	addresses := make(map[string]struct{}, count)
	privKeys := make(map[string]struct{}, count)

	for i := 0; i < count; i++ {
		address, privKeyHex, err := kg.GenerateAddress()
		require.NoError(t, err, "第 %d 次生成地址不应返回错误", i+1)

		_, addrExists := addresses[address]
		assert.False(t, addrExists, "第 %d 次生成的地址与之前重复: %s", i+1, address)
		addresses[address] = struct{}{}

		_, keyExists := privKeys[privKeyHex]
		assert.False(t, keyExists, "第 %d 次生成的私钥与之前重复", i+1)
		privKeys[privKeyHex] = struct{}{}
	}

	assert.Equal(t, count, len(addresses), "应生成 %d 个不重复的地址", count)
	assert.Equal(t, count, len(privKeys), "应生成 %d 个不重复的私钥", count)
}

// TestKeyGenerator_GenerateAddress_Base58Check校验 验证地址可被正确 Base58Check 解码
// 解码后应为 21 字节 payload (0x41 前缀 + 20 字节地址) + 4 字节校验码
func TestKeyGenerator_GenerateAddress_Base58Check校验(t *testing.T) {
	kg := NewKeyGenerator()

	address, _, err := kg.GenerateAddress()
	require.NoError(t, err, "生成地址不应返回错误")

	// Base58 解码
	decoded := base58Decode(address)
	require.NotNil(t, decoded, "Base58 解码不应返回 nil")

	// 完整数据 = 21 字节 payload + 4 字节校验码 = 25 字节
	assert.Equal(t, 25, len(decoded), "Base58Check 解码后应为 25 字节，实际: %d", len(decoded))

	// 分离 payload 和校验码
	payload := decoded[:21]
	checksum := decoded[21:]

	// 验证前缀字节为 0x41 (TRON 主网)
	assert.Equal(t, byte(0x41), payload[0], "payload 第一字节应为 0x41 (TRON 主网标识)")

	// 验证校验码: 双 SHA256(payload) 的前 4 字节
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])
	expectedChecksum := hash2[:4]

	assert.Equal(t, expectedChecksum, checksum, "Base58Check 校验码不匹配")
}

// base58Decode 将 Base58 编码的字符串解码为字节数组
func base58Decode(input string) []byte {
	// 构建字符到值的映射
	alphabetMap := make(map[byte]int64)
	for i := 0; i < len(base58Alphabet); i++ {
		alphabetMap[base58Alphabet[i]] = int64(i)
	}

	result := big.NewInt(0)
	base := big.NewInt(58)

	for i := 0; i < len(input); i++ {
		val, ok := alphabetMap[input[i]]
		if !ok {
			return nil // 无效字符
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(val))
	}

	// 转换为字节
	resultBytes := result.Bytes()

	// 处理前导 '1' (代表 0x00 字节)
	numLeadingZeros := 0
	for i := 0; i < len(input); i++ {
		if input[i] != '1' {
			break
		}
		numLeadingZeros++
	}

	// 拼接前导零和实际数据
	decoded := make([]byte, numLeadingZeros+len(resultBytes))
	copy(decoded[numLeadingZeros:], resultBytes)

	return decoded
}
