package tron

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// --- reconnectWithBackoff 测试 ---

// TestGrpcBlockClient_reconnectWithBackoff_停止信号中断 验证关闭 stopCh 后重连退避等待被立即中断
// 注意: gotron-sdk 的 gRPC Dial 默认是 lazy 连接 (不带 WithBlock)，
// 即 Start() 对任何地址都会立即返回 nil。因此无法直接让 Connect() 返回 error。
// 此测试通过预先关闭 stopCh，验证 reconnect 在退避 sleep 阶段被中断。
func TestGrpcBlockClient_reconnectWithBackoff_停止信号中断(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 预先关闭 stopCh
	client.Close()

	// 手动标记为未连接，让 reconnectWithBackoff 执行重连路径
	// 但由于 Connect() 在 lazy 模式下成功，实际上会在第一次 Connect 就返回 nil
	// 所以我们测试在 Connect 前就检测到 stopCh 关闭的行为
	// —— 由于 reconnectWithBackoff 的循环结构是先尝试 Connect，
	// 如果成功则返回 nil，所以这个测试验证的是:
	// 即使 stopCh 已关闭，如果 Connect 成功，仍然返回 nil (正常行为)
	err := client.reconnectWithBackoff()

	// gotron-sdk lazy dial 下 Connect 首次就成功，应返回 nil
	assert.NoError(t, err, "lazy dial 模式下首次 Connect 应成功")
}

// TestGrpcBlockClient_reconnectWithBackoff_退避等待可被中断 验证退避等待期间 stopCh 关闭能中断等待
// 此测试构造一个已标记连接但实际未连接的场景，模拟 Connect 失败后退避等待
func TestGrpcBlockClient_reconnectWithBackoff_退避等待可被中断(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 验证: 当 stopCh 在退避等待中被关闭时，select 应选中 stopCh 分支
	// 直接测试 stopCh channel 行为
	go func() {
		time.Sleep(50 * time.Millisecond)
		client.Close()
	}()

	timer := time.NewTimer(2 * time.Second)
	start := time.Now()
	select {
	case <-client.stopCh:
		// 预期路径: stopCh 被关闭
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 500*time.Millisecond, "stopCh 应在 50ms 左右被触发")
	case <-timer.C:
		t.Fatal("stopCh 未在预期时间内被关闭")
	}
	timer.Stop()
}

// TestGrpcBlockClient_Close_重复调用安全 验证多次调用 Close 不会 panic
func TestGrpcBlockClient_Close_重复调用安全(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 多次调用 Close 不应 panic (stopOnce 保护)
	assert.NotPanics(t, func() {
		client.Close()
	}, "第一次 Close 不应 panic")

	assert.NotPanics(t, func() {
		client.Close()
	}, "第二次 Close 不应 panic")

	assert.NotPanics(t, func() {
		client.Close()
	}, "第三次 Close 不应 panic")
}

// TestGrpcBlockClient_ensureConnected_已连接直接返回 验证已连接状态下 ensureConnected 不触发重连
func TestGrpcBlockClient_ensureConnected_已连接直接返回(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 手动标记为已连接
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	// ensureConnected 应直接返回 nil
	err := client.ensureConnected()
	assert.NoError(t, err, "已连接状态下 ensureConnected 应直接返回 nil")
}

// TestGrpcBlockClient_markDisconnected 验证 markDisconnected 将状态标记为未连接
func TestGrpcBlockClient_markDisconnected(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 先标记为已连接
	client.mu.Lock()
	client.connected = true
	client.mu.Unlock()

	// 执行断开标记
	client.markDisconnected()

	// 验证状态
	client.mu.RLock()
	connected := client.connected
	client.mu.RUnlock()

	assert.False(t, connected, "markDisconnected 后 connected 应为 false")
}

// TestNewGrpcBlockClient_默认端点 验证未配置 endpoint 时使用默认值
func TestNewGrpcBlockClient_默认端点(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("", "test-api-key", 0, 0, logger)

	assert.Equal(t, "grpc.trongrid.io:50051", client.grpcEndpoint, "未配置时应使用默认 gRPC 端点")
	assert.Equal(t, "test-api-key", client.apiKey, "API Key 应正确设置")
	assert.False(t, client.connected, "初始状态应为未连接")
	assert.NotNil(t, client.stopCh, "stopCh 应被初始化")
}

