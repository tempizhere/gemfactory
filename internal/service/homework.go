// Package service содержит бизнес-логику приложения.
package service

import (
	"fmt"
	"gemfactory/internal/model"
	"gemfactory/internal/storage/repository"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// HomeworkService содержит бизнес-логику для работы с домашними заданиями
type HomeworkService struct {
	playlistRepo    model.PlaylistTracksRepository
	trackingRepo    model.HomeworkTrackingRepository
	configRepo      model.ConfigRepository
	playlistService *PlaylistService
	taskService     *TaskService
	logger          *zap.Logger
}

// NewHomeworkService создает новый сервис домашних заданий
func NewHomeworkService(db *bun.DB, playlistService *PlaylistService, taskService *TaskService, logger *zap.Logger) *HomeworkService {
	return &HomeworkService{
		playlistRepo:    repository.NewPlaylistTracksRepository(db, logger),
		trackingRepo:    repository.NewHomeworkTrackingRepository(db, logger),
		configRepo:      repository.NewConfigRepository(db, logger),
		playlistService: playlistService,
		taskService:     taskService,
		logger:          logger,
	}
}

// GetRandomHomework возвращает случайное домашнее задание для пользователя
func (s *HomeworkService) GetRandomHomework(userID int64) (*model.Homework, error) {
	// Проверяем, что PlaylistService доступен
	if s.playlistService == nil {
		return nil, fmt.Errorf("playlist service not available")
	}

	// Получаем информацию о плейлисте через PlaylistService
	playlistInfo, err := s.playlistService.GetPlaylistInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist info: %w", err)
	}

	spotifyID := playlistInfo.SpotifyID
	s.logger.Info("Using Spotify ID from playlist service", zap.String("spotify_id", spotifyID))

	// Проверяем может ли пользователь запросить новое домашнее задание
	canRequest, err := s.canUserRequestHomework(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user can request homework: %w", err)
	}

	if !canRequest {
		return nil, fmt.Errorf("user cannot request homework yet, please wait")
	}

	// Получаем уже выданные треки пользователю
	issuedTrackIDs, err := s.trackingRepo.GetIssuedTrackIDs(userID, spotifyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issued track IDs: %w", err)
	}

	// Получаем случайный трек из плейлиста, исключая уже выданные
	s.logger.Info("Getting random track", zap.String("spotify_id", spotifyID), zap.Strings("exclude_track_ids", issuedTrackIDs))
	track, err := s.playlistRepo.GetRandomTrack(spotifyID, issuedTrackIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get random track from playlist: %w", err)
	}

	s.logger.Info("Got track from playlist", zap.Bool("track_found", track != nil))
	if track != nil {
		s.logger.Info("Track details", zap.String("track_id", track.TrackID), zap.String("artist", track.Artist), zap.String("title", track.Title))
	}

	if track == nil {
		// Если все треки уже выданы, возвращаем первый незавершенный
		pendingTrackings, err := s.trackingRepo.GetPendingByUserID(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get pending homework: %w", err)
		}

		if len(pendingTrackings) == 0 {
			return nil, fmt.Errorf("no tracks available for homework")
		}

		// Возвращаем первый незавершенный трек
		pending := pendingTrackings[0]
		return &model.Homework{
			UserID:    userID,
			TrackID:   pending.TrackID,
			Artist:    "",                // Будет заполнено из плейлиста
			Title:     "",                // Будет заполнено из плейлиста
			PlayCount: pending.PlayCount, // Используем сохраненное количество
			Completed: false,
		}, nil
	}

	// Генерируем случайное количество прослушиваний (1-6)
	playCount := rand.Intn(6) + 1

	// Создаем отслеживание выданного домашнего задания
	tracking := &model.HomeworkTracking{
		UserID:    userID,
		TrackID:   track.TrackID,
		SpotifyID: spotifyID,
		PlayCount: playCount,
		IssuedAt:  time.Now(),
	}

	err = s.trackingRepo.Create(tracking)
	if err != nil {
		return nil, fmt.Errorf("failed to create homework tracking: %w", err)
	}

	// Создаем новое домашнее задание
	homework := &model.Homework{
		UserID:    userID,
		TrackID:   track.TrackID,
		Artist:    track.Artist,
		Title:     track.Title,
		PlayCount: playCount,
		Completed: false,
	}

	return homework, nil
}

