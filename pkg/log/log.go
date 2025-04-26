package log

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init initializes the global logger with configurable log level
func Init() (*zap.Logger, error) {
	cfg := zap.Config{
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:     "msg",
			LevelKey:       "level",
			TimeKey:        "time",
			CallerKey:      "caller",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		Development: true, // Включаем режим разработки для отладки
	}

	// Читаем LOG_LEVEL из окружения
	logLevelRaw := os.Getenv("LOG_LEVEL")
	logLevel := strings.ToLower(logLevelRaw)
	if logLevel == "" {
		logLevel = "info" // Фаллбэк на info
	}

	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		logLevel = "info"
		level = zapcore.InfoLevel
	}

	cfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %v", err)
	}

	// Логируем инициализацию логгера с подробной информацией
	logger.Info("Logger initialized",
		zap.String("log_level_raw", logLevelRaw),
		zap.String("log_level_applied", logLevel),
		zap.String("level", level.String()),
	)
	if logLevelRaw == "" {
		logger.Warn("LOG_LEVEL not set in environment, using default", zap.String("default", logLevel))
	}
	if level == zapcore.DebugLevel {
		logger.Debug("Debug logging enabled")
	}

	return logger, nil
}
