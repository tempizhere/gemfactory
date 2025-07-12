package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthServer представляет HTTP сервер для health check
type HealthServer struct {
	server    *http.Server
	logger    *zap.Logger
	port      int
	startTime time.Time
}

// Убеждаемся, что HealthServer реализует HealthServerInterface
var _ HealthServerInterface = (*HealthServer)(nil)

// HealthStatus представляет статус здоровья системы
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
}

// NewHealthServer создает новый health check сервер
func NewHealthServer(port int, logger *zap.Logger) *HealthServer {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	hs := &HealthServer{
		server:    server,
		logger:    logger,
		port:      port,
		startTime: time.Now(),
	}

	// Регистрируем маршруты
	mux.HandleFunc("/health", hs.healthHandler)
	mux.HandleFunc("/ready", hs.readyHandler)

	return hs
}

// Start запускает health check сервер
func (hs *HealthServer) Start() error {
	hs.logger.Info("Starting health check server", zap.Int("port", hs.port))
	return hs.server.ListenAndServe()
}

// Stop останавливает health check сервер
func (hs *HealthServer) Stop(ctx context.Context) error {
	hs.logger.Info("Stopping health check server")
	return hs.server.Shutdown(ctx)
}

// healthHandler обрабатывает запросы /health
func (hs *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(hs.startTime).String(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode health status", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// readyHandler обрабатывает запросы /ready
func (hs *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "ready",
		Timestamp: time.Now(),
		Uptime:    time.Since(hs.startTime).String(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(status); err != nil {
		hs.logger.Error("Failed to encode ready status", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
