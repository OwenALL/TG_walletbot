// Package port 定义领域层端口接口
package port

import (
	"context"

	"github.com/shopspring/decimal"
)

// AddressGenerator 区块链地址生成器端口
type AddressGenerator interface {
	// GenerateAddress 生成新的区块链地址和对应的私钥
	// 返回: 地址, 私钥 (hex 编码), 错误
	GenerateAddress() (address string, privateKeyHex string, err error)
}

// PrivateKeyCrypto 私钥加密/解密端口
type PrivateKeyCrypto interface {
	// EncryptPrivateKey 加密私钥
	// 返回: base64 编码的密文
	EncryptPrivateKey(privateKeyHex string, encryptionKey string) (string, error)
	// DecryptPrivateKey 解密私钥
	// 返回: 原始私钥 hex 字符串
	DecryptPrivateKey(encrypted string, encryptionKey string) (string, error)
}

// --- 区块扫描端口 (充值监控使用) ---

// BlockData 区块数据 (适配层统一格式，隔离 protobuf 细节)
type BlockData struct {
	Number       int64
	Transactions []BlockTxData
}

// BlockTxData 区块内单笔交易数据
type BlockTxData struct {
	TxID      string
	Success   bool
	Contracts []TxContractData
}

// TxContractData 交易内合约调用数据
type TxContractData struct {
	Type            string // "TransferContract" / "TriggerSmartContract"
	OwnerAddress    string // 发起方地址 (Base58)
	ToAddress       string // 接收方地址 (Base58, 仅 TransferContract)
	Amount          int64  // 金额 sun (仅 TransferContract)
	ContractAddress string // 合约地址 (Base58, 仅 TriggerSmartContract)
	Data            string // 合约调用数据 hex (仅 TriggerSmartContract)
}

// TxReceipt 交易回执信息
type TxReceipt struct {
	Confirmations int
}

// BlockScanner 区块链区块扫描端口
// 用于充值监控的区块数据读取，支持批量拉取以加速追赶
type BlockScanner interface {
	// Connect 建立连接
	Connect() error
	// Close 关闭连接释放资源
	Close()
	// GetCurrentBlockNumber 获取当前最新区块高度
	GetCurrentBlockNumber(ctx context.Context) (int64, error)
	// GetBlockByNum 获取指定区块号的完整区块数据
	GetBlockByNum(ctx context.Context, num int64) (*BlockData, error)
	// GetBlockRange 批量获取指定范围的区块 [start, end)
	// 用于追赶落后区块，比逐块获取效率高数十倍
	GetBlockRange(ctx context.Context, start, end int64) ([]*BlockData, error)
	// GetTransactionReceipt 获取交易回执 (确认数)
	GetTransactionReceipt(ctx context.Context, txID string) (*TxReceipt, error)
}

// WithdrawalSender 提币发送器端口
type WithdrawalSender interface {
	// SendWithdrawal 执行提币发送
	// 返回: (txHash, isManual, error)
	// isManual=true 表示金额超过阈值需人工处理
	SendWithdrawal(ctx context.Context, toAddress, currency string, amount decimal.Decimal) (txHash string, isManual bool, err error)
}
