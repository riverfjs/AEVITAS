package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initialises the logging system.
// Logs are written to workspace/logs/ with daily rotation and a 100 MB per-file size cap.
// Files are named myclaw-YYYYMMDD.log; a symlink myclaw.log always points to the current file.
func InitLogger(workspace string, debug bool) (*zap.Logger, error) {
	logDir := filepath.Join(workspace, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Daily rotation + 100 MB size cap.
	// Pattern: logs/myclaw-20260224.log (new file each day).
	// When the size cap is hit within the same day the timestamp in the suffix
	// gains a time component, e.g. myclaw-20260224T153012.log.
	// logs/myclaw.log is a symlink that always points to the current file.
	rotator, err := rotatelogs.New(
		filepath.Join(logDir, "myclaw-%Y%m%d.log"),
		rotatelogs.WithLinkName(filepath.Join(logDir, "myclaw.log")),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithRotationSize(100*1024*1024), // 100 MB
		rotatelogs.WithMaxAge(30*24*time.Hour),     // keep 30 days
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log rotator: %w", err)
	}

	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	isDaemon := os.Getenv("MYCLAW_DAEMON") == "1"

	var core zapcore.Core
	if isDaemon {
		core = zapcore.NewCore(fileEncoder, zapcore.AddSync(rotator), level)
	} else {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		core = zapcore.NewTee(
			zapcore.NewCore(fileEncoder, zapcore.AddSync(rotator), level),
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
		)
	}

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}
