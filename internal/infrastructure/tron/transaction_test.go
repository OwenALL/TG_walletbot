package tron

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// --- signTransaction 测试 ---

// TestSignTransaction_正常签名 验证对合法交易的签名流程
func TestSignTransaction_正常签名(t *testing.T) {
	// 生成一个真实私钥用于签名
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err, "生成私钥不应失败")

	// 构造一个简单的模拟交易 JSON (需要包含 txID 字段)
	txID := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	rawTx := []byte(`{"txID":"` + txID + `","raw_data":{"contract":[]}}`)

	signedTx, err := signTransaction(rawTx, privKeyHex)
	require.NoError(t, err, "签名不应返回错误")
	require.NotNil(t, signedTx, "签名结果不应为 nil")

	// 解析签名后的交易
	var tx map[string]interface{}
	err = json.Unmarshal(signedTx, &tx)
	require.NoError(t, err, "解析签名后的交易不应失败")

	// 验证包含 signature 字段
	signatures, ok := tx["signature"]
	require.True(t, ok, "签名后的交易应包含 signature 字段")

	sigArray, ok := signatures.([]interface{})
	require.True(t, ok, "signature 应为数组")
	require.Equal(t, 1, len(sigArray), "应有恰好 1 个签名")

	// 验证签名是有效的 hex
	sigHex, ok := sigArray[0].(string)
	require.True(t, ok, "签名应为字符串")
	_, err = hex.DecodeString(sigHex)
	assert.NoError(t, err, "签名应为有效的 hex 字符串")

	// secp256k1 compact 签名长度为 65 字节 (130 hex chars)
	assert.Equal(t, 130, len(sigHex), "compact 签名 hex 长度应为 130 字符 (65 字节)")

	// 验证 txID 保留
	assert.Equal(t, txID, tx["txID"], "txID 应保持不变")
}

// TestSignTransaction_异常场景 验证各种错误输入的处理
func TestSignTransaction_异常场景(t *testing.T) {
	tests := []struct {
		name          string
		rawTx         []byte
		privateKeyHex string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "无效 JSON",
			rawTx:         []byte("not json"),
			privateKeyHex: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr:       true,
			errContains:   "解析交易失败",
		},
		{
			name:          "缺少 txID",
			rawTx:         []byte(`{"raw_data":{"contract":[]}}`),
			privateKeyHex: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr:       true,
			errContains:   "交易缺少 txID",
		},
		{
			name:          "空 txID",
			rawTx:         []byte(`{"txID":"","raw_data":{}}`),
			privateKeyHex: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr:       true,
			errContains:   "交易缺少 txID",
		},
		{
			name:          "无效 txID hex",
			rawTx:         []byte(`{"txID":"ZZZZ"}`),
			privateKeyHex: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			wantErr:       true,
			errContains:   "解码 txID 失败",
		},
		{
			name:          "无效私钥 hex",
			rawTx:         []byte(`{"txID":"aabb"}`),
			privateKeyHex: "ZZZZ",
			wantErr:       true,
			errContains:   "解码私钥失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := signTransaction(tt.rawTx, tt.privateKeyHex)
			if tt.wantErr {
				require.Error(t, err, "应返回错误")
				assert.Contains(t, err.Error(), tt.errContains, "错误信息应包含: %s", tt.errContains)
			} else {
				assert.NoError(t, err, "不应返回错误")
			}
		})
	}
}

// TestSignTransaction_确定性签名 验证同一私钥和交易 ID 始终生成相同签名 (RFC 6979)
func TestSignTransaction_确定性签名(t *testing.T) {
	privKeyHex := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	txID := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	rawTx := []byte(`{"txID":"` + txID + `"}`)

	// 签名多次
	var signatures []string
	for i := 0; i < 5; i++ {
		signedTx, err := signTransaction(rawTx, privKeyHex)
		require.NoError(t, err, "第 %d 次签名不应失败", i+1)

		var tx map[string]interface{}
		err = json.Unmarshal(signedTx, &tx)
		require.NoError(t, err)

		sigs := tx["signature"].([]interface{})
		signatures = append(signatures, sigs[0].(string))
	}

	// 验证所有签名相同 (确定性)
	for i := 1; i < len(signatures); i++ {
		assert.Equal(t, signatures[0], signatures[i], "RFC 6979 确定性签名: 第 %d 次签名应与第一次相同", i+1)
	}
}

// --- clearBytes 测试 ---

// TestClearBytes_正常清零 验证 clearBytes 将字节数组全部置零
func TestClearBytes_正常清零(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "32 字节私钥",
			input: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
		},
		{
			name:  "单字节",
			input: []byte{0xff},
		},
		{
			name:  "全 0xff",
			input: []byte{0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearBytes(tt.input)

			for i, v := range tt.input {
				assert.Equal(t, byte(0), v, "字节位置 %d 应被清零", i)
			}
		})
	}
}

