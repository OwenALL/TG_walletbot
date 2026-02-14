// Package logger 提供基于 Zap 的结构化日志
package logger

import (
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New 根据配置创建 Zap Logger 实例
func New(cfg *config.LogConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// 设置日志级别
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// 设置输出目标
	if cfg.Output == "file" && cfg.FilePath != "" {
		zapCfg.OutputPaths = []string{cfg.FilePath}
		zapCfg.ErrorOutputPaths = []string{cfg.FilePath}
	}

	// 设置时间格式
	zapCfg.EncoderConfig.TimeKey = "time"
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := zapCfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return nil, err
	}

	return logger, nil
}
