// Package server 提供 HTTP 服务器初始化
package server

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/OwenALL/TG_walletbot/internal/infrastructure/config"
	"go.uber.org/zap"
)

// NewGin 创建并配置 Gin 引擎
func NewGin(cfg *config.Config, logger *zap.Logger) *gin.Engine {
	// 根据环境设置 Gin 模式
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// 使用自定义的日志和恢复中间件 (后续由 middleware 包提供)
	engine.Use(gin.Recovery())

	return engine
}

// Run 启动 HTTP 服务器
func Run(engine *gin.Engine, cfg *config.ServerConfig, logger *zap.Logger) error {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	logger.Info("HTTP 服务器启动",
		zap.String("addr", addr),
		zap.Int("read_timeout", cfg.ReadTimeout),
		zap.Int("write_timeout", cfg.WriteTimeout),
	)

	_ = time.Duration(cfg.ReadTimeout) * time.Second
	_ = time.Duration(cfg.WriteTimeout) * time.Second

	return engine.Run(addr)
}
