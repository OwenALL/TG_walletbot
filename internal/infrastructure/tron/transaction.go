package tron

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// SendTRXResult TRX 转账结果
type SendTRXResult struct {
	TxID    string `json:"txid"`
	Success bool   `json:"result"`
}

// SendTRC20Result TRC20 转账结果
type SendTRC20Result struct {
	TxID    string `json:"txid"`
	Success bool   `json:"result"`
}

// SendTRX 发送 TRX 原生代币
// fromAddr: 发送方地址 (Base58)
// toAddr: 接收方地址 (Base58)
// amountSun: 金额 (sun, 1 TRX = 1,000,000 sun)
// privateKeyHex: 发送方私钥 (hex 编码)
func (c *Client) SendTRX(ctx context.Context, fromAddr, toAddr string, amountSun int64, privateKeyHex string) (*SendTRXResult, error) {
	// 1. 构造交易: 调用 /wallet/createtransaction
	createReq := map[string]interface{}{
		"owner_address": fromAddr,
		"to_address":    toAddr,
		"amount":        amountSun,
		"visible":       true,
	}

	rawTx, err := c.postJSON(ctx, "/wallet/createtransaction", createReq)
	if err != nil {
		return nil, fmt.Errorf("构造 TRX 交易失败: %w", err)
	}

	// 解析交易获取 txID
	var txData map[string]interface{}
	if err := json.Unmarshal(rawTx, &txData); err != nil {
		return nil, fmt.Errorf("解析交易数据失败: %w", err)
	}

	txID, ok := txData["txID"].(string)
	if !ok || txID == "" {
		return nil, fmt.Errorf("构造交易失败: 无效的响应 %s", string(rawTx))
	}

	// 2. 签名交易
	signedTx, err := signTransaction(rawTx, privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("签名 TRX 交易失败: %w", err)
	}

	// 3. 广播交易
	result, err := c.broadcastTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("广播 TRX 交易失败: %w", err)
	}

	return &SendTRXResult{
		TxID:    txID,
		Success: result,
	}, nil
}

// SendTRC20 发送 TRC20 代币 (如 USDT)
// fromAddr: 发送方地址 (Base58)
// toAddr: 接收方地址 (Base58)
// contractAddr: TRC20 合约地址 (Base58)
// amount: 金额 (最小单位, 如 USDT 精度 6 则 1 USDT = 1000000)
// privateKeyHex: 发送方私钥 (hex 编码)
func (c *Client) SendTRC20(ctx context.Context, fromAddr, toAddr, contractAddr string, amount *big.Int, privateKeyHex string) (*SendTRC20Result, error) {
	// 构造 TRC20 transfer(address,uint256) 方法调用参数
	// 方法签名: a9059cbb
	parameter := encodeTRC20Transfer(toAddr, amount)

	// 1. 构造智能合约调用交易
	triggerReq := map[string]interface{}{
		"owner_address":     fromAddr,
		"contract_address":  contractAddr,
		"function_selector": "transfer(address,uint256)",
		"parameter":         parameter,
		"fee_limit":         c.feeLimit, // 手续费上限 (sun)，通过配置 tron.fee_limit 调整
		"call_value":        0,
		"visible":           true,
	}

	respBody, err := c.postJSON(ctx, "/wallet/triggersmartcontract", triggerReq)
	if err != nil {
		return nil, fmt.Errorf("构造 TRC20 交易失败: %w", err)
	}

	// 解析响应获取交易体
	var triggerResp triggerSmartContractResponse
	if err := json.Unmarshal(respBody, &triggerResp); err != nil {
		return nil, fmt.Errorf("解析 TRC20 交易响应失败: %w", err)
	}

	if triggerResp.Result.Code != "" && triggerResp.Result.Code != "SUCCESS" {
		return nil, fmt.Errorf("TRC20 交易构造失败: %s - %s", triggerResp.Result.Code, triggerResp.Result.Message)
	}

	// 获取交易体
	txBytes, err := json.Marshal(triggerResp.Transaction)
	if err != nil {
		return nil, fmt.Errorf("序列化交易体失败: %w", err)
	}

	txID, _ := triggerResp.Transaction["txID"].(string)
	if txID == "" {
		return nil, fmt.Errorf("构造 TRC20 交易失败: 缺少 txID")
	}

	// 2. 签名交易
	signedTx, err := signTransaction(txBytes, privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("签名 TRC20 交易失败: %w", err)
	}

	// 3. 广播交易
	result, err := c.broadcastTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("广播 TRC20 交易失败: %w", err)
	}

	return &SendTRC20Result{
		TxID:    txID,
		Success: result,
	}, nil
}

