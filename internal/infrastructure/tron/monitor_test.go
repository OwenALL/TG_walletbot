package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/OwenALL/TG_walletbot/internal/app"
	"github.com/OwenALL/TG_walletbot/internal/domain/entity"
	"github.com/OwenALL/TG_walletbot/internal/domain/port"
)

// --- hexAddrToBase58 测试 ---

// TestHexAddrToBase58_已知地址转换 使用已知的 TRON 地址验证 hex -> Base58 转换正确性
func TestHexAddrToBase58_已知地址转换(t *testing.T) {
	kg := NewKeyGenerator()
	address, _, err := kg.GenerateAddress()
	require.NoError(t, err, "生成地址不应失败")

	// 从 Base58 地址解码出 hex
	decoded := base58Decode(address)
	require.NotNil(t, decoded, "Base58 解码不应返回 nil")
	require.Equal(t, 25, len(decoded), "解码后应为 25 字节")

	// 提取 20 字节地址部分 (跳过 0x41 前缀和 4 字节校验码)
	addrBytes := decoded[1:21]
	hexAddr := hex.EncodeToString(addrBytes)

	// 使用 hexAddrToBase58 转换回来
	result, err := hexAddrToBase58(hexAddr)
	require.NoError(t, err, "hexAddrToBase58 不应返回错误")

	assert.Equal(t, address, result, "转换回的 Base58 地址应与原始地址一致")
}

// TestHexAddrToBase58_表驱动测试 验证多种 hex 输入的处理
func TestHexAddrToBase58_表驱动测试(t *testing.T) {
	tests := []struct {
		name      string
		hexAddr   string
		wantErr   bool
		wantStart string // 期望地址开头字符 (TRON 地址以 T 开头)
	}{
		{
			name:      "标准 40 字符 hex",
			hexAddr:   "41b00f80145ef33d37ede2b5b4a400fbf2300c23",
			wantErr:   false,
			wantStart: "T",
		},
		{
			name:      "带前导零的 hex (补齐到 40 字符)",
			hexAddr:   "00000000000000000000000000000000000000ff",
			wantErr:   false,
			wantStart: "T",
		},
		{
			name:      "短于 40 字符的 hex (自动补零)",
			hexAddr:   "ff",
			wantErr:   false,
			wantStart: "T",
		},
		{
			name:    "无效 hex 字符",
			hexAddr: "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ",
			wantErr: true,
		},
		{
			name:    "奇数长度的 hex (补零后仍有效)",
			hexAddr: "abc",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hexAddrToBase58(tt.hexAddr)

			if tt.wantErr {
				assert.Error(t, err, "应返回错误")
				return
			}

			require.NoError(t, err, "不应返回错误")
			assert.NotEmpty(t, result, "结果不应为空")

			if tt.wantStart != "" {
				assert.True(t, len(result) > 0 && string(result[0]) == tt.wantStart,
					"地址应以 '%s' 开头，实际: '%s'", tt.wantStart, result)
			}
		})
	}
}

// TestHexAddrToBase58_往返一致性 验证多个随机生成的地址进行 hex -> Base58 -> hex 往返转换保持一致
func TestHexAddrToBase58_往返一致性(t *testing.T) {
	kg := NewKeyGenerator()

	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("地址 #%d", i+1), func(t *testing.T) {
			address, _, err := kg.GenerateAddress()
			require.NoError(t, err)

			decoded := base58Decode(address)
			require.NotNil(t, decoded)
			require.Equal(t, 25, len(decoded))
			hexAddr := hex.EncodeToString(decoded[1:21])

			result, err := hexAddrToBase58(hexAddr)
			require.NoError(t, err)

			assert.Equal(t, address, result, "往返转换后地址应保持一致")
		})
	}
}

// --- processBlockData 测试 ---

// mockBlockScanner 用于测试的 mock BlockScanner 实现
type mockBlockScanner struct {
	connectErr       error
	currentBlockNum  int64
	currentBlockErr  error
	blocks           map[int64]*port.BlockData
	blockRangeResult []*port.BlockData
	blockRangeErr    error
	txReceipt        *port.TxReceipt
	txReceiptErr     error
}

func (m *mockBlockScanner) Connect() error { return m.connectErr }
func (m *mockBlockScanner) Close()         {}
func (m *mockBlockScanner) GetCurrentBlockNumber(_ context.Context) (int64, error) {
	return m.currentBlockNum, m.currentBlockErr
}
func (m *mockBlockScanner) GetBlockByNum(_ context.Context, num int64) (*port.BlockData, error) {
	if block, ok := m.blocks[num]; ok {
		return block, nil
	}
	return nil, fmt.Errorf("区块 %d 不存在", num)
}
func (m *mockBlockScanner) GetBlockRange(_ context.Context, _, _ int64) ([]*port.BlockData, error) {
	return m.blockRangeResult, m.blockRangeErr
}
func (m *mockBlockScanner) GetTransactionReceipt(_ context.Context, _ string) (*port.TxReceipt, error) {
	return m.txReceipt, m.txReceiptErr
}

// mockCacheRepo 实现 port.CacheRepository 用于测试
type mockCacheRepo struct {
	data map[string]string
}

