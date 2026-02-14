package tron

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/TGlimmer/TG_walletbot/internal/app"
	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/domain/port"
)

const (
	// Redis 键: 最后处理的区块号
	cacheKeyLastBlock = "tron:monitor:last_block"
	// 默认轮询间隔 (TRON 出块约 3 秒)
	defaultBlockPollInterval = 3 * time.Second
	// 每次最多处理的区块数 (防止长时间未运行后一次处理太多)
	maxBlocksPerPoll = 100
	// 落后超过此数时启用批量拉取
	batchThreshold = 5
	// 地址缓存刷新间隔
	addressCacheRefreshInterval = 5 * time.Minute
	// TRC20 transfer(address,uint256) 函数选择器
	transferSelector = "a9059cbb"
)

// DepositMonitor TRON 链上充值监控服务
// 采用 gRPC 区块扫描模式: 通过 BlockScanner 接口读取区块数据
// 稳态逐块扫描，追赶时批量拉取 (GetBlockRange)
type DepositMonitor struct {
	scanner    port.BlockScanner
	depositUC  *app.DepositUseCase
	walletRepo port.WalletRepository
	userRepo   port.UserRepository
	notifySvc  port.NotificationService
	cacheRepo  port.CacheRepository
	logger     *zap.Logger

	pollInterval time.Duration // 轮询间隔
	usdtContract string        // USDT TRC20 合约地址 (Base58)

	// 地址缓存: address(string) -> userID(uint64)
	// 使用 RWMutex 替代 sync.Map，支持增量更新
	addrMu        sync.RWMutex
	addrCache     map[string]uint64
	lastRefreshAt time.Time // 上次刷新时间，用于增量查询
}

// NewDepositMonitor 创建充值监控服务
func NewDepositMonitor(
	scanner port.BlockScanner,
	depositUC *app.DepositUseCase,
	walletRepo port.WalletRepository,
	userRepo port.UserRepository,
	notifySvc port.NotificationService,
	cacheRepo port.CacheRepository,
	usdtContract string,
	logger *zap.Logger,
) *DepositMonitor {
	if usdtContract == "" {
		usdtContract = USDTContractAddress
	}
	return &DepositMonitor{
		scanner:      scanner,
		depositUC:    depositUC,
		walletRepo:   walletRepo,
		userRepo:     userRepo,
		notifySvc:    notifySvc,
		cacheRepo:    cacheRepo,
		logger:       logger,
		pollInterval: defaultBlockPollInterval,
		usdtContract: usdtContract,
		addrCache:    make(map[string]uint64),
	}
}

// Start 启动充值监控 (阻塞运行，直到 ctx 取消)
func (m *DepositMonitor) Start(ctx context.Context) {
	m.logger.Info("TRON 充值监控服务启动 (gRPC 区块扫描模式)",
		zap.Duration("poll_interval", m.pollInterval),
		zap.String("usdt_contract", m.usdtContract),
	)

	// 建立 gRPC 连接
	if err := m.scanner.Connect(); err != nil {
		m.logger.Error("gRPC 连接失败，将在首次轮询时重试", zap.Error(err))
	}
	defer m.scanner.Close()

	// 首次全量加载充值地址缓存
	m.fullRefreshAddressCache(ctx)

	// 获取起始区块号
	lastBlock := m.getLastProcessedBlock(ctx)
	if lastBlock == 0 {
		// 首次启动: 从当前区块开始 (不回扫历史)
		currentBlock, err := m.scanner.GetCurrentBlockNumber(ctx)
		if err != nil {
			m.logger.Error("获取当前区块号失败，将在下次轮询重试", zap.Error(err))
		} else {
			lastBlock = currentBlock
			m.saveLastProcessedBlock(ctx, lastBlock)
			m.logger.Info("首次启动，从当前区块开始监控", zap.Int64("block", lastBlock))
		}
	} else {
		m.logger.Info("从上次断点继续监控", zap.Int64("last_block", lastBlock))
	}

	blockTicker := time.NewTicker(m.pollInterval)
	defer blockTicker.Stop()

	cacheTicker := time.NewTicker(addressCacheRefreshInterval)
	defer cacheTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("TRON 充值监控服务停止")
			return
		case <-blockTicker.C:
			m.scanNewBlocks(ctx, &lastBlock)
		case <-cacheTicker.C:
			m.incrementalRefreshAddressCache(ctx)
		}
	}
}

