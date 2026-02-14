// Package logger 提供全局 Logger 访问
package logger

import (
	"sync"

	"go.uber.org/zap"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// Init 初始化全局 Logger
func Init(l *zap.Logger) {
	once.Do(func() {
		globalLogger = l
	})
}

// L 获取全局 Logger 实例
func L() *zap.Logger {
	if globalLogger == nil {
		l, _ := zap.NewProduction()
		return l
	}
	return globalLogger
}

// S 获取全局 SugaredLogger 实例
func S() *zap.SugaredLogger {
	return L().Sugar()
}