// TestClearBytes_空切片 验证空切片不 panic
func TestClearBytes_空切片(t *testing.T) {
	assert.NotPanics(t, func() {
		clearBytes([]byte{})
	}, "空切片不应 panic")

	assert.NotPanics(t, func() {
		clearBytes(nil)
	}, "nil 切片不应 panic")
}

// --- encodeTRC20Transfer 测试 ---

// TestEncodeTRC20Transfer_编码正确性 验证 ABI 编码参数格式
func TestEncodeTRC20Transfer_编码正确性(t *testing.T) {
	// 先生成一个真实地址
	kg := NewKeyGenerator()
	address, _, err := kg.GenerateAddress()
	require.NoError(t, err)

	amount := big.NewInt(1000000) // 1 USDT

	result := encodeTRC20Transfer(address, amount)

	// 总长度 = 64 (地址参数) + 64 (金额参数) = 128 字符
	assert.Equal(t, 128, len(result), "ABI 编码参数总长度应为 128 字符 (64+64)")

	// 金额部分 (后 64 字符)
	amountPart := result[64:]
	amountBig, ok := new(big.Int).SetString(amountPart, 16)
	require.True(t, ok, "金额部分应为有效 hex")
	assert.Equal(t, int64(1000000), amountBig.Int64(), "金额解码后应为 1000000")
}

// TestEncodeTRC20Transfer_大金额 验证大金额的 uint256 编码
func TestEncodeTRC20Transfer_大金额(t *testing.T) {
	kg := NewKeyGenerator()
	address, _, err := kg.GenerateAddress()
	require.NoError(t, err)

	// 10 亿 USDT = 10^9 * 10^6 = 10^15
	amount := new(big.Int).SetUint64(1000000000000000)
	result := encodeTRC20Transfer(address, amount)

	assert.Equal(t, 128, len(result), "ABI 编码参数总长度应为 128 字符")

	// 验证金额部分
	amountPart := result[64:]
	amountBig, ok := new(big.Int).SetString(amountPart, 16)
	require.True(t, ok)
	assert.Equal(t, amount.String(), amountBig.String(), "大金额应正确编码")
}

// --- base58ToHex 测试 ---

// TestBase58ToHex_往返一致性 验证 Base58 -> hex -> Base58 往返转换
func TestBase58ToHex_往返一致性(t *testing.T) {
	kg := NewKeyGenerator()

	for i := 0; i < 5; i++ {
		t.Run("地址 #"+string(rune('1'+i)), func(t *testing.T) {
			address, _, err := kg.GenerateAddress()
			require.NoError(t, err)

			// Base58 -> hex
			hexAddr := base58ToHex(address)
			require.NotEmpty(t, hexAddr, "hex 转换结果不应为空")

			// hex 应为 40 字符 (20 字节)
			assert.Equal(t, 40, len(hexAddr), "地址 hex 长度应为 40 字符")

			// hex -> Base58
			base58Addr, err := hexAddrToBase58(hexAddr)
			require.NoError(t, err)
			assert.Equal(t, address, base58Addr, "往返转换应保持一致")
		})
	}
}

// TestBase58ToHex_无效输入 验证无效输入返回空字符串
func TestBase58ToHex_无效输入(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isEmpty bool
	}{
		{
			name:    "空字符串",
			input:   "",
			isEmpty: true,
		},
		{
			name:    "太短的字符串",
			input:   "T",
			isEmpty: true,
		},
		{
			name:    "包含无效字符 (Base58 中无 0, O, I, l)",
			input:   "T0OIl",
			isEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base58ToHex(tt.input)
			if tt.isEmpty {
				assert.Empty(t, result, "无效输入应返回空字符串")
			}
		})
	}
}

// --- base58DecodeTx 测试 ---

// TestBase58DecodeTx_正常解码 验证正常 Base58 字符串解码
func TestBase58DecodeTx_正常解码(t *testing.T) {
	// 使用已知的 TRON 地址
	kg := NewKeyGenerator()
	address, _, err := kg.GenerateAddress()
	require.NoError(t, err)

	decoded := base58DecodeTx(address)
	require.NotNil(t, decoded, "正常地址解码不应返回 nil")
	assert.Equal(t, 25, len(decoded), "TRON 地址解码后应为 25 字节")
	assert.Equal(t, byte(0x41), decoded[0], "首字节应为 0x41 (TRON 主网)")
}

