// Package exchange 提供汇率数据获取和自动更新功能
package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// CoinGeckoClient CoinGecko 免费 API 客户端
type CoinGeckoClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
}

// PriceResult 价格查询结果
type PriceResult struct {
	TRON   map[string]float64 // {"usd": 0.12, "cny": 0.85}
	Tether map[string]float64 // {"usd": 1.0, "cny": 7.1}
}

// coinGeckoResponse CoinGecko API 原始响应结构
type coinGeckoResponse struct {
	Tron   map[string]float64 `json:"tron"`
	Tether map[string]float64 `json:"tether"`
}

// NewCoinGeckoClient 创建 CoinGecko API 客户端
func NewCoinGeckoClient(logger *zap.Logger) *CoinGeckoClient {
	return &CoinGeckoClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://api.coingecko.com/api/v3",
		logger:  logger,
	}
}

// GetPrices 获取 TRON 和 USDT 的 USD/CNY 价格
// 使用指数退避重试策略，最多重试 3 次
func (c *CoinGeckoClient) GetPrices(ctx context.Context) (*PriceResult, error) {
	url := c.baseURL + "/simple/price?ids=tron,tether&vs_currencies=usd,cny"

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// 指数退避: 1s, 2s, 4s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			c.logger.Warn("CoinGecko API 请求重试",
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", backoff),
				zap.Error(lastErr),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, err := c.doRequest(ctx, url)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("CoinGecko API 请求失败 (已重试 3 次): %w", lastErr)
}

// doRequest 执行单次 HTTP 请求
func (c *CoinGeckoClient) doRequest(ctx context.Context, url string) (*PriceResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "WalletBot/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 CoinGecko API 失败: %w", err)
	}
	defer resp.Body.Close()

	// 处理 429 Too Many Requests
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("CoinGecko API 请求频率超限 (429)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CoinGecko API 返回异常状态码 %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var apiResp coinGeckoResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 CoinGecko 响应 JSON 失败: %w", err)
	}

	// 校验响应数据完整性
	if err := validatePriceData(apiResp); err != nil {
		return nil, err
	}

	return &PriceResult{
		TRON:   apiResp.Tron,
		Tether: apiResp.Tether,
	}, nil
}

// validatePriceData 校验 API 返回的价格数据完整性
func validatePriceData(resp coinGeckoResponse) error {
	if resp.Tron == nil {
		return fmt.Errorf("CoinGecko 响应缺少 tron 数据")
	}
	if resp.Tether == nil {
		return fmt.Errorf("CoinGecko 响应缺少 tether 数据")
	}
	if _, ok := resp.Tron["usd"]; !ok {
		return fmt.Errorf("CoinGecko 响应缺少 tron.usd 数据")
	}
	if _, ok := resp.Tron["cny"]; !ok {
		return fmt.Errorf("CoinGecko 响应缺少 tron.cny 数据")
	}
	if _, ok := resp.Tether["usd"]; !ok {
		return fmt.Errorf("CoinGecko 响应缺少 tether.usd 数据")
	}
	if _, ok := resp.Tether["cny"]; !ok {
		return fmt.Errorf("CoinGecko 响应缺少 tether.cny 数据")
	}

	// 价格必须为正数
	if resp.Tron["usd"] <= 0 || resp.Tron["cny"] <= 0 {
		return fmt.Errorf("CoinGecko 返回的 TRON 价格无效: usd=%.8f, cny=%.8f", resp.Tron["usd"], resp.Tron["cny"])
	}
	if resp.Tether["usd"] <= 0 || resp.Tether["cny"] <= 0 {
		return fmt.Errorf("CoinGecko 返回的 Tether 价格无效: usd=%.8f, cny=%.8f", resp.Tether["usd"], resp.Tether["cny"])
	}

	return nil
}