// scanNewBlocks 扫描新产生的区块
// 落后 <= batchThreshold 块时逐块扫描，落后更多时批量拉取
func (m *DepositMonitor) scanNewBlocks(ctx context.Context, lastBlock *int64) {
	currentBlock, err := m.scanner.GetCurrentBlockNumber(ctx)
	if err != nil {
		m.logger.Error("获取当前区块号失败", zap.Error(err))
		return
	}

	if currentBlock <= *lastBlock {
		return // 没有新区块
	}

	startBlock := *lastBlock + 1
	behind := currentBlock - startBlock + 1

	// 限制单次处理区块数
	if behind > maxBlocksPerPoll {
		startBlock = currentBlock - maxBlocksPerPoll + 1
		m.logger.Warn("积压区块过多，跳过部分区块",
			zap.Int64("skipped_from", *lastBlock+1),
			zap.Int64("start_from", startBlock),
			zap.Int64("current", currentBlock),
		)
		behind = maxBlocksPerPoll
	}

	// 根据落后程度选择扫描策略
	if behind > batchThreshold {
		m.scanBlocksBatch(ctx, lastBlock, startBlock, currentBlock)
	} else {
		m.scanBlocksSequential(ctx, lastBlock, startBlock, currentBlock)
	}
}

// scanBlocksBatch 批量模式: 使用 GetBlockRange 一次获取多块
// 追赶场景下比逐块模式快 20-50 倍
func (m *DepositMonitor) scanBlocksBatch(ctx context.Context, lastBlock *int64, start, end int64) {
	m.logger.Info("进入批量追赶模式",
		zap.Int64("start", start),
		zap.Int64("end", end),
		zap.Int64("count", end-start+1),
	)

	// GetBlockRange 参数: [start, end)，所以 end 要 +1
	blocks, err := m.scanner.GetBlockRange(ctx, start, end+1)
	if err != nil {
		m.logger.Warn("批量获取区块失败，降级为逐块模式",
			zap.Int64("start", start),
			zap.Int64("end", end),
			zap.Error(err),
		)
		m.scanBlocksSequential(ctx, lastBlock, start, end)
		return
	}

	processedCount := 0
	depositCount := 0
	var latestBlock int64

	for _, block := range blocks {
		select {
		case <-ctx.Done():
			// 中断前保存已处理的最新进度
			if latestBlock > 0 {
				*lastBlock = latestBlock
				m.saveLastProcessedBlock(ctx, latestBlock)
			}
			return
		default:
		}

		if block == nil || block.Number == 0 {
			continue
		}

		deposits := m.processBlockData(ctx, block)
		processedCount++
		depositCount += deposits
		latestBlock = block.Number
	}

	// 批量处理完成后统一保存进度 (避免逐块写 Redis)
	if latestBlock > 0 {
		*lastBlock = latestBlock
		m.saveLastProcessedBlock(ctx, latestBlock)
	}

	if processedCount > 0 {
		m.logger.Info("批量区块扫描完成",
			zap.Int("blocks", processedCount),
			zap.Int("deposits", depositCount),
			zap.Int64("latest", *lastBlock),
		)
	}
}

// scanBlocksSequential 逐块模式: 稳态时使用，每块单独获取
func (m *DepositMonitor) scanBlocksSequential(ctx context.Context, lastBlock *int64, start, end int64) {
	processedCount := 0
	depositCount := 0

	for blockNum := start; blockNum <= end; blockNum++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		block, err := m.scanner.GetBlockByNum(ctx, blockNum)
		if err != nil {
			m.logger.Error("获取区块失败",
				zap.Int64("block", blockNum),
				zap.Error(err),
			)
			break // 停在失败的区块，下次重试
		}

		deposits := m.processBlockData(ctx, block)
		processedCount++
		depositCount += deposits

		*lastBlock = blockNum
		m.saveLastProcessedBlock(ctx, blockNum)
	}

	if processedCount > 0 {
		m.logger.Debug("区块扫描完成",
			zap.Int("blocks", processedCount),
			zap.Int("deposits", depositCount),
			zap.Int64("latest", *lastBlock),
		)
	}
}

// processBlockData 处理单个区块的所有交易
func (m *DepositMonitor) processBlockData(ctx context.Context, block *port.BlockData) int {
	if block == nil || block.Number == 0 {
		return 0
	}

	depositCount := 0
	for _, tx := range block.Transactions {
		if !tx.Success {
			continue
		}

		for _, contract := range tx.Contracts {
			switch contract.Type {
			case "TransferContract":
				if m.processTRXFromData(ctx, tx.TxID, contract) {
					depositCount++
				}
			case "TriggerSmartContract":
				if m.processTRC20FromData(ctx, tx.TxID, contract) {
					depositCount++
				}
			}
		}
	}
	return depositCount
}