// TestBase58DecodeTx_无效字符 验证包含无效字符时返回 nil
func TestBase58DecodeTx_无效字符(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "包含 0",
			input: "T0INVALID",
		},
		{
			name:  "包含 O",
			input: "TOINVALID",
		},
		{
			name:  "包含 I",
			input: "TIINVALID",
		},
		{
			name:  "包含 l (小写 L)",
			input: "TlINVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded := base58DecodeTx(tt.input)
			assert.Nil(t, decoded, "包含无效 Base58 字符时应返回 nil")
		})
	}
}

// TestBase58DecodeTx_前导1 验证前导 '1' 字符正确处理为 0x00 字节
func TestBase58DecodeTx_前导1(t *testing.T) {
	// Base58 中 '1' 代表 0x00 字节
	decoded := base58DecodeTx("111")
	require.NotNil(t, decoded)
	assert.Equal(t, 3, len(decoded), "3 个 '1' 应解码为 3 个 0x00 字节")
	for i, b := range decoded {
		assert.Equal(t, byte(0), b, "位置 %d 应为 0x00", i)
	}
}

// --- Client HTTP 测试 (使用 httptest) ---

// TestClient_SendTRX_成功 使用 mock HTTP server 验证 SendTRX 完整流程
func TestClient_SendTRX_成功(t *testing.T) {
	// 创建一个测试私钥
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	requestCount := 0

	// mock TronGrid API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/wallet/createtransaction":
			// 返回包含 txID 的模拟交易
			json.NewEncoder(w).Encode(map[string]interface{}{
				"txID":     "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
				"raw_data": map[string]interface{}{},
			})
		case "/wallet/broadcasttransaction":
			// 返回广播成功
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": true,
				"txid":   "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
			})
		default:
			http.Error(w, "unknown path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	result, err := client.SendTRX(ctx, "TFromAddress1234", "TToAddress5678", 1000000, privKeyHex)
	require.NoError(t, err, "SendTRX 不应返回错误")
	require.NotNil(t, result, "结果不应为 nil")

	assert.Equal(t, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", result.TxID)
	assert.True(t, result.Success, "交易应成功")
	assert.Equal(t, 2, requestCount, "应发送 2 个请求 (createtransaction + broadcasttransaction)")
}

// TestClient_SendTRX_构造交易失败 验证 HTTP 错误处理
func TestClient_SendTRX_构造交易失败(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.SendTRX(ctx, "TFrom", "TTo", 1000000, "aabb")
	require.Error(t, err, "服务器返回错误时 SendTRX 应返回错误")
	assert.Contains(t, err.Error(), "构造 TRX 交易失败", "错误信息应包含构造失败描述")
}

// TestClient_SendTRX_无效响应 验证无效 JSON 响应处理
func TestClient_SendTRX_无效响应(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 返回没有 txID 的响应
		json.NewEncoder(w).Encode(map[string]interface{}{
			"Error": "something went wrong",
		})
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.SendTRX(ctx, "TFrom", "TTo", 1000000, "aabb")
	require.Error(t, err, "缺少 txID 时应返回错误")
	assert.Contains(t, err.Error(), "构造交易失败", "错误信息应包含构造交易失败描述")
}

// TestClient_SendTRX_广播失败 验证广播交易失败的处理
func TestClient_SendTRX_广播失败(t *testing.T) {
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/wallet/createtransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"txID":     "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd",
				"raw_data": map[string]interface{}{},
			})
		case "/wallet/broadcasttransaction":
			// 广播失败
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result":  false,
				"code":    "BANDWITH_ERROR",
				"message": "bandwidth not enough",
			})
		}
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err = client.SendTRX(ctx, "TFrom", "TTo", 1000000, privKeyHex)
	require.Error(t, err, "广播失败时应返回错误")
	assert.Contains(t, err.Error(), "广播 TRX 交易失败", "错误信息应说明广播失败")
}

// TestClient_postRaw_APIKey设置 验证 API Key 被正确设置到请求头
func TestClient_postRaw_APIKey设置(t *testing.T) {
	var receivedAPIKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("TRON-PRO-API-KEY")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		apiKey:     "my-test-api-key",
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.postRaw(ctx, "/test", []byte(`{}`))
	require.NoError(t, err)

	assert.Equal(t, "my-test-api-key", receivedAPIKey, "请求头应包含 API Key")
}

// TestClient_postRaw_无APIKey 验证未设置 API Key 时不添加请求头
func TestClient_postRaw_无APIKey(t *testing.T) {
	var hasAPIKeyHeader bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hasAPIKeyHeader = r.Header.Get("TRON-PRO-API-KEY") != ""
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		apiKey:     "",
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.postRaw(ctx, "/test", []byte(`{}`))
	require.NoError(t, err)

	assert.False(t, hasAPIKeyHeader, "未设置 API Key 时不应添加请求头")
}