func newMockCacheRepo() *mockCacheRepo {
	return &mockCacheRepo{data: make(map[string]string)}
}

func (m *mockCacheRepo) Get(_ context.Context, key string) (string, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("key not found")
}
func (m *mockCacheRepo) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	m.data[key] = fmt.Sprintf("%v", value)
	return nil
}
func (m *mockCacheRepo) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}
func (m *mockCacheRepo) Exists(_ context.Context, key string) (bool, error) {
	_, ok := m.data[key]
	return ok, nil
}
func (m *mockCacheRepo) Incr(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *mockCacheRepo) SetExpire(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (m *mockCacheRepo) SetNX(_ context.Context, _ string, _ interface{}, _ time.Duration) (bool, error) {
	return true, nil
}
func (m *mockCacheRepo) HSet(_ context.Context, _ string, _ ...interface{}) error {
	return nil
}
func (m *mockCacheRepo) HGet(_ context.Context, _, _ string) (string, error) {
	return "", nil
}
func (m *mockCacheRepo) HGetAll(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}

// newTestMonitor 创建用于测试的 DepositMonitor (使用 NewDepositMonitor 确保 addrCache 被初始化)
func newTestMonitor(logger *zap.Logger) *DepositMonitor {
	return NewDepositMonitor(
		&mockBlockScanner{},
		nil, nil, nil, nil, nil,
		USDTContractAddress,
		logger,
	)
}

// TestProcessBlockData_空区块 验证空区块返回 0 充值
func TestProcessBlockData_空区块(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	tests := []struct {
		name  string
		block *port.BlockData
	}{
		{
			name:  "nil 区块",
			block: nil,
		},
		{
			name:  "区块号为 0",
			block: &port.BlockData{Number: 0},
		},
		{
			name:  "无交易的区块",
			block: &port.BlockData{Number: 100, Transactions: nil},
		},
		{
			name:  "空交易列表",
			block: &port.BlockData{Number: 100, Transactions: []port.BlockTxData{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			count := monitor.processBlockData(ctx, tt.block)
			assert.Equal(t, 0, count, "空区块应返回 0 充值")
		})
	}
}

// TestProcessBlockData_失败交易跳过 验证失败的交易不被处理
func TestProcessBlockData_失败交易跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	monitor.AddAddress("TMonitoredAddress123456789012345", 1)

	block := &port.BlockData{
		Number: 100,
		Transactions: []port.BlockTxData{
			{
				TxID:    "failed_tx_001",
				Success: false,
				Contracts: []port.TxContractData{
					{
						Type:      "TransferContract",
						ToAddress: "TMonitoredAddress123456789012345",
						Amount:    1000000,
					},
				},
			},
		},
	}

	ctx := context.Background()
	count := monitor.processBlockData(ctx, block)
	assert.Equal(t, 0, count, "失败的交易不应被计为充值")
}

// TestProcessBlockData_非监控地址跳过 验证转入非监控地址的交易不被处理
func TestProcessBlockData_非监控地址跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	block := &port.BlockData{
		Number: 100,
		Transactions: []port.BlockTxData{
			{
				TxID:    "tx_to_unknown_addr",
				Success: true,
				Contracts: []port.TxContractData{
					{
						Type:      "TransferContract",
						ToAddress: "TUnknownAddress1234567890123456",
						Amount:    5000000,
					},
				},
			},
		},
	}

	ctx := context.Background()
	count := monitor.processBlockData(ctx, block)
	assert.Equal(t, 0, count, "转入非监控地址的交易不应被计为充值")
}

// TestProcessTRC20FromData_非USDT合约跳过 验证非 USDT 合约的 TRC20 交易被跳过
func TestProcessTRC20FromData_非USDT合约跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	ctx := context.Background()
	contract := port.TxContractData{
		Type:            "TriggerSmartContract",
		ContractAddress: "TOtherContractAddress1234567890",
		Data:            "a9059cbb" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" + "0000000000000000000000000000000000000000000000000000000000100000",
	}

	result := monitor.processTRC20FromData(ctx, "tx_001", contract)
	assert.False(t, result, "非 USDT 合约的 TRC20 交易应被跳过")
}

// TestProcessTRC20FromData_非transfer方法跳过 验证非 transfer 方法的合约调用被跳过
func TestProcessTRC20FromData_非transfer方法跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	ctx := context.Background()
	contract := port.TxContractData{
		Type:            "TriggerSmartContract",
		ContractAddress: USDTContractAddress,
		Data:            "095ea7b3" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" + "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	}

	result := monitor.processTRC20FromData(ctx, "tx_002", contract)
	assert.False(t, result, "非 transfer 方法的调用应被跳过")
}

// TestProcessTRC20FromData_数据长度不足跳过 验证 data 长度不足 136 字符时被跳过
func TestProcessTRC20FromData_数据长度不足跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	ctx := context.Background()
	tests := []struct {
		name string
		data string
	}{
		{name: "只有函数选择器", data: "a9059cbb"},
		{name: "缺少金额部分", data: "a9059cbb" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678"},
		{name: "金额部分不完整", data: "a9059cbb" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" + "00000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := port.TxContractData{
				Type:            "TriggerSmartContract",
				ContractAddress: USDTContractAddress,
				Data:            tt.data,
			}
			result := monitor.processTRC20FromData(ctx, "tx_short", contract)
			assert.False(t, result, "数据长度不足时应跳过")
		})
	}
}