// MarkCompleted отмечает домашнее задание как завершенное
func (s *HomeworkService) MarkCompleted(userID int64, trackID string) error {
	// Проверяем, что PlaylistService доступен
	if s.playlistService == nil {
		return fmt.Errorf("playlist service not available")
	}

	// Получаем информацию о плейлисте через PlaylistService
	playlistInfo, err := s.playlistService.GetPlaylistInfo()
	if err != nil {
		return fmt.Errorf("failed to get playlist info: %w", err)
	}

	spotifyID := playlistInfo.SpotifyID
	if spotifyID == "" {
		return fmt.Errorf("failed to extract Spotify ID from playlist URL")
	}

	err = s.trackingRepo.MarkCompleted(userID, trackID, spotifyID)
	if err != nil {
		return fmt.Errorf("failed to mark homework as completed: %w", err)
	}

	return nil
}

// GetUserHomework возвращает домашние задания пользователя
func (s *HomeworkService) GetUserHomework(userID int64) ([]model.HomeworkTracking, error) {
	return s.trackingRepo.GetByUserID(userID)
}

// GetPendingHomework возвращает незавершенные домашние задания пользователя
func (s *HomeworkService) GetPendingHomework(userID int64) ([]model.HomeworkTracking, error) {
	return s.trackingRepo.GetPendingByUserID(userID)
}

// CanRequestHomework проверяет может ли пользователь запросить новое домашнее задание
func (s *HomeworkService) CanRequestHomework(userID int64) (bool, error) {
	return s.canUserRequestHomework(userID)
}

// GetTimeUntilNextRequest возвращает время до следующего возможного запроса
func (s *HomeworkService) GetTimeUntilNextRequest(userID int64) time.Duration {
	// Получаем время сброса из задачи homework_reset_daily
	resetTime, err := s.getHomeworkResetTime()
	if err != nil {
		s.logger.Error("Failed to get homework reset time", zap.Error(err))
		return 0
	}

	// Парсим время сброса (формат HH:MM)
	timeParts := strings.Split(resetTime, ":")
	if len(timeParts) != 2 {
		s.logger.Error("Invalid time format", zap.String("time", resetTime))
		return 0
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		s.logger.Error("Invalid hour", zap.String("hour", timeParts[0]), zap.Error(err))
		return 0
	}

	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		s.logger.Error("Invalid minute", zap.String("minute", timeParts[1]), zap.Error(err))
		return 0
	}

	// Вычисляем время следующего сброса
	now := time.Now()
	nextReset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// Если время сброса уже прошло сегодня, следующий сброс завтра
	if nextReset.Before(now) {
		nextReset = nextReset.AddDate(0, 0, 1)
	}

	// Возвращаем время до следующего сброса
	return nextReset.Sub(now)
}

// GetActiveHomework возвращает активное домашнее задание пользователя
func (s *HomeworkService) GetActiveHomework(userID int64) (*model.Homework, error) {
	pendingTrackings, err := s.trackingRepo.GetPendingByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending homework: %w", err)
	}

	if len(pendingTrackings) == 0 {
		return nil, nil // Нет активных домашних заданий
	}

	latest := pendingTrackings[0]

	// Получаем информацию о треке из плейлиста
	tracks, err := s.playlistRepo.GetBySpotifyID(latest.SpotifyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}

	var track *model.PlaylistTracks
	for _, t := range tracks {
		if t.TrackID == latest.TrackID {
			track = &t
			break
		}
	}

	if track == nil {
		return nil, fmt.Errorf("track not found in playlist")
	}

	return &model.Homework{
		UserID:    userID,
		TrackID:   track.TrackID,
		Artist:    track.Artist,
		Title:     track.Title,
		PlayCount: latest.PlayCount, // Используем сохраненное количество
		Completed: false,
	}, nil
}