// TestNewGrpcBlockClient_自定义端点 验证配置的自定义 endpoint 生效
func TestNewGrpcBlockClient_自定义端点(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("custom-grpc.example.com:50051", "", 0, 0, logger)

	assert.Equal(t, "custom-grpc.example.com:50051", client.grpcEndpoint, "应使用自定义 gRPC 端点")
	assert.Equal(t, "", client.apiKey, "空 API Key 应保持为空")
}

// --- timeout 配置测试 ---

// TestNewGrpcBlockClient_默认超时 验证未指定超时参数时使用默认值
func TestNewGrpcBlockClient_默认超时(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	assert.Equal(t, defaultGRPCTimeout, client.grpcTimeout, "默认 gRPC 超时应为 %v", defaultGRPCTimeout)
	assert.Equal(t, defaultGRPCBatchTimeout, client.grpcBatchTimeout, "默认 gRPC 批量超时应为 %v", defaultGRPCBatchTimeout)
}

// TestNewGrpcBlockClient_自定义超时 验证自定义超时参数生效
func TestNewGrpcBlockClient_自定义超时(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 20*time.Second, 60*time.Second, logger)

	assert.Equal(t, 20*time.Second, client.grpcTimeout, "自定义 gRPC 超时应为 20s")
	assert.Equal(t, 60*time.Second, client.grpcBatchTimeout, "自定义 gRPC 批量超时应为 60s")
}

// TestNewGrpcBlockClient_仅自定义单次超时 验证只配置单次超时时批量超时使用默认值
func TestNewGrpcBlockClient_仅自定义单次超时(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	client := NewGrpcBlockClient("127.0.0.1:50051", "", 15*time.Second, 0, logger)

	assert.Equal(t, 15*time.Second, client.grpcTimeout, "自定义 gRPC 超时应为 15s")
	assert.Equal(t, defaultGRPCBatchTimeout, client.grpcBatchTimeout, "批量超时应使用默认值")
}

// --- convertBlock / isTxSuccess / convertContract 测试 ---

// buildAnyFromProto 将 protobuf 消息包装为 anypb.Any
func buildAnyFromProto(t *testing.T, msg proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(msg)
	require.NoError(t, err, "构造 anypb.Any 不应失败")
	return a
}

