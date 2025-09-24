// Package health содержит health check сервер.
package health

import (
	"context"
	"fmt"
	"gemfactory/internal/storage"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server представляет health check сервер
type Server struct {
	server *http.Server
	db     *storage.Postgres
	logger *zap.Logger
}

// NewServer создает новый health check сервер
func NewServer(port string, logger *zap.Logger, db *storage.Postgres) *Server {
	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	healthServer := &Server{
		server: server,
		db:     db,
		logger: logger,
	}

	// Регистрируем маршруты
	mux.HandleFunc("/health", healthServer.healthHandler)
	mux.HandleFunc("/ready", healthServer.readyHandler)
	mux.HandleFunc("/live", healthServer.liveHandler)

	return healthServer
}

// Start запускает health check сервер
func (s *Server) Start() error {
	s.logger.Info("Starting health check server", zap.String("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

// Stop останавливает health check сервер
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.logger.Info("Stopping health check server")
	return s.server.Shutdown(ctx)
}

// healthHandler обрабатывает запросы /health
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	code := http.StatusOK

	// Проверяем подключение к базе данных
	if err := s.checkDatabase(); err != nil {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
		s.logger.Error("Health check failed", zap.Error(err))
	}

	// Проверяем другие компоненты
	if err := s.checkComponents(); err != nil {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
		s.logger.Error("Component check failed", zap.Error(err))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":"%s","timestamp":"%s"}`, status, time.Now().Format(time.RFC3339))
}

// readyHandler обрабатывает запросы /ready
func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	status := "ready"
	code := http.StatusOK

	// Проверяем готовность к работе
	if err := s.checkReadiness(); err != nil {
		status = "not ready"
		code = http.StatusServiceUnavailable
		s.logger.Error("Readiness check failed", zap.Error(err))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"status":"%s","timestamp":"%s"}`, status, time.Now().Format(time.RFC3339))
}

// liveHandler обрабатывает запросы /live
func (s *Server) liveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"alive","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

// checkDatabase проверяет подключение к базе данных
func (s *Server) checkDatabase() error {
	if s.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Выполняем простой запрос
	_, err := s.db.GetDB().Query("SELECT 1")
	if err != nil {
		return fmt.Errorf("database query failed: %w", err)
	}

	return nil
}

// checkComponents проверяет другие компоненты
func (s *Server) checkComponents() error {
	// Здесь можно добавить проверки других компонентов
	// Например, проверка подключения к Spotify API, Telegram API и т.д.

	return nil
}

// checkReadiness проверяет готовность к работе
func (s *Server) checkReadiness() error {
	// Проверяем, что все необходимые компоненты инициализированы
	if s.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	// Проверяем подключение к базе данных
	if err := s.checkDatabase(); err != nil {
		return fmt.Errorf("database is not ready: %w", err)
	}

	return nil
}