// ResetAllHomework сбрасывает все домашние задания (отмечает как выполненные)
func (s *HomeworkService) ResetAllHomework() error {
	s.logger.Info("Starting homework reset for all users")

	// Получаем все незавершенные домашние задания
	pendingTrackings, err := s.trackingRepo.GetAllPending()
	if err != nil {
		return fmt.Errorf("failed to get pending homework: %w", err)
	}

	if len(pendingTrackings) == 0 {
		s.logger.Info("No pending homework to reset")
		return nil
	}

	resetCount := 0
	for _, tracking := range pendingTrackings {
		err = s.trackingRepo.MarkCompleted(tracking.UserID, tracking.TrackID, tracking.SpotifyID)
		if err != nil {
			s.logger.Error("Failed to mark homework as completed during reset",
				zap.Int64("user_id", tracking.UserID),
				zap.String("track_id", tracking.TrackID),
				zap.Error(err))
			continue
		}
		resetCount++
	}

	s.logger.Info("Homework reset completed", zap.Int("reset_count", resetCount))
	return nil
}

// canUserRequestHomework проверяет может ли пользователь запросить новое домашнее задание
// с учетом времени сброса из задачи homework_reset_daily
func (s *HomeworkService) canUserRequestHomework(userID int64) (bool, error) {
	// Получаем время сброса из задачи homework_reset_daily
	resetTime, err := s.getHomeworkResetTime()
	if err != nil {
		return false, fmt.Errorf("failed to get homework reset time: %w", err)
	}

	// Парсим время сброса (формат HH:MM)
	timeParts := strings.Split(resetTime, ":")
	if len(timeParts) != 2 {
		return false, fmt.Errorf("invalid time format: %s, expected HH:MM", resetTime)
	}

	hour, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return false, fmt.Errorf("invalid hour: %s", timeParts[0])
	}

	minute, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return false, fmt.Errorf("invalid minute: %s", timeParts[1])
	}

	lastTime, err := s.trackingRepo.GetLastRequestTime(userID)
	if err != nil {
		return false, fmt.Errorf("failed to get last request time: %w", err)
	}

	// Если домашних заданий не было, можно запросить
	if lastTime == nil {
		return true, nil
	}

	// Вычисляем время следующего сброса
	now := time.Now()
	nextReset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// Если время сброса уже прошло сегодня, следующий сброс завтра
	if nextReset.Before(now) {
		nextReset = nextReset.AddDate(0, 0, 1)
	}

	return lastTime.Before(nextReset.AddDate(0, 0, -1)), nil
}

// getHomeworkResetTime получает время сброса из задачи homework_reset_daily
func (s *HomeworkService) getHomeworkResetTime() (string, error) {
	// Получаем задачу homework_reset_daily
	task, err := s.taskService.GetByName("homework_reset_daily")
	if err != nil {
		return "", fmt.Errorf("failed to get homework_reset_daily task: %w", err)
	}

	if task == nil {
		return "", fmt.Errorf("homework_reset_daily task not found")
	}

	// Парсим cron выражение (формат: "0 0 * * *" = минута час день месяц день_недели)
	// Нас интересует час (второй элемент)
	cronParts := strings.Fields(task.CronExpression)
	if len(cronParts) < 2 {
		return "", fmt.Errorf("invalid cron expression: %s", task.CronExpression)
	}

	hour := cronParts[1]
	minute := cronParts[0]

	// Формируем время в формате HH:MM
	resetTime := fmt.Sprintf("%s:%s", hour, minute)

	s.logger.Debug("Extracted homework reset time from task",
		zap.String("cron_expression", task.CronExpression),
		zap.String("reset_time", resetTime))

	return resetTime, nil
}