// TestProcessTRC20FromData_解析地址和金额 验证 TRC20 transfer data 解析的地址和金额正确性
func TestProcessTRC20FromData_解析地址和金额(t *testing.T) {
	kg := NewKeyGenerator()
	realAddress, _, err := kg.GenerateAddress()
	require.NoError(t, err, "生成地址不应失败")

	decoded := base58Decode(realAddress)
	require.NotNil(t, decoded)
	require.Equal(t, 25, len(decoded))
	addrHex := hex.EncodeToString(decoded[1:21])

	paddedAddr := fmt.Sprintf("%064s", addrHex)
	amount := big.NewInt(1000000)
	paddedAmount := fmt.Sprintf("%064x", amount)
	data := transferSelector + paddedAddr + paddedAmount

	// 验证地址解析
	parsedAddr, parseErr := hexAddrToBase58(data[32:72])
	require.NoError(t, parseErr, "地址解析不应失败")
	assert.Equal(t, realAddress, parsedAddr, "解析出的地址应与原始地址一致")

	// 验证金额解析
	amountHex := data[72:136]
	amountBig, ok := new(big.Int).SetString(amountHex, 16)
	require.True(t, ok, "金额 hex 解析不应失败")
	assert.Equal(t, int64(1000000), amountBig.Int64(), "解析出的金额应为 1000000 (1 USDT)")
}

// TestProcessTRC20FromData_收款地址不在监控列表 验证收款地址非监控地址时跳过
func TestProcessTRC20FromData_收款地址不在监控列表(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	kg := NewKeyGenerator()
	realAddress, _, err := kg.GenerateAddress()
	require.NoError(t, err)
	decoded := base58Decode(realAddress)
	addrHex := hex.EncodeToString(decoded[1:21])

	paddedAddr := fmt.Sprintf("%064s", addrHex)
	paddedAmount := fmt.Sprintf("%064x", big.NewInt(1000000))
	data := transferSelector + paddedAddr + paddedAmount

	ctx := context.Background()
	contract := port.TxContractData{
		Type:            "TriggerSmartContract",
		ContractAddress: USDTContractAddress,
		Data:            data,
	}

	result := monitor.processTRC20FromData(ctx, "tx_not_monitored", contract)
	assert.False(t, result, "收款地址不在监控列表时应跳过")
}

// TestProcessTRC20FromData_金额为零跳过 验证金额为零时跳过
func TestProcessTRC20FromData_金额为零跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	kg := NewKeyGenerator()
	realAddress, _, err := kg.GenerateAddress()
	require.NoError(t, err)
	monitor.AddAddress(realAddress, 1)

	decoded := base58Decode(realAddress)
	addrHex := hex.EncodeToString(decoded[1:21])
	paddedAddr := fmt.Sprintf("%064s", addrHex)
	paddedAmount := fmt.Sprintf("%064x", big.NewInt(0))
	data := transferSelector + paddedAddr + paddedAmount

	ctx := context.Background()
	contract := port.TxContractData{
		Type:            "TriggerSmartContract",
		ContractAddress: USDTContractAddress,
		Data:            data,
	}

	result := monitor.processTRC20FromData(ctx, "tx_zero_amount", contract)
	assert.False(t, result, "金额为零时应跳过")
}

// TestProcessTRXFromData_金额验证 验证 TRX 充值的金额边界处理
func TestProcessTRXFromData_金额验证(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitorAddr := "TMonitoredAddress123456789012345"
	monitor := newTestMonitor(logger)
	monitor.AddAddress(monitorAddr, 1)

	tests := []struct {
		name     string
		amount   int64
		expected bool
	}{
		{name: "零金额", amount: 0, expected: false},
		{name: "负金额", amount: -1000000, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			contract := port.TxContractData{
				Type:         "TransferContract",
				OwnerAddress: "TSenderAddress1234567890123456",
				ToAddress:    monitorAddr,
				Amount:       tt.amount,
			}
			result := monitor.processTRXFromData(ctx, "tx_amount_test", contract)
			assert.Equal(t, tt.expected, result, "金额 %d 的处理结果应为 %v", tt.amount, tt.expected)
		})
	}
}

// TestProcessTRXFromData_地址不在监控列表 验证非监控地址被正确跳过
func TestProcessTRXFromData_地址不在监控列表(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	ctx := context.Background()
	contract := port.TxContractData{
		Type:         "TransferContract",
		OwnerAddress: "TSenderAddress1234567890123456",
		ToAddress:    "TNotMonitoredAddr1234567890123",
		Amount:       1000000,
	}

	result := monitor.processTRXFromData(ctx, "tx_not_monitored", contract)
	assert.False(t, result, "非监控地址的交易不应被处理")
}

// --- isMonitoredAddress 测试 ---

// TestIsMonitoredAddress 验证地址查询的正确性
func TestIsMonitoredAddress(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	monitor.AddAddress("TExistingAddr12345678901234567", 100)

	assert.True(t, monitor.isMonitoredAddress("TExistingAddr12345678901234567"), "已添加的地址应返回 true")
	assert.False(t, monitor.isMonitoredAddress("TNotExistingAddr23456789012345"), "未添加的地址应返回 false")
	assert.False(t, monitor.isMonitoredAddress(""), "空地址应返回 false")
}

