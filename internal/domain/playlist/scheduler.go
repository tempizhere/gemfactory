// Package playlist содержит планировщик для автоматического обновления плейлистов.
package playlist

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Scheduler управляет автоматическим обновлением плейлиста
type Scheduler struct {
	manager        PlaylistManager
	spotifyClient  SpotifyClientInterface
	playlistURL    string
	updateInterval time.Duration
	logger         *zap.Logger
	stopChan       chan struct{}
	isRunning      bool
	doneChan       chan struct{}
	lastUpdate     time.Time
	mu             sync.RWMutex
}

// NewScheduler создает новый планировщик обновлений плейлиста
func NewScheduler(
	manager PlaylistManager,
	spotifyClient SpotifyClientInterface,
	playlistURL string,
	updateHours int,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		manager:        manager,
		spotifyClient:  spotifyClient,
		playlistURL:    playlistURL,
		updateInterval: time.Duration(updateHours) * time.Hour,
		logger:         logger,
		stopChan:       make(chan struct{}),
		doneChan:       make(chan struct{}),
	}
}

// Start запускает планировщик обновлений
func (s *Scheduler) Start() {
	if s.isRunning {
		s.logger.Warn("Scheduler is already running")
		return
	}

	if s.playlistURL == "" {
		s.logger.Warn("No playlist URL configured, scheduler will not start")
		return
	}

	s.isRunning = true
	s.logger.Info("Starting playlist update scheduler",
		zap.String("playlist_url", s.playlistURL),
		zap.Duration("update_interval", s.updateInterval))

	go s.run()
}

// Stop останавливает планировщик обновлений
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}

	s.logger.Info("Stopping playlist update scheduler")
	close(s.stopChan)

	// Ждем завершения горутины с таймаутом
	select {
	case <-s.doneChan:
		s.logger.Info("Playlist update scheduler stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Playlist update scheduler stop timeout exceeded")
	}

	s.isRunning = false
}

// run выполняет основной цикл обновлений
func (s *Scheduler) run() {
	defer close(s.doneChan)

	// Выполняем первое обновление сразу
	s.updatePlaylist()

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updatePlaylist()
		case <-s.stopChan:
			s.logger.Info("Playlist update scheduler stopped")
			return
		}
	}
}

// updatePlaylist выполняет обновление плейлиста
func (s *Scheduler) updatePlaylist() {
	s.logger.Info("Starting scheduled playlist update",
		zap.String("playlist_url", s.playlistURL))

	// Сохраняем текущее состояние плейлиста
	currentTrackCount := s.manager.GetTotalTracks()
	wasLoaded := s.manager.IsLoaded()

	const maxRetries = 3
	const baseDelay = 30 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Пытаемся загрузить новый плейлист (БЕЗ очистки текущего)
		if err := s.manager.LoadPlaylistFromSpotify(s.playlistURL); err != nil {
			lastErr = err
			s.logger.Warn("Failed to update playlist from Spotify",
				zap.String("playlist_url", s.playlistURL),
				zap.Int("current_tracks", currentTrackCount),
				zap.Bool("was_loaded", wasLoaded),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Error(err))

			// Если это не последняя попытка, ждем и пробуем еще раз
			if attempt < maxRetries {
				delay := time.Duration(attempt) * baseDelay
				s.logger.Info("Retrying playlist update after delay",
					zap.Duration("delay", delay),
					zap.Int("next_attempt", attempt+1))
				time.Sleep(delay)
				continue
			}

			// Все попытки исчерпаны
			s.logger.Error("All playlist update attempts failed, keeping current version",
				zap.String("playlist_url", s.playlistURL),
				zap.Int("current_tracks", currentTrackCount),
				zap.Bool("was_loaded", wasLoaded),
				zap.Error(lastErr))
			return
		}

		// Успешное обновление
		s.mu.Lock()
		s.lastUpdate = time.Now()
		s.mu.Unlock()

		trackCount := s.manager.GetTotalTracks()
		s.logger.Info("Playlist updated successfully",
			zap.String("playlist_url", s.playlistURL),
			zap.Int("tracks_count", trackCount),
			zap.Int("previous_tracks", currentTrackCount),
			zap.Int("attempt", attempt))
		return
	}
}

// GetLastUpdateTime возвращает время последнего обновления плейлиста
func (s *Scheduler) GetLastUpdateTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// GetNextUpdateTime возвращает время следующего обновления плейлиста
func (s *Scheduler) GetNextUpdateTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate.Add(s.updateInterval)
}
