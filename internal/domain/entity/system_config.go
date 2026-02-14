package entity

import "time"

// SystemConfig 系统配置实体
type SystemConfig struct {
	ID          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	ConfigKey   string    `json:"config_key" gorm:"size:64;not null;uniqueIndex"`
	ConfigValue string    `json:"config_value" gorm:"type:text;not null"`
	Description string    `json:"description" gorm:"size:255;default:''"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "system_configs"
}

// 预定义系统配置 key 常量
const (
	ConfigWithdrawUSDTMin           = "withdraw_usdt_min"
	ConfigWithdrawUSDTMax           = "withdraw_usdt_max"
	ConfigWithdrawUSDTDailyMax      = "withdraw_usdt_daily_max"
	ConfigWithdrawUSDTFee           = "withdraw_usdt_fee"
	ConfigWithdrawUSDTAutoThreshold = "withdraw_usdt_auto_threshold"
	ConfigWithdrawTRXMin            = "withdraw_trx_min"
	ConfigWithdrawTRXMax            = "withdraw_trx_max"
	ConfigWithdrawTRXFee            = "withdraw_trx_fee"
	ConfigWithdrawCNYMin            = "withdraw_cny_min"
	ConfigWithdrawCNYDailyMax       = "withdraw_cny_daily_max"
	ConfigTransferUSDTMin           = "transfer_usdt_min"
	ConfigExchangeMinUSDT           = "exchange_min_usdt"
	ConfigExchangeMinCNY            = "exchange_min_cny"
	ConfigExchangeMinTRX            = "exchange_min_trx"
	ConfigFinanceMin                = "finance_min"
	ConfigFinanceMax                = "finance_max"
	ConfigFinanceAnnualRate         = "finance_annual_rate"
	ConfigPINMaxFail                = "pin_max_fail"
	ConfigPINLockMinutes            = "pin_lock_minutes"
	ConfigRedpacketExpireHours      = "redpacket_expire_hours"
	ConfigDepositConfirmations      = "deposit_confirmations"
	ConfigAdminTelegramID           = "admin_telegram_id"
	ConfigSupportURL                = "support_url"
	ConfigTradingGroupURL           = "trading_group_url"
	ConfigGameCenterURL             = "game_center_url"
	ConfigMaintenanceMode           = "maintenance_mode"

	// TRON 链上运营参数 (从 config.yaml 迁移至 DB，支持后台动态调整)
	ConfigTronFeeLimit         = "tron_fee_limit"
	ConfigTronUSDTContract     = "tron_usdt_contract"
	ConfigTronGRPCTimeout      = "tron_grpc_timeout"
	ConfigTronGRPCBatchTimeout = "tron_grpc_batch_timeout"
)