// --- AddAddress / addressCount / SetPollInterval 测试 ---

// TestDepositMonitor_AddAddress 验证动态添加监控地址
func TestDepositMonitor_AddAddress(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	tests := []struct {
		name    string
		address string
		userID  uint64
		stored  bool
	}{
		{name: "正常添加地址", address: "TValidAddress12345678901234567", userID: 100, stored: true},
		{name: "空地址不添加", address: "", userID: 200, stored: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor.AddAddress(tt.address, tt.userID)
			if tt.stored {
				assert.True(t, monitor.isMonitoredAddress(tt.address), "地址应被存储")
			}
		})
	}
}

// TestDepositMonitor_addressCount 验证 addressCount 正确计数
func TestDepositMonitor_addressCount(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	assert.Equal(t, 0, monitor.addressCount(), "初始地址数量应为 0")

	monitor.AddAddress("TAddr1", 1)
	monitor.AddAddress("TAddr2", 2)
	monitor.AddAddress("TAddr3", 3)
	assert.Equal(t, 3, monitor.addressCount(), "添加 3 个地址后数量应为 3")

	// 重复添加同一地址不应增加计数
	monitor.AddAddress("TAddr1", 10)
	assert.Equal(t, 3, monitor.addressCount(), "重复添加同一地址后数量仍应为 3")
}

// TestDepositMonitor_SetPollInterval 验证轮询间隔设置
func TestDepositMonitor_SetPollInterval(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	assert.Equal(t, defaultBlockPollInterval, monitor.pollInterval, "默认轮询间隔应为 3 秒")

	monitor.SetPollInterval(5 * defaultBlockPollInterval)
	assert.Equal(t, 5*defaultBlockPollInterval, monitor.pollInterval, "轮询间隔应被更新")
}

// TestNewDepositMonitor_默认合约地址 验证未指定合约地址时使用默认值
func TestNewDepositMonitor_默认合约地址(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, nil, "", logger)
	assert.Equal(t, USDTContractAddress, monitor.usdtContract, "未指定合约地址时应使用默认 USDT 合约地址")
}

// TestNewDepositMonitor_自定义合约地址 验证指定自定义合约地址生效
func TestNewDepositMonitor_自定义合约地址(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	customContract := "TCustomContractAddr123456789012"
	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, nil, customContract, logger)
	assert.Equal(t, customContract, monitor.usdtContract, "应使用自定义合约地址")
}

// TestNewDepositMonitor_addrCache已初始化 验证 addrCache 被正确初始化
func TestNewDepositMonitor_addrCache已初始化(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	assert.NotPanics(t, func() {
		monitor.AddAddress("TTestAddr", 1)
	}, "addrCache 应已被初始化为非 nil map")
}

// TestProcessBlockData_混合交易类型 验证区块内包含多种交易类型的处理
func TestProcessBlockData_混合交易类型(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)
	monitor.AddAddress("TMonitoredAddr00001234567890123", 1)

	block := &port.BlockData{
		Number: 500,
		Transactions: []port.BlockTxData{
			{TxID: "tx_001", Success: true, Contracts: []port.TxContractData{
				{Type: "TransferContract", ToAddress: "TOtherAddr000012345678901234567", Amount: 1000000},
			}},
			{TxID: "tx_002", Success: false, Contracts: []port.TxContractData{
				{Type: "TransferContract", ToAddress: "TMonitoredAddr00001234567890123", Amount: 2000000},
			}},
			{TxID: "tx_003", Success: true, Contracts: []port.TxContractData{
				{Type: "UnknownContract"},
			}},
			{TxID: "tx_004", Success: true, Contracts: []port.TxContractData{
				{Type: "TriggerSmartContract", ContractAddress: "TOtherToken123456789012345678901",
					Data: "a9059cbb" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" + "0000000000000000000000000000000000000000000000000000000000100000"},
			}},
		},
	}

	ctx := context.Background()
	count := monitor.processBlockData(ctx, block)
	assert.Equal(t, 0, count, "不满足条件的交易不应被计为充值")
}

// --- 并发安全测试 ---

// TestDepositMonitor_AddAddress_并发安全 验证并发添加地址
func TestDepositMonitor_AddAddress_并发安全(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	var wg sync.WaitGroup
	count := 100

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			addr := fmt.Sprintf("TAddr%040d", idx)
			monitor.AddAddress(addr, uint64(idx))
		}(i)
	}

	wg.Wait()
	assert.Equal(t, count, monitor.addressCount(), "并发添加后应存储 %d 个地址", count)
}

// TestDepositMonitor_并发读写安全 验证并发读取和写入地址缓存
func TestDepositMonitor_并发读写安全(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	monitor := newTestMonitor(logger)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			monitor.AddAddress(fmt.Sprintf("TWriter%040d", idx), uint64(idx))
		}(i)
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			monitor.isMonitoredAddress(fmt.Sprintf("TWriter%040d", idx))
			monitor.addressCount()
		}(i)
	}

	wg.Wait()
	assert.GreaterOrEqual(t, monitor.addressCount(), 0, "地址数量应非负")
}

