package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init инициализирует zap-логгер для всего приложения
func Init() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	// Устанавливаем человекочитаемый формат времени
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return cfg.Build()
}