// signTransaction 对交易进行签名
// rawTx: 原始交易 JSON
// privateKeyHex: 私钥 hex
// 返回: 包含签名的交易 JSON
func signTransaction(rawTx []byte, privateKeyHex string) ([]byte, error) {
	// 解析交易
	var tx map[string]interface{}
	if err := json.Unmarshal(rawTx, &tx); err != nil {
		return nil, fmt.Errorf("解析交易失败: %w", err)
	}

	// 获取 txID (即交易哈希)
	txID, ok := tx["txID"].(string)
	if !ok || txID == "" {
		return nil, fmt.Errorf("交易缺少 txID")
	}

	// 将 txID (hex) 转为字节
	txIDBytes, err := hex.DecodeString(txID)
	if err != nil {
		return nil, fmt.Errorf("解码 txID 失败: %w", err)
	}

	// 解析私钥
	privKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("解码私钥失败: %w", err)
	}
	defer clearBytes(privKeyBytes) // 签名完成后清零私钥内存

	privKey := secp256k1.PrivKeyFromBytes(privKeyBytes)

	// TRON txID 已经是 SHA256(raw_data) 的结果，直接签名（不要二次哈希）
	// secp256k1 签名 (使用确定性签名 RFC 6979)
	sig := ecdsa.SignCompact(privKey, txIDBytes, false)

	// 将签名转为 hex
	sigHex := hex.EncodeToString(sig)

	// 添加签名到交易
	tx["signature"] = []string{sigHex}

	return json.Marshal(tx)
}

// broadcastTransaction 广播已签名的交易
func (c *Client) broadcastTransaction(ctx context.Context, signedTx []byte) (bool, error) {
	respBody, err := c.postRaw(ctx, "/wallet/broadcasttransaction", signedTx)
	if err != nil {
		return false, err
	}

	var result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
		TxID    string `json:"txid"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return false, fmt.Errorf("解析广播响应失败: %w", err)
	}

	if !result.Result {
		return false, fmt.Errorf("广播失败: %s - %s", result.Code, result.Message)
	}

	return true, nil
}

// postJSON 发送 POST JSON 请求
func (c *Client) postJSON(ctx context.Context, path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.postRaw(ctx, path, jsonBody)
}

// postRaw 发送 POST 请求 (原始 body)
func (c *Client) postRaw(ctx context.Context, path string, body []byte) ([]byte, error) {
	url := c.apiURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TronGrid API 返回状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// encodeTRC20Transfer 编码 TRC20 transfer(address,uint256) 方法的参数
// 返回: ABI 编码的参数 (不含函数选择器)
func encodeTRC20Transfer(toAddr string, amount *big.Int) string {
	// 将 Base58 地址转为 hex (去掉 0x41 前缀 和 4 字节校验码)
	addrHex := base58ToHex(toAddr)

	// 参数1: address — 左补零到 32 字节 (64 hex chars)
	addrParam := fmt.Sprintf("%064s", addrHex)

	// 参数2: uint256 — 左补零到 32 字节 (64 hex chars)
	amountHex := fmt.Sprintf("%064x", amount)

	return addrParam + amountHex
}

// base58ToHex 将 TRON Base58Check 地址转为 20 字节 hex (不含 0x41 前缀)
func base58ToHex(address string) string {
	decoded := base58DecodeTx(address)
	if decoded == nil || len(decoded) < 25 {
		return ""
	}
	// decoded = 0x41 + 20 bytes address + 4 bytes checksum
	// 取 20 字节地址部分 (跳过 0x41 前缀)
	addrBytes := decoded[1 : len(decoded)-4]
	return hex.EncodeToString(addrBytes)
}

// base58DecodeTx 将 Base58 编码字符串解码为字节数组
func base58DecodeTx(input string) []byte {
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
	output := make([]byte, leadingZeros+len(decoded))
	copy(output[leadingZeros:], decoded)

	return output
}

// clearBytes 清零字节数组 (安全: 用完私钥后清除内存)
func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// triggerSmartContractResponse TriggerSmartContract API 响应
type triggerSmartContractResponse struct {
	Result struct {
		Result  bool   `json:"result"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"result"`
	Transaction map[string]interface{} `json:"transaction"`
}
