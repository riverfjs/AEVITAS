package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
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

	// 使用 lumberjack 实现日志轮转
	// 按天分割（每天 00:00 自动轮转）+ 按大小轮转（单文件最大 100MB）
	logRotator := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100,   // MB，单文件最大大小
		MaxBackups: 30,    // 保留最近 30 个备份文件
		MaxAge:     30,    // 保留最近 30 天的日志
		LocalTime:  true,  // 使用本地时间
		Compress:   false, // 不压缩（方便直接查看）
	}

	// 文件输出配置
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	
	// 检查是否为后台运行模式（通过环境变量）
	isDaemon := os.Getenv("MYCLAW_DAEMON") == "1"
	
	var core zapcore.Core
	if isDaemon {
		// 后台模式：只输出到文件，避免与 nohup 重复
		core = zapcore.NewCore(fileEncoder, zapcore.AddSync(logRotator), level)
	} else {
		// 前台模式：同时输出到文件和控制台
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(logRotator), level),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		)
	}

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

