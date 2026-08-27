package tron

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"unsafe"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
)

// mockConfigRepo 测试用的 SystemConfigRepository mock
type mockConfigRepo struct {
	data map[string]string
}

func newMockConfigRepo(data map[string]string) *mockConfigRepo {
	if data == nil {
		data = make(map[string]string)
	}
	return &mockConfigRepo{data: data}
}

func (m *mockConfigRepo) Get(_ context.Context, key string) (string, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return "", nil
}

func (m *mockConfigRepo) Set(_ context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func (m *mockConfigRepo) GetAll(_ context.Context) ([]*entity.SystemConfig, error) {
	var configs []*entity.SystemConfig
	for k, v := range m.data {
		configs = append(configs, &entity.SystemConfig{ConfigKey: k, ConfigValue: v})
	}
	return configs, nil
}

func (m *mockConfigRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			result[k] = v
		}
	}
	return result, nil
}

// makeWritableString 创建拥有可写底层内存的字符串
// Go 字符串字面量存储在只读数据段，直接对其底层内存写操作会导致 panic。
// 通过 []byte -> string 转换，保证底层内存是堆分配的可写内存。
func makeWritableString(s string) string {
	b := []byte(s)
	return unsafe.String(&b[0], len(b))
}

// --- clearString 测试 ---

// TestClearString_正常清零 验证 clearString 将字符串底层内存全部置零
func TestClearString_正常清零(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "普通私钥 (64 字符 hex)",
			input: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:  "短字符串",
			input: "hello",
		},
		{
			name:  "单字符",
			input: "x",
		},
		{
			name:  "含特殊字符",
			input: "abc123!@#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用可写内存的字符串 (避免对只读数据段写操作导致 panic)
			s := makeWritableString(tt.input)
			originalLen := len(s)
			require.Greater(t, originalLen, 0, "测试输入不应为空")

			// 获取底层内存指针，在清零前记录
			ptr := unsafe.StringData(s)

			// 执行清零
			clearString(&s)

			// 验证字符串被设为空
			assert.Equal(t, "", s, "clearString 后字符串应为空")
			assert.Equal(t, 0, len(s), "clearString 后字符串长度应为 0")

			// 验证底层内存确实被清零 (通过 unsafe 读取原始内存位置)
			b := unsafe.Slice(ptr, originalLen)
			for i, v := range b {
				assert.Equal(t, byte(0), v, "字节位置 %d 应被清零，实际值: 0x%02x", i, v)
			}
		})
	}
}

// TestClearString_空字符串 验证 clearString 处理空字符串不 panic
func TestClearString_空字符串(t *testing.T) {
	s := ""

	// 不应 panic
	assert.NotPanics(t, func() {
		clearString(&s)
	}, "clearString 处理空字符串不应 panic")

	assert.Equal(t, "", s, "空字符串清零后仍应为空")
}

// TestClearString_多次调用幂等 验证对同一指针多次调用不会 panic
func TestClearString_多次调用幂等(t *testing.T) {
	s := makeWritableString("test-private-key-data")

	clearString(&s)
	assert.Equal(t, "", s)

	// 再次调用不应 panic (此时 s 是空字符串，直接走 early return)
	assert.NotPanics(t, func() {
		clearString(&s)
	}, "对已清零的字符串再次调用不应 panic")
}

// --- NewWithdrawalSender 测试 ---

// TestNewWithdrawalSender_创建成功 验证 WithdrawalSender 正确初始化
func TestNewWithdrawalSender_创建成功(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	crypto := NewCryptoService()
	client := &Client{apiURL: "https://test.example.com", logger: logger}
	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client,
		crypto,
		"THotWalletAddr123",
		"encrypted-key-hex",
		"encryption-key",
		configRepo,
		USDTContractAddress,
		logger,
	)

	require.NotNil(t, sender, "WithdrawalSender 不应为 nil")
	assert.Equal(t, "THotWalletAddr123", sender.hotWalletAddr, "热钱包地址应正确设置")
	assert.Equal(t, "encrypted-key-hex", sender.hotWalletKey, "加密后私钥应正确设置")
	assert.Equal(t, "encryption-key", sender.encryptionKey, "解密密钥应正确设置")
	assert.NotNil(t, sender.configRepo, "configRepo 应正确设置")
	assert.Equal(t, USDTContractAddress, sender.usdtContract, "USDT 合约地址应正确设置")
}

// --- SendWithdrawal 测试 ---

// TestSendWithdrawal_超过阈值转人工 验证金额超过阈值时返回人工处理标记
func TestSendWithdrawal_超过阈值转人工(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	crypto := NewCryptoService()
	client := &Client{apiURL: "https://test.example.com", logger: logger}
	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "100",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", "encrypted-key",
		"enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	txHash, isManual, err := sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(150.0))

	require.NoError(t, err, "超过阈值不应返回错误")
	assert.Equal(t, "", txHash, "超过阈值时 txHash 应为空")
	assert.True(t, isManual, "超过阈值时应标记为人工处理")
}