// processTRXFromData 处理 TRX 原生转账
func (m *DepositMonitor) processTRXFromData(ctx context.Context, txID string, c port.TxContractData) bool {
	// 检查收款地址是否在监控列表中
	if !m.isMonitoredAddress(c.ToAddress) {
		return false
	}

	// 转换金额: sun → TRX (1 TRX = 1,000,000 sun)
	amount := decimal.NewFromInt(c.Amount).Shift(-6)
	if amount.IsZero() || amount.IsNegative() {
		return false
	}

	m.handleDeposit(ctx, txID, c.OwnerAddress, c.ToAddress, entity.CurrencyTRX, amount)
	return true
}

// processTRC20FromData 处理 TRC20 代币转账 (USDT)
func (m *DepositMonitor) processTRC20FromData(ctx context.Context, txID string, c port.TxContractData) bool {
	// 只处理 USDT 合约
	if c.ContractAddress != m.usdtContract {
		return false
	}

	// 检查是否为 transfer(address,uint256) 调用
	data := c.Data
	if !strings.HasPrefix(data, transferSelector) {
		return false
	}
	// data 格式: a9059cbb (8) + address_padded (64) + amount (64) = 136 chars
	if len(data) < 136 {
		return false
	}

	// 解析收款地址: data[32:72] 为 20 字节地址 hex (无 0x41 前缀)
	toAddrHex := data[32:72]
	toAddress, err := hexAddrToBase58(toAddrHex)
	if err != nil {
		m.logger.Debug("解析 TRC20 收款地址失败",
			zap.String("hex", toAddrHex),
			zap.Error(err),
		)
		return false
	}

	// 检查收款地址是否在监控列表中
	if !m.isMonitoredAddress(toAddress) {
		return false
	}

	// 解析金额: data[72:136] 为 uint256 金额
	amountHex := data[72:136]
	amountBig, ok := new(big.Int).SetString(amountHex, 16)
	if !ok || amountBig.Sign() <= 0 {
		return false
	}
	// USDT TRC20 精度 6 位
	amount := decimal.NewFromBigInt(amountBig, -6)

	m.handleDeposit(ctx, txID, c.OwnerAddress, toAddress, entity.CurrencyUSDT, amount)
	return true
}

// handleDeposit 处理发现的充值交易
func (m *DepositMonitor) handleDeposit(ctx context.Context, txHash, from, to, currency string, amount decimal.Decimal) {
	m.logger.Info("发现链上充值",
		zap.String("tx_hash", txHash),
		zap.String("from", from),
		zap.String("to", to),
		zap.String("currency", currency),
		zap.String("amount", amount.String()),
	)

	// 获取确认数 (区块扫描的交易通常已有足够确认)
	confirmations := 3
	receipt, err := m.scanner.GetTransactionReceipt(ctx, txHash)
	if err == nil && receipt != nil && receipt.Confirmations > 0 {
		confirmations = receipt.Confirmations
	}

	// 调用用例层处理充值 (内部已有 txHash 去重)
	notification, err := m.depositUC.ProcessChainDeposit(ctx, txHash, from, to, currency, amount, confirmations)
	if err != nil {
		m.logger.Error("处理链上充值失败",
			zap.String("tx_hash", txHash),
			zap.String("currency", currency),
			zap.String("amount", amount.String()),
			zap.Error(err),
		)
		return
	}

	if notification != nil {
		m.sendNotification(ctx, notification)
	}
}

// sendNotification 发送充值到账通知
func (m *DepositMonitor) sendNotification(ctx context.Context, n *app.DepositNotification) {
	if m.notifySvc == nil {
		return
	}

	user, err := m.userRepo.FindByID(ctx, n.UserID)
	if err != nil || user == nil {
		m.logger.Error("发送充值通知失败: 查找用户失败",
			zap.Uint64("user_id", n.UserID),
			zap.Error(err),
		)
		return
	}

	m.notifySvc.SendDepositNotification(
		ctx,
		user.TelegramID,
		n.Currency,
		n.Amount.String(),
		n.TxHash,
		n.NewBalance.String(),
	)

	m.logger.Info("充值到账通知已发送",
		zap.Uint64("user_id", n.UserID),
		zap.Int64("telegram_id", user.TelegramID),
		zap.String("currency", n.Currency),
		zap.String("amount", n.Amount.String()),
	)
}

// --- 区块进度持久化 (Redis) ---

// getLastProcessedBlock 从 Redis 获取上次处理的区块号
func (m *DepositMonitor) getLastProcessedBlock(ctx context.Context) int64 {
	val, err := m.cacheRepo.Get(ctx, cacheKeyLastBlock)
	if err != nil || val == "" {
		return 0
	}
	blockNum, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0
	}
	return blockNum
}

// saveLastProcessedBlock 将当前处理进度保存到 Redis
func (m *DepositMonitor) saveLastProcessedBlock(ctx context.Context, blockNum int64) {
	if err := m.cacheRepo.Set(ctx, cacheKeyLastBlock, strconv.FormatInt(blockNum, 10), 0); err != nil {
		m.logger.Error("保存区块进度失败", zap.Int64("block", blockNum), zap.Error(err))
	}
}

