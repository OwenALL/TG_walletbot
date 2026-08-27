// Package database 提供数据库连接管理
package database

import (
	"fmt"
	"time"

	"github.com/OwenALL/TG_walletbot/internal/infrastructure/config"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewMySQL 创建 MySQL 数据库连接
func NewMySQL(cfg *config.DatabaseConfig, logger *zap.Logger) (*gorm.DB, error) {
	// 根据环境设置 GORM 日志级别
	logLevel := gormlogger.Silent
	if cfg.Charset == "" {
		cfg.Charset = "utf8mb4"
	}

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	// 获取底层 sql.DB 进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 sql.DB 失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL 连接测试失败: %w", err)
	}

	logger.Info("MySQL 连接成功",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	return db, nil
}
