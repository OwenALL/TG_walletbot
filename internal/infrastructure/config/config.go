// Package config 提供应用配置加载功能
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const (
	// jwtSecretMinLen JWT Secret 最小长度要求
	jwtSecretMinLen = 32
)

// commonWeakSecrets 常见弱密钥列表 (小写比较)
var commonWeakSecrets = []string{
	"secret",
	"jwt-secret",
	"jwt_secret",
	"changeme",
	"change_me",
	"change_me_in_production",
	"password",
	"123456",
	"your-secret-key",
	"my-secret-key",
	"test",
	"default",
}

// Config 应用全局配置
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Bot      BotConfig      `mapstructure:"bot"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Tron     TronConfig     `mapstructure:"tron"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name  string `mapstructure:"name"`
	Env   string `mapstructure:"env"`
	Debug bool   `mapstructure:"debug"`
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

// BotConfig Telegram Bot 配置
type BotConfig struct {
	Token      string `mapstructure:"token"`
	WebhookURL string `mapstructure:"webhook_url"`
	Mode       string `mapstructure:"mode"` // polling / webhook
}

// DatabaseConfig MySQL 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxLifetime  int    `mapstructure:"max_lifetime"`
}

// DSN 生成 MySQL 连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name, d.Charset)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

// TronConfig TRON 区块链配置
// 运营参数 (fee_limit, usdt_contract, grpc_timeout, grpc_batch_timeout, auto_send_threshold)
// 已迁移至 system_configs 数据库表，通过管理后台动态调整，此处仅保留基础设施连接信息
type TronConfig struct {
	APIURL                string `mapstructure:"api_url"`
	GRPCEndpoint          string `mapstructure:"grpc_endpoint"`
	APIKey                string `mapstructure:"api_key"`
	EncryptionKey         string `mapstructure:"encryption_key"`
	HotWalletAddress      string `mapstructure:"hot_wallet_address"`
	HotWalletEncryptedKey string `mapstructure:"hot_wallet_encrypted_key"`
	Enabled               bool   `mapstructure:"enabled"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	Output   string `mapstructure:"output"`
	FilePath string `mapstructure:"file_path"`
}

// Load 加载配置文件
// 配置优先级: 环境变量 > 配置文件 > 默认值
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 配置文件
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// 环境变量覆盖
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定环境变量到配置键
	bindEnvVars(v)

	// 读取配置文件 (不存在也不报错，使用环境变量和默认值)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &cfg, nil
}

// ValidateForAPI 验证 API 服务必需的配置项
// 检查 JWT Secret 长度和弱密钥，验证失败返回错误
func (c *Config) ValidateForAPI() error {
	if err := validateJWTSecret(c.JWT.Secret); err != nil {
		return err
	}
	return nil
}

// IsWeakJWTSecret 检查 JWT Secret 是否为常见弱密钥
// 返回 true 表示密钥不安全，应发出警告
func IsWeakJWTSecret(secret string) bool {
	lower := strings.ToLower(secret)
	for _, weak := range commonWeakSecrets {
		if lower == weak {
			return true
		}
	}
	return false
}

// validateJWTSecret 验证 JWT Secret 安全性
func validateJWTSecret(secret string) error {
	if len(secret) < jwtSecretMinLen {
		return fmt.Errorf(
			"JWT Secret 长度不足: 当前 %d 字符，最低要求 %d 字符。请设置环境变量 JWT_SECRET 或修改配置文件 jwt.secret",
			len(secret), jwtSecretMinLen,
		)
	}
	if IsWeakJWTSecret(secret) {
		return fmt.Errorf(
			"JWT Secret 为已知弱密钥 (%q)，请更换为高强度随机字符串。可使用: openssl rand -hex 32",
			secret,
		)
	}
	return nil
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "walletbot")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.debug", true)

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 10)
	v.SetDefault("server.write_timeout", 10)

	v.SetDefault("bot.token", "")
	v.SetDefault("bot.mode", "polling")

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.name", "walletbot")
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_lifetime", 3600)

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("jwt.secret", "change_me_in_production")
	v.SetDefault("jwt.expire_hours", 24)

	v.SetDefault("tron.api_url", "https://api.trongrid.io")
	v.SetDefault("tron.grpc_endpoint", "grpc.trongrid.io:50051")
	v.SetDefault("tron.api_key", "")
	v.SetDefault("tron.encryption_key", "")
	v.SetDefault("tron.hot_wallet_address", "")
	v.SetDefault("tron.hot_wallet_encrypted_key", "")
	v.SetDefault("tron.enabled", false)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}

// bindEnvVars 绑定环境变量到配置键
func bindEnvVars(v *viper.Viper) {
	_ = v.BindEnv("bot.token", "BOT_TOKEN")
	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.name", "DB_NAME")
	_ = v.BindEnv("redis.addr", "REDIS_ADDR")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("jwt.secret", "JWT_SECRET")
	_ = v.BindEnv("server.port", "API_PORT")
	_ = v.BindEnv("app.env", "APP_ENV")
	_ = v.BindEnv("app.debug", "APP_DEBUG")
	_ = v.BindEnv("tron.api_url", "TRON_API_URL")
	_ = v.BindEnv("tron.grpc_endpoint", "TRON_GRPC_ENDPOINT")
	_ = v.BindEnv("tron.api_key", "TRON_API_KEY")
	_ = v.BindEnv("tron.encryption_key", "TRON_ENCRYPTION_KEY")
	_ = v.BindEnv("tron.hot_wallet_address", "TRON_HOT_WALLET_ADDRESS")
	_ = v.BindEnv("tron.hot_wallet_encrypted_key", "TRON_HOT_WALLET_ENCRYPTED_KEY")
	_ = v.BindEnv("tron.enabled", "TRON_ENABLED")
}