// TestSendWithdrawal_动态阈值 验证阈值从 configRepo 动态读取
func TestSendWithdrawal_动态阈值(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	crypto := NewCryptoService()
	client := &Client{apiURL: "https://test.example.com", logger: logger}
	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "50",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", "encrypted-key",
		"enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()

	// 金额 60 > 阈值 50，应转人工
	_, isManual, err := sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(60.0))
	require.NoError(t, err)
	assert.True(t, isManual, "60 > 50 应转人工")

	// 动态修改阈值为 200
	configRepo.data[entity.ConfigWithdrawUSDTAutoThreshold] = "200"

	// 金额 60 <= 新阈值 200，应自动处理 (会因解密失败报错，但不应标记为人工)
	_, isManual, err = sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(60.0))
	// 解密会失败，但验证的是阈值判断逻辑
	assert.False(t, isManual, "60 <= 200 不应转人工")
	assert.Error(t, err, "解密失败应报错")
}

// TestSendWithdrawal_不支持的币种 验证非 TRX/USDT 币种返回错误
func TestSendWithdrawal_不支持的币种(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// 先加密一个测试私钥，让解密能成功
	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	client := &Client{apiURL: "https://test.example.com", logger: logger}
	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	_, _, err = sender.SendWithdrawal(ctx, "TToAddr", "BTC", decimal.NewFromFloat(1.0))

	require.Error(t, err, "不支持的币种应返回错误")
	assert.Contains(t, err.Error(), "不支持的提币币种", "错误信息应说明不支持")
}

// TestSendWithdrawal_解密私钥失败 验证解密失败时返回错误
func TestSendWithdrawal_解密私钥失败(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	crypto := NewCryptoService()
	client := &Client{apiURL: "https://test.example.com", logger: logger}
	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", "invalid-encrypted-data",
		"wrong-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	_, _, err := sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(10.0))

	require.Error(t, err, "解密失败应返回错误")
	assert.Contains(t, err.Error(), "解密热钱包私钥失败")
}

// TestSendWithdrawal_TRX自动发送成功 验证 TRX 金额在阈值内的自动发送流程
func TestSendWithdrawal_TRX自动发送成功(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	// mock TronGrid HTTP API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wallet/createtransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"txID":     "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344",
				"raw_data": map[string]interface{}{},
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": true,
				"txid":   "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344",
			})
		}
	}))
	defer server.Close()

	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	txHash, isManual, err := sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(10.0))

	require.NoError(t, err, "TRX 自动发送不应返回错误")
	assert.False(t, isManual, "自动发送不应标记为人工")
	assert.NotEmpty(t, txHash, "应返回交易 hash")
}

// TestSendWithdrawal_USDT自动发送成功 验证 USDT 金额在阈值内的自动发送流程
func TestSendWithdrawal_USDT自动发送成功(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wallet/triggersmartcontract":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": map[string]interface{}{"result": true},
				"transaction": map[string]interface{}{
					"txID":     "11223344556677881122334455667788112233445566778811223344556677aa",
					"raw_data": map[string]interface{}{},
				},
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result": true,
				"txid":   "11223344556677881122334455667788112233445566778811223344556677aa",
			})
		}
	}))
	defer server.Close()

	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	txHash, isManual, err := sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyUSDT, decimal.NewFromFloat(50.0))

	require.NoError(t, err, "USDT 自动发送不应返回错误")
	assert.False(t, isManual, "自动发送不应标记为人工")
	assert.NotEmpty(t, txHash, "应返回交易 hash")
}

// TestSendWithdrawal_TRX发送失败 验证 TRX 发送失败时返回错误
func TestSendWithdrawal_TRX发送失败(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	_, _, err = sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(10.0))

	require.Error(t, err, "发送失败应返回错误")
	assert.Contains(t, err.Error(), "发送 TRX 失败")
}

// TestSendWithdrawal_TRX广播失败 验证 TRX 广播不成功时返回错误
func TestSendWithdrawal_TRX广播失败(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wallet/createtransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"txID":     "aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344",
				"raw_data": map[string]interface{}{},
			})
		case "/wallet/broadcasttransaction":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"result":  false,
				"code":    "ERROR",
				"message": "broadcast failed",
			})
		}
	}))
	defer server.Close()

	client := &Client{
		apiURL:     server.URL,
		httpClient: server.Client(),
		logger:     logger,
		feeLimit:   defaultFeeLimit,
	}

	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	_, _, err = sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyTRX, decimal.NewFromFloat(10.0))

	require.Error(t, err, "广播失败应返回错误")
}

// TestSendWithdrawal_USDT金额无效 验证 USDT 金额为零时返回错误
func TestSendWithdrawal_USDT金额无效(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	kg := NewKeyGenerator()
	_, privKeyHex, err := kg.GenerateAddress()
	require.NoError(t, err)

	crypto := NewCryptoService()
	encryptedKey, err := crypto.EncryptPrivateKey(privKeyHex, "test-enc-key")
	require.NoError(t, err)

	client := &Client{
		apiURL: "https://test.example.com",
		logger: logger,
	}

	configRepo := newMockConfigRepo(map[string]string{
		entity.ConfigWithdrawUSDTAutoThreshold: "1000",
	})

	sender := NewWithdrawalSender(
		client, crypto, "THotWallet", encryptedKey,
		"test-enc-key", configRepo,
		USDTContractAddress, logger,
	)

	ctx := context.Background()
	_, _, err = sender.SendWithdrawal(ctx, "TToAddr", entity.CurrencyUSDT, decimal.NewFromFloat(0.0))

	require.Error(t, err, "零金额 USDT 应返回错误")
	assert.Contains(t, err.Error(), "USDT 金额无效")
}

// --- 编译验证: 确保使用的导入不产生 unused import 错误 ---
var (
	_ = (*big.Int)(nil)
)