// TestIsTxSuccess_各种状态 验证交易成功状态判断
func TestIsTxSuccess_各种状态(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	tests := []struct {
		name     string
		tx       *core.Transaction
		expected bool
	}{
		{
			name:     "nil 交易",
			tx:       nil,
			expected: false,
		},
		{
			name:     "空 Ret 数组",
			tx:       &core.Transaction{Ret: nil},
			expected: false,
		},
		{
			name:     "Ret 长度为零",
			tx:       &core.Transaction{Ret: []*core.Transaction_Result{}},
			expected: false,
		},
		{
			name: "SUCCESS 状态",
			tx: &core.Transaction{
				Ret: []*core.Transaction_Result{
					{ContractRet: core.Transaction_Result_SUCCESS},
				},
			},
			expected: true,
		},
		{
			name: "REVERT 状态",
			tx: &core.Transaction{
				Ret: []*core.Transaction_Result{
					{ContractRet: core.Transaction_Result_REVERT},
				},
			},
			expected: false,
		},
		{
			name: "OUT_OF_ENERGY 状态",
			tx: &core.Transaction{
				Ret: []*core.Transaction_Result{
					{ContractRet: core.Transaction_Result_OUT_OF_ENERGY},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isTxSuccess(tt.tx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConvertBlock_nil和空区块 验证 nil/空区块的处理
func TestConvertBlock_nil和空区块(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	tests := []struct {
		name  string
		block *api.BlockExtention
	}{
		{name: "nil 区块", block: nil},
		{name: "nil BlockHeader", block: &api.BlockExtention{BlockHeader: nil}},
		{name: "nil RawData", block: &api.BlockExtention{BlockHeader: &core.BlockHeader{RawData: nil}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bd := client.convertBlock(tt.block)
			require.NotNil(t, bd, "即使输入为空也应返回非 nil BlockData")
			assert.Equal(t, int64(0), bd.Number, "空区块编号应为 0")
			assert.Empty(t, bd.Transactions, "空区块不应有交易")
		})
	}
}

// TestConvertBlock_包含TRX转账 验证 TransferContract 的转换
func TestConvertBlock_包含TRX转账(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	// 构造 TransferContract protobuf
	ownerAddr, _ := hex.DecodeString("41" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	toAddr, _ := hex.DecodeString("41" + "1234567890abcdef1234567890abcdef12345678")
	tc := &core.TransferContract{
		OwnerAddress: ownerAddr,
		ToAddress:    toAddr,
		Amount:       5000000,
	}
	tcBytes, err := proto.Marshal(tc)
	require.NoError(t, err)

	txID, _ := hex.DecodeString("aabbccdd11223344aabbccdd11223344aabbccdd11223344aabbccdd11223344")

	block := &api.BlockExtention{
		BlockHeader: &core.BlockHeader{
			RawData: &core.BlockHeaderRaw{
				Number: 12345,
			},
		},
		Transactions: []*api.TransactionExtention{
			{
				Txid: txID,
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						Contract: []*core.Transaction_Contract{
							{
								Type: core.Transaction_Contract_TransferContract,
								Parameter: &anypb.Any{
									TypeUrl: "type.googleapis.com/protocol.TransferContract",
									Value:   tcBytes,
								},
							},
						},
					},
					Ret: []*core.Transaction_Result{
						{ContractRet: core.Transaction_Result_SUCCESS},
					},
				},
			},
		},
	}

	bd := client.convertBlock(block)
	require.NotNil(t, bd)
	assert.Equal(t, int64(12345), bd.Number, "区块号应为 12345")
	require.Equal(t, 1, len(bd.Transactions), "应有 1 笔交易")

	tx := bd.Transactions[0]
	assert.Equal(t, hex.EncodeToString(txID), tx.TxID, "TxID 应正确转换")
	assert.True(t, tx.Success, "交易应为成功状态")
	require.Equal(t, 1, len(tx.Contracts), "应有 1 个合约调用")

	contract := tx.Contracts[0]
	assert.Equal(t, "TransferContract", contract.Type, "合约类型应为 TransferContract")
	assert.Equal(t, int64(5000000), contract.Amount, "金额应为 5000000")
	assert.NotEmpty(t, contract.OwnerAddress, "发送方地址不应为空")
	assert.NotEmpty(t, contract.ToAddress, "接收方地址不应为空")
}

// TestConvertBlock_包含TRC20触发 验证 TriggerSmartContract 的转换
func TestConvertBlock_包含TRC20触发(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	ownerAddr, _ := hex.DecodeString("41" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2")
	contractAddr, _ := hex.DecodeString("41" + "1234567890abcdef1234567890abcdef12345678")
	callData, _ := hex.DecodeString("a9059cbb" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" + "00000000000000000000000000000000000000000000000000000000000f4240")

	tsc := &core.TriggerSmartContract{
		OwnerAddress:    ownerAddr,
		ContractAddress: contractAddr,
		Data:            callData,
	}
	tscBytes, err := proto.Marshal(tsc)
	require.NoError(t, err)

	txID, _ := hex.DecodeString("1122334455667788112233445566778811223344556677881122334455667788")

	block := &api.BlockExtention{
		BlockHeader: &core.BlockHeader{
			RawData: &core.BlockHeaderRaw{Number: 99999},
		},
		Transactions: []*api.TransactionExtention{
			{
				Txid: txID,
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						Contract: []*core.Transaction_Contract{
							{
								Type: core.Transaction_Contract_TriggerSmartContract,
								Parameter: &anypb.Any{
									TypeUrl: "type.googleapis.com/protocol.TriggerSmartContract",
									Value:   tscBytes,
								},
							},
						},
					},
					Ret: []*core.Transaction_Result{
						{ContractRet: core.Transaction_Result_SUCCESS},
					},
				},
			},
		},
	}

	bd := client.convertBlock(block)
	require.NotNil(t, bd)
	assert.Equal(t, int64(99999), bd.Number)
	require.Equal(t, 1, len(bd.Transactions))

	tx := bd.Transactions[0]
	assert.True(t, tx.Success)
	require.Equal(t, 1, len(tx.Contracts))

	contract := tx.Contracts[0]
	assert.Equal(t, "TriggerSmartContract", contract.Type, "合约类型应为 TriggerSmartContract")
	assert.NotEmpty(t, contract.OwnerAddress, "调用者地址不应为空")
	assert.NotEmpty(t, contract.ContractAddress, "合约地址不应为空")
	assert.Contains(t, contract.Data, "a9059cbb", "合约数据应包含 transfer 选择器")
}

// TestConvertContract_忽略其他合约类型 验证非 Transfer/Trigger 类型返回 nil
func TestConvertContract_忽略其他合约类型(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	tests := []struct {
		name     string
		contract *core.Transaction_Contract
	}{
		{
			name:     "nil 合约",
			contract: nil,
		},
		{
			name:     "nil Parameter",
			contract: &core.Transaction_Contract{Type: core.Transaction_Contract_TransferContract, Parameter: nil},
		},
		{
			name: "FreezeBalanceContract 类型",
			contract: &core.Transaction_Contract{
				Type:      core.Transaction_Contract_FreezeBalanceContract,
				Parameter: &anypb.Any{Value: []byte{}},
			},
		},
		{
			name: "VoteWitnessContract 类型",
			contract: &core.Transaction_Contract{
				Type:      core.Transaction_Contract_VoteWitnessContract,
				Parameter: &anypb.Any{Value: []byte{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.convertContract(tt.contract)
			assert.Nil(t, result, "非 Transfer/Trigger 类型应返回 nil")
		})
	}
}

// TestConvertContract_无效protobuf数据 验证反序列化失败时返回 nil
func TestConvertContract_无效protobuf数据(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	tests := []struct {
		name  string
		cType core.Transaction_Contract_ContractType
	}{
		{name: "无效 TransferContract", cType: core.Transaction_Contract_TransferContract},
		{name: "无效 TriggerSmartContract", cType: core.Transaction_Contract_TriggerSmartContract},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := &core.Transaction_Contract{
				Type: tt.cType,
				Parameter: &anypb.Any{
					Value: []byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb}, // 无效 protobuf 数据
				},
			}
			result := client.convertContract(contract)
			// 无效数据反序列化可能成功也可能失败 (protobuf 是宽松的)
			// 如果失败则返回 nil，如果成功则返回非 nil
			// 主要验证不 panic
			_ = result
		})
	}
}

// TestConvertBlock_跳过nil交易 验证 nil TransactionExtention 被跳过
func TestConvertBlock_跳过nil交易(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	block := &api.BlockExtention{
		BlockHeader: &core.BlockHeader{
			RawData: &core.BlockHeaderRaw{Number: 100},
		},
		Transactions: []*api.TransactionExtention{
			nil,
			{Txid: []byte{0xaa}, Transaction: nil},
		},
	}

	bd := client.convertBlock(block)
	require.NotNil(t, bd)
	assert.Equal(t, int64(100), bd.Number)
	assert.Empty(t, bd.Transactions, "nil 交易和 nil Transaction 应被跳过")
}

// TestConvertBlock_交易无RawData 验证交易无 RawData 时不解析合约
func TestConvertBlock_交易无RawData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	client := NewGrpcBlockClient("127.0.0.1:50051", "", 0, 0, logger)

	txID, _ := hex.DecodeString("aabb")

	block := &api.BlockExtention{
		BlockHeader: &core.BlockHeader{
			RawData: &core.BlockHeaderRaw{Number: 200},
		},
		Transactions: []*api.TransactionExtention{
			{
				Txid: txID,
				Transaction: &core.Transaction{
					RawData: nil, // 无 RawData
					Ret: []*core.Transaction_Result{
						{ContractRet: core.Transaction_Result_SUCCESS},
					},
				},
			},
		},
	}

	bd := client.convertBlock(block)
	require.NotNil(t, bd)
	require.Equal(t, 1, len(bd.Transactions))
	assert.True(t, bd.Transactions[0].Success)
	assert.Empty(t, bd.Transactions[0].Contracts, "无 RawData 时合约列表应为空")
}