// --- handleDeposit / sendNotification 测试 ---

// mockDepositUseCase 模拟 DepositUseCase 行为
// 由于 DepositUseCase 是具体结构体而非接口，我们无法直接 mock
// 但可以测试 handleDeposit 中依赖 scanner.GetTransactionReceipt 的部分
// 以及 sendNotification 中依赖 userRepo 和 notifySvc 的部分

// mockUserRepo 实现 port.UserRepository 用于测试
type mockUserRepo struct {
	users map[uint64]*entity.User
	err   error
}

func (m *mockUserRepo) Create(_ context.Context, _ *entity.User) error {
	return nil
}
func (m *mockUserRepo) FindByID(_ context.Context, id uint64) (*entity.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, nil
}
func (m *mockUserRepo) FindByTelegramID(_ context.Context, _ int64) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindByUsername(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Update(_ context.Context, _ *entity.User) error { return nil }
func (m *mockUserRepo) List(_ context.Context, _, _ int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) CountAll(_ context.Context) (int64, error) { return 0, nil }

// mockNotifySvc 实现 port.NotificationService 用于测试
type mockNotifySvc struct {
	depositCalls []depositNotifyCall
}

type depositNotifyCall struct {
	TelegramID int64
	Currency   string
	Amount     string
	TxHash     string
	NewBalance string
}

func (m *mockNotifySvc) SendDepositNotification(_ context.Context, telegramID int64, currency, amount, txHash, newBalance string) {
	m.depositCalls = append(m.depositCalls, depositNotifyCall{
		TelegramID: telegramID,
		Currency:   currency,
		Amount:     amount,
		TxHash:     txHash,
		NewBalance: newBalance,
	})
}
func (m *mockNotifySvc) SendWithdrawalResult(_ context.Context, _ int64, _ bool, _, _, _, _ string) {
}
func (m *mockNotifySvc) SendTransferReceived(_ context.Context, _ int64, _, _, _, _ string) {}
func (m *mockNotifySvc) SendPaymentRequest(_ context.Context, _ int64, _, _, _ string, _ uint64) {
}
func (m *mockNotifySvc) SendSecurityAlert(_ context.Context, _ int64, _, _ string) {}

// mockWalletRepo 实现 port.WalletRepository 用于测试
type mockWalletRepo struct {
	wallets     []*entity.Wallet
	walletsErr  error
	afterWallets    []*entity.Wallet
	afterWalletsErr error
}

func (m *mockWalletRepo) Create(_ context.Context, _ *entity.Wallet) error { return nil }
func (m *mockWalletRepo) FindByID(_ context.Context, _ uint64) (*entity.Wallet, error) {
	return nil, nil
}
func (m *mockWalletRepo) FindByUserIDAndCurrency(_ context.Context, _ uint64, _ string) (*entity.Wallet, error) {
	return nil, nil
}
func (m *mockWalletRepo) FindByUserID(_ context.Context, _ uint64) ([]*entity.Wallet, error) {
	return nil, nil
}
func (m *mockWalletRepo) FindByDepositAddress(_ context.Context, _ string) (*entity.Wallet, error) {
	return nil, nil
}
func (m *mockWalletRepo) UpdateBalance(_ context.Context, _ uint64, _ decimal.Decimal) error {
	return nil
}
func (m *mockWalletRepo) FreezeBalance(_ context.Context, _ uint64, _ decimal.Decimal) error {
	return nil
}
func (m *mockWalletRepo) UnfreezeBalance(_ context.Context, _ uint64, _ decimal.Decimal) error {
	return nil
}
func (m *mockWalletRepo) UpdateDepositAddress(_ context.Context, _ uint64, _ string) error {
	return nil
}
func (m *mockWalletRepo) ListAllWithDepositAddress(_ context.Context) ([]*entity.Wallet, error) {
	return m.wallets, m.walletsErr
}
func (m *mockWalletRepo) ListDepositAddressesCreatedAfter(_ context.Context, _ time.Time) ([]*entity.Wallet, error) {
	return m.afterWallets, m.afterWalletsErr
}

// --- getLastProcessedBlock / saveLastProcessedBlock 测试 ---

// TestGetLastProcessedBlock_正常获取 验证从缓存正确获取区块号
func TestGetLastProcessedBlock_正常获取(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := newMockCacheRepo()
	cache.data["tron:monitor:last_block"] = "12345"

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	blockNum := monitor.getLastProcessedBlock(ctx)
	assert.Equal(t, int64(12345), blockNum, "应正确获取缓存的区块号")
}

// TestGetLastProcessedBlock_缓存为空 验证缓存为空时返回 0
func TestGetLastProcessedBlock_缓存为空(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	blockNum := monitor.getLastProcessedBlock(ctx)
	assert.Equal(t, int64(0), blockNum, "缓存为空时应返回 0")
}

// TestGetLastProcessedBlock_无效值 验证非数字值时返回 0
func TestGetLastProcessedBlock_无效值(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := newMockCacheRepo()
	cache.data["tron:monitor:last_block"] = "not_a_number"

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	blockNum := monitor.getLastProcessedBlock(ctx)
	assert.Equal(t, int64(0), blockNum, "非数字值时应返回 0")
}

// TestSaveLastProcessedBlock_保存成功 验证区块号正确保存到缓存
func TestSaveLastProcessedBlock_保存成功(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	monitor.saveLastProcessedBlock(ctx, 99999)

	assert.Equal(t, "99999", cache.data["tron:monitor:last_block"], "区块号应正确保存")
}

// TestSaveLastProcessedBlock_往返一致 验证保存后读取的值一致
func TestSaveLastProcessedBlock_往返一致(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	monitor.saveLastProcessedBlock(ctx, 54321)

	blockNum := monitor.getLastProcessedBlock(ctx)
	assert.Equal(t, int64(54321), blockNum, "保存后读取的值应一致")
}

// --- fullRefreshAddressCache 测试 ---

// TestFullRefreshAddressCache_正常加载 验证全量刷新正确加载钱包地址
func TestFullRefreshAddressCache_正常加载(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		wallets: []*entity.Wallet{
			{UserID: 1, DepositAddress: "TAddr1"},
			{UserID: 2, DepositAddress: "TAddr2"},
			{UserID: 3, DepositAddress: ""},          // 空地址应被跳过
			{UserID: 4, DepositAddress: "TAddr4"},
		},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	monitor.fullRefreshAddressCache(ctx)

	assert.Equal(t, 3, monitor.addressCount(), "应加载 3 个有效地址 (跳过空地址)")
	assert.True(t, monitor.isMonitoredAddress("TAddr1"), "TAddr1 应在缓存中")
	assert.True(t, monitor.isMonitoredAddress("TAddr2"), "TAddr2 应在缓存中")
	assert.False(t, monitor.isMonitoredAddress(""), "空地址不应在缓存中")
	assert.True(t, monitor.isMonitoredAddress("TAddr4"), "TAddr4 应在缓存中")
}

// TestFullRefreshAddressCache_仓库错误 验证仓库返回错误时不清空缓存
func TestFullRefreshAddressCache_仓库错误(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		walletsErr: fmt.Errorf("数据库连接失败"),
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)
	// 预先添加一些地址
	monitor.AddAddress("TExisting", 1)

	ctx := context.Background()
	monitor.fullRefreshAddressCache(ctx)

	// 仓库错误时不应修改现有缓存
	assert.Equal(t, 1, monitor.addressCount(), "仓库错误时不应清空现有缓存")
	assert.True(t, monitor.isMonitoredAddress("TExisting"), "现有地址应保留")
}