// TestNewClient_默认配置 验证未指定 API URL 时使用默认值
func TestNewClient_默认配置(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient("", "test-key", 0, logger)

	assert.Equal(t, "https://api.trongrid.io", client.apiURL, "未配置时应使用默认 API URL")
	assert.Equal(t, "test-key", client.apiKey, "API Key 应正确设置")
	assert.NotNil(t, client.httpClient, "HTTP client 不应为 nil")
}

// TestNewClient_自定义配置 验证自定义 API URL 生效
func TestNewClient_自定义配置(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient("https://custom-api.example.com", "", 0, logger)

	assert.Equal(t, "https://custom-api.example.com", client.apiURL, "应使用自定义 API URL")
}

// TestNewClient_FeeLimit默认值 验证未指定 fee limit 时使用默认值
func TestNewClient_FeeLimit默认值(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient("", "", 0, logger)
	assert.Equal(t, int64(defaultFeeLimit), client.feeLimit, "未配置时应使用默认 fee limit")
}

// TestNewClient_FeeLimit自定义 验证自定义 fee limit 生效
func TestNewClient_FeeLimit自定义(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewClient("", "", 50000000, logger)
	assert.Equal(t, int64(50000000), client.feeLimit, "应使用自定义 fee limit")
}

// --- SendTRC20 测试 ---

// TestClient_SendTRC20_成功 使用 mock HTTP server 验证 SendTRC20 完整流程
func TestClient_SendTRC20_成功(t *testing.T) {
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/wallet/triggersmartcontract":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{
					"result": true,
				},
				"transaction": map[string]interface{}{
					"txID":     "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd",
					"raw_data": map[string]interface{}{},
				},
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": true,
				"txid":   "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd",
			})
		default:
			http.Error(w, "unknown path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	ctx := context.Background()
	amount := big.NewInt(1000000)
	result, err := client.SendTRC20(ctx, "TFromAddr", "TToAddr", "TContractAddr", amount, privKeyHex)
	require.NoError(t, err, "SendTRC20 不应返回错误")
	require.NotNil(t, result, "结果不应为 nil")

	assert.Equal(t, "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd", result.TxID)
	assert.True(t, result.Success)
	assert.Equal(t, 2, requestCount, "应发送 2 个请求")
}

// TestClient_SendTRC20_构造失败 验证 triggersmartcontract 失败时返回错误
func TestClient_SendTRC20_构造失败(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	ctx := context.Background()
	_, err := client.SendTRC20(ctx, "TFrom", "TTo", "TContract", big.NewInt(1000000), "aabb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "构造 TRC20 交易失败")
}

// TestClient_SendTRC20_合约调用失败 验证合约调用返回错误码时的处理
func TestClient_SendTRC20_合约调用失败(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]interface{}{
				"result":  false,
				"code":    "CONTRACT_VALIDATE_ERROR",
				"message": "contract validate error",
			},
			"transaction": map[string]interface{}{},
		})
	}))
	defer server.Close()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	ctx := context.Background()
	_, err = client.SendTRC20(ctx, "TFrom", "TTo", "TContract", big.NewInt(1000000), privKeyHex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TRC20 交易构造失败")
}

// TestClient_SendTRC20_缺少txID 验证构造成功但缺少 txID 时返回错误
func TestClient_SendTRC20_缺少txID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]interface{}{
				"result": true,
			},
			"transaction": map[string]interface{}{
				// 缺少 txID
				"raw_data": map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	ctx := context.Background()
	_, err = client.SendTRC20(ctx, "TFrom", "TTo", "TContract", big.NewInt(1000000), privKeyHex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "缺少 txID")
}

// TestClient_SendTRC20_广播失败 验证广播失败时返回错误
func TestClient_SendTRC20_广播失败(t *testing.T) {
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/wallet/triggersmartcontract":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{"result": true},
				"transaction": map[string]interface{}{
					"txID":     "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344",
					"raw_data": map[string]interface{}{},
				},
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result":  false,
				"code":    "BANDWITH_ERROR",
				"message": "bandwidth not enough",
			})
		}
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	ctx := context.Background()
	_, err = client.SendTRC20(ctx, "TFrom", "TTo", "TContract", big.NewInt(1000000), privKeyHex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "广播 TRC20 交易失败")
}

// TestClient_postJSON_序列化 验证 postJSON 正确序列化请求体
func TestClient_postJSON_序列化(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.postJSON(ctx, "/test", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})
	require.NoError(t, err)

	assert.Equal(t, "value1", receivedBody["key1"])
	assert.Equal(t, float64(42), receivedBody["key2"])
}

// TestClient_broadcastTransaction_解析响应失败 验证无效响应的处理
func TestClient_broadcastTransaction_解析响应失败(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	logger, _ := zap.NewDevelopment()
	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
	}

	ctx := context.Background()
	_, err := client.broadcastTransaction(ctx, []byte(`{"signature":["abc"]}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析广播响应失败")
}
