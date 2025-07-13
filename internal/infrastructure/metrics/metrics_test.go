package metrics

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMetrics_TimeUpdates(t *testing.T) {
	logger := zap.NewNop()
	metrics := NewMetrics(logger)

	// Тестируем начальное состояние
	stats := metrics.GetStats()
	system := stats["system"].(map[string]interface{})

	if system["last_cache_update"] != "Не установлено" {
		t.Errorf("Expected 'Не установлено', got %v", system["last_cache_update"])
	}

	if system["next_cache_update"] != "Не установлено" {
		t.Errorf("Expected 'Не установлено', got %v", system["next_cache_update"])
	}

	// Тестируем установку времени последнего обновления
	now := time.Now()
	metrics.SetCacheUpdateStatus(false) // Это установит lastCacheUpdate

	stats = metrics.GetStats()
	system = stats["system"].(map[string]interface{})

	if system["last_cache_update"] == "Не установлено" {
		t.Error("Expected time format, got 'Не установлено'")
	}

	// Тестируем установку времени следующего обновления
	future := now.Add(8 * time.Hour)
	metrics.SetNextCacheUpdate(future)

	stats = metrics.GetStats()
	system = stats["system"].(map[string]interface{})

	if system["next_cache_update"] == "Не установлено" {
		t.Error("Expected time format, got 'Не установлено'")
	}

	expectedFormat := future.Format("02.01.06 15:04")
	if system["next_cache_update"] != expectedFormat {
		t.Errorf("Expected %s, got %v", expectedFormat, system["next_cache_update"])
	}
}

func TestMetrics_FormatTime(t *testing.T) {
	logger := zap.NewNop()
	metrics := NewMetrics(logger)

	// Тест с нулевым временем
	result := metrics.formatTime(time.Time{})
	if result != "Не установлено" {
		t.Errorf("Expected 'Не установлено', got %s", result)
	}

	// Тест с реальным временем
	testTime := time.Date(2024, 12, 25, 15, 30, 0, 0, time.UTC)
	result = metrics.formatTime(testTime)
	expected := "25.12.24 15:30"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestMetrics_FormatDuration(t *testing.T) {
	logger := zap.NewNop()
	metrics := NewMetrics(logger)

	// Тест с коротким периодом (секунды)
	duration := 115556550 * time.Nanosecond // 115.55655ms
	result := metrics.formatDuration(duration)
	expected := "менее часа"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с минутами
	duration = 2*time.Minute + 6*time.Second + 665504400*time.Nanosecond
	result = metrics.formatDuration(duration)
	expected = "менее часа"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с часами
	duration = 2*time.Hour + 30*time.Minute + 15*time.Second
	result = metrics.formatDuration(duration)
	expected = "2 ч 30 мин"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с 1 днем
	duration = 25*time.Hour + 30*time.Minute
	result = metrics.formatDuration(duration)
	expected = "1 день 1 ч"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с 2 днями
	duration = 2*24*time.Hour + 3*time.Hour + 15*time.Minute
	result = metrics.formatDuration(duration)
	expected = "2 дня 3 ч"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с 5 днями
	duration = 5*24*time.Hour + 2*time.Hour + 30*time.Minute
	result = metrics.formatDuration(duration)
	expected = "5 дней 2 ч"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с 1 годом
	duration = 366*24*time.Hour + 5*time.Hour
	result = metrics.formatDuration(duration)
	expected = "1 год 1 дней 5 ч"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с 2 годами
	duration = 2*365*24*time.Hour + 10*24*time.Hour + 3*time.Hour
	result = metrics.formatDuration(duration)
	expected = "2 года 10 дней 3 ч"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestMetrics_SetNextCacheUpdate(t *testing.T) {
	logger := zap.NewNop()
	metrics := NewMetrics(logger)

	// Проверяем начальное состояние
	stats := metrics.GetStats()
	system := stats["system"].(map[string]interface{})
	if system["next_cache_update"] != "Не установлено" {
		t.Errorf("Expected 'Не установлено', got %v", system["next_cache_update"])
	}

	// Устанавливаем время следующего обновления
	future := time.Now().Add(8 * time.Hour)
	metrics.SetNextCacheUpdate(future)

	// Проверяем, что время установлено
	stats = metrics.GetStats()
	system = stats["system"].(map[string]interface{})
	expectedFormat := future.Format("02.01.06 15:04")
	if system["next_cache_update"] != expectedFormat {
		t.Errorf("Expected %s, got %v", expectedFormat, system["next_cache_update"])
	}
}