// TestFullRefreshAddressCache_替换旧缓存 验证全量刷新会替换旧缓存数据
func TestFullRefreshAddressCache_替换旧缓存(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		wallets: []*entity.Wallet{
			{UserID: 10, DepositAddress: "TNew1"},
			{UserID: 20, DepositAddress: "TNew2"},
		},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)
	// 预先有旧数据
	monitor.AddAddress("TOld1", 1)
	monitor.AddAddress("TOld2", 2)
	monitor.AddAddress("TOld3", 3)

	ctx := context.Background()
	monitor.fullRefreshAddressCache(ctx)

	assert.Equal(t, 2, monitor.addressCount(), "全量刷新后应只有新数据")
	assert.True(t, monitor.isMonitoredAddress("TNew1"), "新地址应在缓存中")
	assert.True(t, monitor.isMonitoredAddress("TNew2"), "新地址应在缓存中")
	assert.False(t, monitor.isMonitoredAddress("TOld1"), "旧地址应被替换")
}

// --- incrementalRefreshAddressCache 测试 ---

// TestIncrementalRefreshAddressCache_首次刷新降级全量 验证 lastRefreshAt 为零值时降级全量刷新
func TestIncrementalRefreshAddressCache_首次刷新降级全量(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		wallets: []*entity.Wallet{
			{UserID: 1, DepositAddress: "TAddr1"},
		},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)
	// lastRefreshAt 默认为零值

	ctx := context.Background()
	monitor.incrementalRefreshAddressCache(ctx)

	assert.Equal(t, 1, monitor.addressCount(), "首次增量刷新应降级为全量刷新")
	assert.True(t, monitor.isMonitoredAddress("TAddr1"))
}

// TestIncrementalRefreshAddressCache_增量新增 验证增量只添加新地址不覆盖旧数据
func TestIncrementalRefreshAddressCache_增量新增(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		afterWallets: []*entity.Wallet{
			{UserID: 100, DepositAddress: "TNewAddr"},
		},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)
	monitor.AddAddress("TOldAddr", 1)

	// 设置 lastRefreshAt 为非零值，使增量刷新路径生效
	monitor.addrMu.Lock()
	monitor.lastRefreshAt = time.Now().Add(-10 * time.Minute)
	monitor.addrMu.Unlock()

	ctx := context.Background()
	monitor.incrementalRefreshAddressCache(ctx)

	assert.Equal(t, 2, monitor.addressCount(), "增量刷新后应有旧+新共 2 个地址")
	assert.True(t, monitor.isMonitoredAddress("TOldAddr"), "旧地址应保留")
	assert.True(t, monitor.isMonitoredAddress("TNewAddr"), "新地址应被添加")
}

