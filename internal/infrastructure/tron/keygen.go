// Package tron 提供 TRON 区块链基础设施
package tron

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

// KeyGenerator 基于 secp256k1 的 TRON 地址生成器实现
// 实现 port.AddressGenerator 接口
type KeyGenerator struct{}

// NewKeyGenerator 创建地址生成器
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{}
}

// GenerateAddress 生成新的 TRON 地址
// 流程: 生成私钥 -> 推导公钥 -> Keccak256 哈希 -> 取后 20 字节 -> 加 0x41 前缀 -> Base58Check 编码
func (g *KeyGenerator) GenerateAddress() (string, string, error) {
	// 生成 secp256k1 私钥
	privKey, err := secp256k1.GeneratePrivateKeyFromRand(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("生成私钥失败: %w", err)
	}

	// 获取私钥 hex
	privKeyBytes := privKey.Serialize()
	privKeyHex := hex.EncodeToString(privKeyBytes)

	// 从私钥推导公钥 (非压缩格式: 65 字节, 0x04 前缀)
	pubKey := privKey.PubKey()
	pubKeyUncompressed := pubKey.SerializeUncompressed()

	// Keccak256 哈希公钥 (去掉 0x04 前缀的 64 字节)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(pubKeyUncompressed[1:]) // 跳过 0x04 前缀
	pubKeyHash := hasher.Sum(nil)

	// 取哈希后 20 字节作为地址
	addressBytes := pubKeyHash[12:] // 32 - 20 = 12, 取后 20 字节

	// 添加 0x41 前缀 (TRON 主网)
	payload := make([]byte, 21)
	payload[0] = 0x41
	copy(payload[1:], addressBytes)

	// Base58Check 编码
	address := base58CheckEncode(payload)

	return address, privKeyHex, nil
}

// base58CheckEncode 将字节数组进行 Base58Check 编码
// 算法: payload + 双 SHA256(payload) 前 4 字节 -> Base58
func base58CheckEncode(payload []byte) string {
	// 双 SHA256 计算校验码
	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])
	checksum := hash2[:4]

	// 拼接 payload + 校验码
	fullPayload := make([]byte, len(payload)+4)
	copy(fullPayload, payload)
	copy(fullPayload[len(payload):], checksum)

	// Base58 编码
	return base58Encode(fullPayload)
}

// base58Alphabet Base58 字符集
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// base58Encode 将字节数组编码为 Base58 字符串
func base58Encode(input []byte) string {
	// 转换为大整数
	num := new(big.Int).SetBytes(input)
	zero := big.NewInt(0)
	base := big.NewInt(58)
	mod := new(big.Int)

	var result []byte
	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// 处理前导零字节 (在 Base58 中表示为 '1')
	for _, b := range input {
		if b != 0 {
			break
		}
		result = append(result, base58Alphabet[0])
	}

	// 反转结果
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
