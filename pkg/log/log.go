package log

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConfig struct {
	Level      string `env:"LOG_LEVEL" envDefault:"info"`
	Format     string `env:"LOG_FORMAT" envDefault:"json"`
	Output     string `env:"LOG_OUTPUT" envDefault:"stdout"`
	FilePath   string `env:"LOG_FILE_PATH" envDefault:"logs/gemfactory.log"`
	MaxSize    int    `env:"LOG_MAX_SIZE" envDefault:"100"`
	MaxBackups int    `env:"LOG_MAX_BACKUPS" envDefault:"3"`
	MaxAge     int    `env:"LOG_MAX_AGE" envDefault:"28"`
}

// Init инициализирует zap-логгер для всего приложения
func Init() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	// Устанавливаем человекочитаемый формат времени
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.LevelKey = "level"
	return cfg.Build()
}

// InitWithConfig инициализирует логгер с кастомной конфигурацией
func InitWithConfig(config LogConfig) (*zap.Logger, error) {
	// Настройка энкодера
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = "msg"
	encoderConfig.LevelKey = "level"

	// Выбор формата (JSON или Console)
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Настройка уровня логирования
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Настройка вывода
	var cores []zapcore.Core

	// Console output
	if config.Output == "stdout" || config.Output == "both" {
		consoleCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
		cores = append(cores, consoleCore)
	}

	// File output
	if config.Output == "file" || config.Output == "both" {
		// Создаем директорию для логов
		if err := os.MkdirAll(filepath.Dir(config.FilePath), 0755); err != nil {
			return nil, err
		}

		// Настройка ротации логов
		rotator := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize, // MB
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge, // days
			Compress:   true,
		}

		fileCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(rotator),
			level,
		)
		cores = append(cores, fileCore)
	}

	// Создаем логгер с несколькими ядрами
	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// NewNop создает пустой логгер для случаев, когда логирование не нужно
func NewNop() *zap.Logger {
	return zap.NewNop()
}