// --- 地址缓存管理 ---

// isMonitoredAddress 检查地址是否在监控列表中 (读操作，使用 RLock)
func (m *DepositMonitor) isMonitoredAddress(addr string) bool {
	m.addrMu.RLock()
	_, ok := m.addrCache[addr]
	m.addrMu.RUnlock()
	return ok
}

// fullRefreshAddressCache 全量加载所有充值地址到内存缓存
// 用于首次启动或增量刷新异常时的降级方案
func (m *DepositMonitor) fullRefreshAddressCache(ctx context.Context) {
	wallets, err := m.walletRepo.ListAllWithDepositAddress(ctx)
	if err != nil {
		m.logger.Error("全量加载充值地址缓存失败", zap.Error(err))
		return
	}

	newCache := make(map[string]uint64, len(wallets))
	for _, w := range wallets {
		if w.DepositAddress != "" {
			newCache[w.DepositAddress] = w.UserID
		}
	}

	m.addrMu.Lock()
	m.addrCache = newCache
	m.lastRefreshAt = time.Now()
	m.addrMu.Unlock()

	m.logger.Info("全量刷新充值地址缓存", zap.Int("count", len(newCache)))
}

// incrementalRefreshAddressCache 增量更新地址缓存
// 只查询上次刷新之后新创建的充值地址，减少数据库查询开销
// 增量查询异常时自动降级为全量刷新
func (m *DepositMonitor) incrementalRefreshAddressCache(ctx context.Context) {
	m.addrMu.RLock()
	lastRefresh := m.lastRefreshAt
	m.addrMu.RUnlock()

	// 首次刷新或时间为零值时，执行全量加载
	if lastRefresh.IsZero() {
		m.fullRefreshAddressCache(ctx)
		return
	}

	wallets, err := m.walletRepo.ListDepositAddressesCreatedAfter(ctx, lastRefresh)
	if err != nil {
		m.logger.Warn("增量刷新充值地址缓存失败，降级为全量刷新", zap.Error(err))
		m.fullRefreshAddressCache(ctx)
		return
	}

	if len(wallets) == 0 {
		m.logger.Debug("增量刷新: 无新增充值地址")
		return
	}

	m.addrMu.Lock()
	added := 0
	for _, w := range wallets {
		if w.DepositAddress != "" {
			if _, exists := m.addrCache[w.DepositAddress]; !exists {
				m.addrCache[w.DepositAddress] = w.UserID
				added++
			}
		}
	}
	m.lastRefreshAt = time.Now()
	m.addrMu.Unlock()

	if added > 0 {
		m.logger.Info("增量刷新充值地址缓存",
			zap.Int("new_addresses", added),
			zap.Int("total", m.addressCount()),
		)
	}
}

// AddAddress 动态添加监控地址 (新用户注册时调用，避免等待缓存刷新)
func (m *DepositMonitor) AddAddress(address string, userID uint64) {
	if address == "" {
		return
	}
	m.addrMu.Lock()
	m.addrCache[address] = userID
	m.addrMu.Unlock()
	m.logger.Debug("添加充值监控地址",
		zap.String("address", address),
		zap.Uint64("user_id", userID),
	)
}

// addressCount 返回当前缓存的地址数量
func (m *DepositMonitor) addressCount() int {
	m.addrMu.RLock()
	defer m.addrMu.RUnlock()
	return len(m.addrCache)
}

// SetPollInterval 设置轮询间隔
func (m *DepositMonitor) SetPollInterval(d time.Duration) {
	m.pollInterval = d
}

// --- 工具函数 ---

// hexAddrToBase58 将 20 字节 hex 地址转为 TRON Base58Check 地址
// hexAddr: 40 字符 hex 字符串 (不含 0x41 前缀)
func hexAddrToBase58(hexAddr string) (string, error) {
	// 去掉可能的前导零
	hexAddr = strings.TrimLeft(hexAddr, "0")
	// 补齐到 40 字符 (20 字节)
	if len(hexAddr) < 40 {
		hexAddr = fmt.Sprintf("%040s", hexAddr)
	}

	addrBytes, err := hex.DecodeString(hexAddr)
	if err != nil {
		return "", fmt.Errorf("hex 解码失败: %w", err)
	}
	if len(addrBytes) != 20 {
		return "", fmt.Errorf("地址长度无效: %d 字节", len(addrBytes))
	}

	// 添加 0x41 前缀 (TRON 主网)
	payload := make([]byte, 21)
	payload[0] = 0x41
	copy(payload[1:], addrBytes)

	return base58CheckEncode(payload), nil
}
