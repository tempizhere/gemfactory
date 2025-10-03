// Package logger содержит настройку логгера.
package logger

import (
	"os"
	"path/filepath"

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
	logPath := getLogPath()

	// Проверяем, можем ли мы создать файл логов
	var fileCore zapcore.Core
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		// Если можем создать файл, используем lumberjack
		fileCore = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    100, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}),
			level,
		)
		if err := logFile.Close(); err != nil {
			// Если не удалось закрыть файл, используем консольный вывод
			fileCore = consoleCore
		}
	} else {
		// Если не можем создать файл, используем только консольный вывод
		fileCore = consoleCore
	}

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

// getLogPath получает путь к файлу логов из переменной окружения или использует значение по умолчанию
func getLogPath() string {
	// Сначала проверяем переменную LOG_PATH
	if logPath := os.Getenv("LOG_PATH"); logPath != "" {
		// Создаем директорию для файла логов если она не существует
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0777); err == nil {
			return logPath
		}
		// Если не удалось создать директорию, продолжаем с другими вариантами
	}

	// Затем проверяем APP_DATA_DIR
	if dataDir := os.Getenv("APP_DATA_DIR"); dataDir != "" {
		// Создаем директорию если она не существует
		if err := os.MkdirAll(dataDir, 0777); err == nil {
			return filepath.Join(dataDir, "app.log")
		}
	}

	// По умолчанию используем локальную папку logs
	if err := os.MkdirAll("logs", 0777); err == nil {
		return "logs/app.log"
	}

	// Если ничего не получилось, используем текущую директорию
	return "app.log"
}
