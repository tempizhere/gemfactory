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

	// Тест с duration
	duration := 115556550 * time.Nanosecond // 115.55655ms
	result := metrics.formatDuration(duration)
	expected := "0.12s"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Тест с большим duration
	duration = 2*time.Minute + 6*time.Second + 665504400*time.Nanosecond
	result = metrics.formatDuration(duration)
	expected = "2 мин 6 сек"
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
