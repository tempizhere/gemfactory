// Package logger содержит настройку логгера.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New создает новый логгер
func New() *zap.Logger {
	// Настраиваем уровень логирования
	level := getLogLevel()

	// Настраиваем кодировщик
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Настраиваем вывод
	var core zapcore.Core

	// Консольный вывод
	consoleCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Файловый вывод
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&lumberjack.Logger{
			Filename:   "/app/logs/app.log",
			MaxSize:    100, // MB
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true,
		}),
		level,
	)

	// Объединяем выводы
	core = zapcore.NewTee(consoleCore, fileCore)

	// Создаем логгер
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger
}

// getLogLevel получает уровень логирования из переменной окружения
func getLogLevel() zapcore.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
