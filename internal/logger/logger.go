package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger 初始化日志系统
// workspace: 工作目录，日志将存放在 workspace/logs 目录
// debug: 是否启用debug级别日志
func InitLogger(workspace string, debug bool) (*zap.Logger, error) {
	// 确保日志目录存在
	logDir := filepath.Join(workspace, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFilePath := filepath.Join(logDir, "myclaw.log")

	// 配置日志级别
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	// 配置编码器
	encoderConfig := zap.NewProductionEncoderConfig()
	// 控制台输出：简洁格式（只显示时:分:秒）
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// 短路径格式：文件名:行号（而不是完整路径）
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 文件输出配置
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// 如果文件打开失败，只使用控制台输出
		fmt.Fprintf(os.Stderr, "Warning: failed to open log file: %v, using console only\n", err)
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
		return zap.New(core, zap.AddCaller()), nil
	}

	// 控制台输出配置
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 创建多重输出core
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(logFile), level),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