// TestIncrementalRefreshAddressCache_无新增 验证无新地址时不修改缓存
func TestIncrementalRefreshAddressCache_无新增(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		afterWallets: []*entity.Wallet{},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)
	monitor.AddAddress("TExisting", 1)

	monitor.addrMu.Lock()
	monitor.lastRefreshAt = time.Now().Add(-10 * time.Minute)
	monitor.addrMu.Unlock()

	ctx := context.Background()
	monitor.incrementalRefreshAddressCache(ctx)

	assert.Equal(t, 1, monitor.addressCount(), "无新增时地址数量不变")
}

// TestIncrementalRefreshAddressCache_错误降级全量 验证增量查询失败时降级全量
func TestIncrementalRefreshAddressCache_错误降级全量(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	walletRepo := &mockWalletRepo{
		afterWalletsErr: fmt.Errorf("查询失败"),
		wallets: []*entity.Wallet{
			{UserID: 1, DepositAddress: "TFullRefresh"},
		},
	}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, walletRepo, nil, nil, newMockCacheRepo(), "", logger)

	monitor.addrMu.Lock()
	monitor.lastRefreshAt = time.Now().Add(-10 * time.Minute)
	monitor.addrMu.Unlock()

	ctx := context.Background()
	monitor.incrementalRefreshAddressCache(ctx)

	assert.Equal(t, 1, monitor.addressCount(), "增量错误应降级全量刷新")
	assert.True(t, monitor.isMonitoredAddress("TFullRefresh"))
}

// --- sendNotification 测试 ---

// TestSendNotification_正常发送 验证通知正确发送
func TestSendNotification_正常发送(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	userRepo := &mockUserRepo{
		users: map[uint64]*entity.User{
			1: {ID: 1, TelegramID: 123456789},
		},
	}
	notifySvc := &mockNotifySvc{}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, userRepo, notifySvc, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	notification := &app.DepositNotification{
		UserID:     1,
		Currency:   entity.CurrencyUSDT,
		Amount:     decimal.NewFromFloat(100.5),
		TxHash:     "abcdef1234567890",
		NewBalance: decimal.NewFromFloat(200.0),
	}

	monitor.sendNotification(ctx, notification)

	require.Equal(t, 1, len(notifySvc.depositCalls), "应发送 1 次充值通知")
	call := notifySvc.depositCalls[0]
	assert.Equal(t, int64(123456789), call.TelegramID, "Telegram ID 应正确")
	assert.Equal(t, entity.CurrencyUSDT, call.Currency)
	assert.Equal(t, "100.5", call.Amount)
	assert.Equal(t, "abcdef1234567890", call.TxHash)
	assert.Equal(t, "200", call.NewBalance)
}

// TestSendNotification_nil通知服务 验证通知服务为 nil 时安全返回
func TestSendNotification_nil通知服务(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, nil, nil, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	notification := &app.DepositNotification{UserID: 1, Currency: "TRX"}

	// 不应 panic
	assert.NotPanics(t, func() {
		monitor.sendNotification(ctx, notification)
	}, "nil 通知服务时不应 panic")
}

// TestSendNotification_用户不存在 验证用户查找失败时安全处理
func TestSendNotification_用户不存在(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	userRepo := &mockUserRepo{
		users: map[uint64]*entity.User{}, // 空用户表
	}
	notifySvc := &mockNotifySvc{}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, userRepo, notifySvc, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	notification := &app.DepositNotification{UserID: 999}

	monitor.sendNotification(ctx, notification)

	assert.Equal(t, 0, len(notifySvc.depositCalls), "用户不存在时不应发送通知")
}

// TestSendNotification_用户查询错误 验证用户仓库返回错误时安全处理
func TestSendNotification_用户查询错误(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	userRepo := &mockUserRepo{
		err: fmt.Errorf("数据库连接失败"),
	}
	notifySvc := &mockNotifySvc{}

	monitor := NewDepositMonitor(&mockBlockScanner{}, nil, nil, userRepo, notifySvc, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	notification := &app.DepositNotification{UserID: 1}

	monitor.sendNotification(ctx, notification)

	assert.Equal(t, 0, len(notifySvc.depositCalls), "用户查询失败时不应发送通知")
}

// --- scanNewBlocks / scanBlocksBatch / scanBlocksSequential 测试 ---

// TestScanNewBlocks_无新区块 验证当前区块号 <= 上次处理区块号时不扫描
func TestScanNewBlocks_无新区块(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		currentBlockNum: 100,
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanNewBlocks(ctx, &lastBlock)

	assert.Equal(t, int64(100), lastBlock, "无新区块时 lastBlock 不应改变")
}

// TestScanNewBlocks_获取当前区块失败 验证获取区块失败时不扫描
func TestScanNewBlocks_获取当前区块失败(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		currentBlockErr: fmt.Errorf("网络错误"),
	}

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, newMockCacheRepo(), "", logger)

	ctx := context.Background()
	lastBlock := int64(50)
	monitor.scanNewBlocks(ctx, &lastBlock)

	assert.Equal(t, int64(50), lastBlock, "获取区块失败时 lastBlock 不应改变")
}

