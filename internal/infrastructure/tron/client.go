// Package tron 提供 TRON 区块链 API 客户端
// HTTP API 用于提币发送，gRPC 用于充值监控区块扫描
package tron

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// USDT TRC20 合约地址 (TRON 主网)
const USDTContractAddress = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

// defaultFeeLimit 默认手续费上限 (100 TRX = 100_000_000 sun)
const defaultFeeLimit = 100_000_000

// Client TRON 区块链 HTTP API 客户端 (提币发送使用)
type Client struct {
	apiURL     string
	apiKey     string
	feeLimit   int64 // TRC20 交易手续费上限 (sun)
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient 创建 TRON HTTP API 客户端
// apiURL: TronGrid HTTP API 地址
// apiKey: TronGrid API Key
// feeLimit: TRC20 交易手续费上限 (sun)，传 0 使用默认值 100 TRX
func NewClient(apiURL, apiKey string, feeLimit int64, logger *zap.Logger) *Client {
	if apiURL == "" {
		apiURL = "https://api.trongrid.io"
	}

	if feeLimit <= 0 {
		feeLimit = defaultFeeLimit
	}

	return &Client{
		apiURL:   apiURL,
		apiKey:   apiKey,
		feeLimit: feeLimit,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}
