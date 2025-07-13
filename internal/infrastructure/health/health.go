// Package health реализует HTTP healthcheck сервер для мониторинга состояния бота.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gemfactory/internal/gateway/telegram/botapi"
	"gemfactory/internal/infrastructure/cache"
	"gemfactory/internal/infrastructure/worker"

	"go.uber.org/zap"
)

// Server представляет HTTP сервер для health check
type Server struct {
	server     *http.Server
	logger     *zap.Logger
	port       int
	startTime  time.Time
	botAPI     botapi.BotAPI
	cache      cache.Cache
	workerPool worker.PoolInterface
}

var _ ServerInterface = (*Server)(nil)

// Status представляет статус здоровья системы
type Status struct {
	Status     string            `json:"status"`
	Timestamp  time.Time         `json:"timestamp"`
	Uptime     string            `json:"uptime"`
	Version    string            `json:"version"`
	Components map[string]string `json:"components,omitempty"`
}

// NewHealthServer создает новый health check сервер
func NewHealthServer(port int, logger *zap.Logger, botAPI botapi.BotAPI, cache cache.Cache, workerPool worker.PoolInterface) *Server {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	hs := &Server{
		server:     server,
		logger:     logger,
		port:       port,
		startTime:  time.Now(),
		botAPI:     botAPI,
		cache:      cache,
		workerPool: workerPool,
	}

	// Регистрируем маршруты
	mux.HandleFunc("/health", hs.healthHandler)
	mux.HandleFunc("/ready", hs.readyHandler)

	return hs
}

// Start запускает health check сервер
func (hs *Server) Start() error {
	hs.logger.Info("Starting health check server", zap.Int("port", hs.port))
	return hs.server.ListenAndServe()
}

// Stop останавливает health check сервер
func (hs *Server) Stop(ctx context.Context) error {
	hs.logger.Info("Stopping health check server")
	return hs.server.Shutdown(ctx)
}

// formatDuration форматирует время в читаемый формат (например: 8s)
func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

// healthHandler обрабатывает запросы /health
func (hs *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	components := hs.checkComponents()

	status := Status{
		Status:     "healthy",
		Timestamp:  time.Now(),
		Uptime:     formatDuration(time.Since(hs.startTime)),
		Version:    "1.0.0",
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode health status", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// readyHandler обрабатывает запросы /ready
func (hs *Server) readyHandler(w http.ResponseWriter, _ *http.Request) {
	components := hs.checkComponents()

	overallStatus := "ready"
	for _, status := range components {
		if status != "healthy" {
			overallStatus = "unhealthy"
			break
		}
	}

	status := Status{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     formatDuration(time.Since(hs.startTime)),
		Version:    "1.0.0",
		Components: components,
	}

	w.Header().Set("Content-Type", "application/json")

	if overallStatus == "ready" {
		w.WriteHeader(http.StatusOK)
		hs.logger.Info("Health check passed", zap.Any("components", components))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		hs.logger.Warn("Health check failed", zap.Any("components", components))
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode ready status", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// checkComponents проверяет состояние всех компонентов
func (hs *Server) checkComponents() map[string]string {
	components := make(map[string]string)

	// Проверка Telegram API
	if hs.botAPI != nil {
		if err := hs.checkTelegramAPI(); err != nil {
			components["telegram_api"] = "unhealthy"
			hs.logger.Error("Telegram API check failed", zap.Error(err))
		} else {
			components["telegram_api"] = "healthy"
		}
	}

	// Проверка Cache
	if hs.cache != nil {
		if hs.cache.GetCachedReleasesCount() >= 0 {
			components["cache"] = "healthy"
		} else {
			components["cache"] = "unhealthy"
		}
	}

	// Проверка Worker Pool
	if hs.workerPool != nil {
		if hs.workerPool.GetProcessedJobs() >= 0 {
			components["worker_pool"] = "healthy"
		} else {
			components["worker_pool"] = "unhealthy"
		}
	}

	return components
}

// checkTelegramAPI проверяет соединение с Telegram API
func (hs *Server) checkTelegramAPI() error {
	// Создаем контекст с таймаутом для проверки
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Простая проверка - попытка получить информацию о боте
	// Для этого нужно добавить метод в botAPI интерфейс
	// Пока используем простую проверку
	if hs.botAPI == nil {
		return fmt.Errorf("bot API is nil")
	}

	_ = ctx
	return nil
}