// TestScanBlocksSequential_正常逐块扫描 验证逐块扫描正确处理区块
func TestScanBlocksSequential_正常逐块扫描(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blocks: map[int64]*port.BlockData{
			101: {Number: 101, Transactions: nil},
			102: {Number: 102, Transactions: nil},
			103: {Number: 103, Transactions: nil},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanBlocksSequential(ctx, &lastBlock, 101, 103)

	assert.Equal(t, int64(103), lastBlock, "扫描完成后 lastBlock 应更新到 103")
	assert.Equal(t, "103", cache.data["tron:monitor:last_block"], "区块进度应保存到缓存")
}

// TestScanBlocksSequential_获取区块失败停止 验证获取某个区块失败时停止扫描
func TestScanBlocksSequential_获取区块失败停止(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blocks: map[int64]*port.BlockData{
			101: {Number: 101},
			// 102 不存在，GetBlockByNum 返回错误
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanBlocksSequential(ctx, &lastBlock, 101, 105)

	assert.Equal(t, int64(101), lastBlock, "获取 102 失败后应停止在 101")
}

// TestScanBlocksSequential_ctx取消中断 验证 context 取消时停止扫描
func TestScanBlocksSequential_ctx取消中断(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blocks: map[int64]*port.BlockData{
			101: {Number: 101},
			102: {Number: 102},
			103: {Number: 103},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	lastBlock := int64(100)
	monitor.scanBlocksSequential(ctx, &lastBlock, 101, 103)

	assert.Equal(t, int64(100), lastBlock, "context 取消后 lastBlock 不应改变")
}

// TestScanBlocksBatch_正常批量扫描 验证批量模式正确处理区块
func TestScanBlocksBatch_正常批量扫描(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blockRangeResult: []*port.BlockData{
			{Number: 101},
			{Number: 102},
			{Number: 103},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanBlocksBatch(ctx, &lastBlock, 101, 103)

	assert.Equal(t, int64(103), lastBlock, "批量扫描后 lastBlock 应更新到 103")
	assert.Equal(t, "103", cache.data["tron:monitor:last_block"], "区块进度应保存到缓存")
}

// TestScanBlocksBatch_获取失败降级逐块 验证批量获取失败时降级为逐块模式
func TestScanBlocksBatch_获取失败降级逐块(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blockRangeErr: fmt.Errorf("批量获取失败"),
		blocks: map[int64]*port.BlockData{
			101: {Number: 101},
			102: {Number: 102},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanBlocksBatch(ctx, &lastBlock, 101, 102)

	assert.Equal(t, int64(102), lastBlock, "降级逐块模式后应处理到 102")
}

// TestScanBlocksBatch_空区块跳过 验证批量结果中 nil 区块被跳过
func TestScanBlocksBatch_空区块跳过(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blockRangeResult: []*port.BlockData{
			nil,
			{Number: 0}, // Number 为 0 应跳过
			{Number: 105},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanBlocksBatch(ctx, &lastBlock, 101, 105)

	assert.Equal(t, int64(105), lastBlock, "应跳过空区块，处理到 105")
}

// TestScanBlocksBatch_ctx取消中断 验证 context 取消时保存已处理进度
func TestScanBlocksBatch_ctx取消中断(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		blockRangeResult: []*port.BlockData{
			{Number: 101},
			{Number: 102},
			{Number: 103},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	lastBlock := int64(100)
	monitor.scanBlocksBatch(ctx, &lastBlock, 101, 103)

	// ctx 取消后，可能已处理部分区块或未处理任何区块
	// 关键是不 panic 且进度已保存
	assert.GreaterOrEqual(t, lastBlock, int64(100), "lastBlock 不应回退")
}

// TestScanNewBlocks_落后少量逐块模式 验证落后 <= batchThreshold 块时使用逐块扫描
func TestScanNewBlocks_落后少量逐块模式(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		currentBlockNum: 103, // 落后 3 块 (101, 102, 103)，<= batchThreshold(5)
		blocks: map[int64]*port.BlockData{
			101: {Number: 101},
			102: {Number: 102},
			103: {Number: 103},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanNewBlocks(ctx, &lastBlock)

	assert.Equal(t, int64(103), lastBlock, "逐块扫描后应更新到 103")
}

// TestScanNewBlocks_落后大量批量模式 验证落后 > batchThreshold 块时使用批量模式
func TestScanNewBlocks_落后大量批量模式(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	scanner := &mockBlockScanner{
		currentBlockNum: 110, // 落后 10 块，> batchThreshold(5)
		blockRangeResult: []*port.BlockData{
			{Number: 101}, {Number: 102}, {Number: 103},
			{Number: 104}, {Number: 105}, {Number: 106},
			{Number: 107}, {Number: 108}, {Number: 109},
			{Number: 110},
		},
	}
	cache := newMockCacheRepo()

	monitor := NewDepositMonitor(scanner, nil, nil, nil, nil, cache, "", logger)

	ctx := context.Background()
	lastBlock := int64(100)
	monitor.scanNewBlocks(ctx, &lastBlock)

	assert.Equal(t, int64(110), lastBlock, "批量扫描后应更新到 110")
}

// --- 编译验证: 确保使用的导入不产生 unused import 错误 ---
var (
	_ = decimal.NewFromInt
	_ = entity.CurrencyTRX
	_ = (*app.DepositNotification)(nil)
)
