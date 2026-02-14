package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"

	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
)

const (
	// 默认超时 (配置未指定时使用)
	defaultGRPCTimeout      = 10 * time.Second
	defaultGRPCBatchTimeout = 30 * time.Second
)

// GrpcBlockClient 基于 gotron-sdk gRPC 的区块扫描客户端
// 实现 port.BlockScanner 接口
type GrpcBlockClient struct {
	grpcEndpoint string
	apiKey       string
	conn         *client.GrpcClient
	logger       *zap.Logger
	mu           sync.RWMutex
	connected    bool
	stopCh       chan struct{} // 停止信号，用于中断重连退避
	stopOnce     sync.Once    // 确保 stopCh 只关闭一次

	grpcTimeout      time.Duration // 单次 gRPC 调用超时
	grpcBatchTimeout time.Duration // 批量 gRPC 调用超时
}

// NewGrpcBlockClient 创建 gRPC 区块扫描客户端
// endpoint: gRPC 端点地址，空值使用默认 grpc.trongrid.io:50051
// apiKey: TronGrid API Key
// timeout: 单次 gRPC 调用超时，传 0 使用默认 10s
// batchTimeout: 批量 gRPC 调用超时，传 0 使用默认 30s
func NewGrpcBlockClient(endpoint, apiKey string, timeout, batchTimeout time.Duration, logger *zap.Logger) *GrpcBlockClient {
	if endpoint == "" {
		endpoint = "grpc.trongrid.io:50051"
	}

	if timeout <= 0 {
		timeout = defaultGRPCTimeout
	}

	if batchTimeout <= 0 {
		batchTimeout = defaultGRPCBatchTimeout
	}

	return &GrpcBlockClient{
		grpcEndpoint:     endpoint,
		apiKey:           apiKey,
		logger:           logger,
		stopCh:           make(chan struct{}),
		grpcTimeout:      timeout,
		grpcBatchTimeout: batchTimeout,
	}
}

// Connect 建立 gRPC 连接
func (c *GrpcBlockClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		return nil
	}

	c.conn = client.NewGrpcClient(c.grpcEndpoint)
	// 使用系统 CA 证书建立 TLS 连接，保护 API Key 和区块数据传输安全
	if err := c.conn.Start(grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))); err != nil {
		return fmt.Errorf("gRPC TLS 连接失败 (%s): %w", c.grpcEndpoint, err)
	}

	if c.apiKey != "" {
		c.conn.SetAPIKey(c.apiKey)
	}

	c.connected = true
	c.logger.Info("TRON gRPC 连接已建立",
		zap.String("endpoint", c.grpcEndpoint),
		zap.Duration("timeout", c.grpcTimeout),
		zap.Duration("batch_timeout", c.grpcBatchTimeout),
	)
	return nil
}

// Close 关闭 gRPC 连接并发送停止信号中断重连退避
func (c *GrpcBlockClient) Close() {
	// 发送停止信号，中断可能正在进行的重连退避等待
	c.stopOnce.Do(func() { close(c.stopCh) })

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Stop()
		c.conn = nil
	}
	c.connected = false
}

// GetCurrentBlockNumber 获取当前最新区块高度
func (c *GrpcBlockClient) GetCurrentBlockNumber(ctx context.Context) (int64, error) {
	if err := c.ensureConnected(); err != nil {
		return 0, err
	}

	type result struct {
		num int64
		err error
	}
	ch := make(chan result, 1)

	go func() {
		block, err := c.conn.GetNowBlock()
		if err != nil {
			ch <- result{0, fmt.Errorf("获取当前区块失败: %w", err)}
			return
		}
		if block == nil || block.BlockHeader == nil || block.BlockHeader.RawData == nil {
			ch <- result{0, fmt.Errorf("获取当前区块返回空数据")}
			return
		}
		ch <- result{block.BlockHeader.RawData.Number, nil}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, c.grpcTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		c.markDisconnected()
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return 0, fmt.Errorf("获取当前区块超时 (%v)", c.grpcTimeout)
	case r := <-ch:
		if r.err != nil {
			c.markDisconnected()
		}
		return r.num, r.err
	}
}

// GetBlockByNum 获取指定区块号的完整区块数据
func (c *GrpcBlockClient) GetBlockByNum(ctx context.Context, num int64) (*port.BlockData, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	type result struct {
		block *api.BlockExtention
		err   error
	}
	ch := make(chan result, 1)

	go func() {
		block, err := c.conn.GetBlockByNum(num)
		ch <- result{block, err}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, c.grpcTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		c.markDisconnected()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("获取区块 %d 超时 (%v)", num, c.grpcTimeout)
	case r := <-ch:
		if r.err != nil {
			c.markDisconnected()
			return nil, fmt.Errorf("获取区块 %d 失败: %w", num, r.err)
		}
		return c.convertBlock(r.block), nil
	}
}

// GetBlockRange 批量获取指定范围的区块 [start, end)
// 底层使用 GetBlockByLimitNext 单次 gRPC 调用，比逐块获取快数十倍
func (c *GrpcBlockClient) GetBlockRange(ctx context.Context, start, end int64) ([]*port.BlockData, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	type result struct {
		blockList *api.BlockListExtention
		err       error
	}
	ch := make(chan result, 1)

	go func() {
		blockList, err := c.conn.GetBlockByLimitNext(start, end)
		ch <- result{blockList, err}
	}()

	// 批量操作使用更长的超时
	timeoutCtx, cancel := context.WithTimeout(ctx, c.grpcBatchTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		c.markDisconnected()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("批量获取区块 [%d, %d) 超时 (%v)", start, end, c.grpcBatchTimeout)
	case r := <-ch:
		if r.err != nil {
			c.markDisconnected()
			return nil, fmt.Errorf("批量获取区块 [%d, %d) 失败: %w", start, end, r.err)
		}
		if r.blockList == nil {
			return nil, nil
		}
		blocks := make([]*port.BlockData, 0, len(r.blockList.Block))
		for _, block := range r.blockList.Block {
			blocks = append(blocks, c.convertBlock(block))
		}
		return blocks, nil
	}
}

// GetTransactionReceipt 获取交易回执 (确认数)
func (c *GrpcBlockClient) GetTransactionReceipt(ctx context.Context, txID string) (*port.TxReceipt, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	type result struct {
		receipt *port.TxReceipt
		err     error
	}
	ch := make(chan result, 1)

	go func() {
		txInfo, err := c.conn.GetTransactionInfoByID(txID)
		if err != nil {
			ch <- result{nil, fmt.Errorf("获取交易 %s 回执失败: %w", txID, err)}
			return
		}

		receipt := &port.TxReceipt{}
		if txInfo != nil && txInfo.BlockNumber > 0 {
			// 计算确认数: 当前块高 - 交易块高
			nowBlock, err := c.conn.GetNowBlock()
			if err == nil && nowBlock != nil && nowBlock.BlockHeader != nil && nowBlock.BlockHeader.RawData != nil {
				currentNum := nowBlock.BlockHeader.RawData.Number
				if currentNum > txInfo.BlockNumber {
					receipt.Confirmations = int(currentNum - txInfo.BlockNumber)
				}
			}
		}
		ch <- result{receipt, nil}
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, c.grpcTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		c.markDisconnected()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("获取交易 %s 回执超时 (%v)", txID, c.grpcTimeout)
	case r := <-ch:
		return r.receipt, r.err
	}
}

// --- 内部方法 ---

// ensureConnected 确保 gRPC 连接可用，断线时自动重连
func (c *GrpcBlockClient) ensureConnected() error {
	c.mu.RLock()
	if c.connected {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()
	return c.reconnectWithBackoff()
}

// reconnectWithBackoff 指数退避重连 (响应 context 取消，避免关闭时阻塞)
func (c *GrpcBlockClient) reconnectWithBackoff() error {
	delay := 1 * time.Second
	maxDelay := 30 * time.Second

	for attempt := 0; attempt < 5; attempt++ {
		if err := c.Connect(); err == nil {
			return nil
		}
		c.logger.Warn("gRPC 重连失败，等待后重试",
			zap.Int("attempt", attempt+1),
			zap.Duration("delay", delay),
		)
		timer := time.NewTimer(delay)
		select {
		case <-c.stopCh:
			timer.Stop()
			return fmt.Errorf("gRPC 重连中断: 收到停止信号")
		case <-timer.C:
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	return fmt.Errorf("gRPC 重连失败，已达最大重试次数 (endpoint: %s)", c.grpcEndpoint)
}

// markDisconnected 标记连接已断开
func (c *GrpcBlockClient) markDisconnected() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
}

// convertBlock 将 protobuf BlockExtention 转换为 port.BlockData
func (c *GrpcBlockClient) convertBlock(block *api.BlockExtention) *port.BlockData {
	if block == nil || block.BlockHeader == nil || block.BlockHeader.RawData == nil {
		return &port.BlockData{}
	}

	bd := &port.BlockData{
		Number: block.BlockHeader.RawData.Number,
	}

	for _, txExt := range block.Transactions {
		if txExt == nil || txExt.Transaction == nil {
			continue
		}

		txData := port.BlockTxData{
			TxID:    hex.EncodeToString(txExt.Txid),
			Success: c.isTxSuccess(txExt.Transaction),
		}

		if txExt.Transaction.RawData != nil {
			for _, contract := range txExt.Transaction.RawData.Contract {
				if cd := c.convertContract(contract); cd != nil {
					txData.Contracts = append(txData.Contracts, *cd)
				}
			}
		}

		bd.Transactions = append(bd.Transactions, txData)
	}

	return bd
}

// isTxSuccess 检查 protobuf 交易是否执行成功
func (c *GrpcBlockClient) isTxSuccess(tx *core.Transaction) bool {
	if tx == nil || len(tx.Ret) == 0 {
		return false
	}
	return tx.Ret[0].ContractRet == core.Transaction_Result_SUCCESS
}

// convertContract 将 protobuf Contract 转换为 port.TxContractData
func (c *GrpcBlockClient) convertContract(contract *core.Transaction_Contract) *port.TxContractData {
	if contract == nil || contract.Parameter == nil {
		return nil
	}

	switch contract.Type {
	case core.Transaction_Contract_TransferContract:
		var tc core.TransferContract
		if err := proto.Unmarshal(contract.Parameter.Value, &tc); err != nil {
			c.logger.Debug("解析 TransferContract 失败", zap.Error(err))
			return nil
		}
		return &port.TxContractData{
			Type:         "TransferContract",
			OwnerAddress: address.Address(tc.OwnerAddress).String(),
			ToAddress:    address.Address(tc.ToAddress).String(),
			Amount:       tc.Amount,
		}

	case core.Transaction_Contract_TriggerSmartContract:
		var tsc core.TriggerSmartContract
		if err := proto.Unmarshal(contract.Parameter.Value, &tsc); err != nil {
			c.logger.Debug("解析 TriggerSmartContract 失败", zap.Error(err))
			return nil
		}
		return &port.TxContractData{
			Type:            "TriggerSmartContract",
			OwnerAddress:    address.Address(tsc.OwnerAddress).String(),
			ContractAddress: address.Address(tsc.ContractAddress).String(),
			Data:            hex.EncodeToString(tsc.Data),
		}

	default:
		return nil // 忽略其他合约类型
	}
}
