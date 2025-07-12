package worker

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestWorkerPool(t *testing.T) {
	logger := zap.NewNop()
	pool := NewWorkerPool(2, 10, logger)

	// Запускаем пул
	pool.Start()
	defer pool.Stop()

	// Ждем немного для запуска воркеров
	time.Sleep(100 * time.Millisecond)

	// Тестируем обработку задач
	var results sync.Map
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		jobID := i

		job := Job{
			UpdateID: jobID,
			UserID:   int64(jobID),
			Command:  "test",
			Handler: func() error {
				defer wg.Done()
				results.Store(jobID, true)
				return nil
			},
		}

		if err := pool.Submit(job); err != nil {
			t.Errorf("Failed to submit job %d: %v", jobID, err)
		}
	}

	wg.Wait()

	// Проверяем результаты
	for i := 0; i < 5; i++ {
		if _, ok := results.Load(i); !ok {
			t.Errorf("Job %d was not processed", i)
		}
	}

	// Проверяем метрики
	metrics := pool.GetMetrics()
	if metrics.processedJobs != 5 {
		t.Errorf("Expected 5 processed jobs, got %d", metrics.processedJobs)
	}
}

func TestWorkerPoolWithErrors(t *testing.T) {
	logger := zap.NewNop()
	pool := NewWorkerPool(1, 5, logger)

	pool.Start()
	defer pool.Stop()

	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	wg.Add(1)

	job := Job{
		UpdateID: 1,
		UserID:   1,
		Command:  "error_test",
		Handler: func() error {
			defer wg.Done()
			return &Error{msg: "test error"}
		},
	}

	if err := pool.Submit(job); err != nil {
		t.Errorf("Failed to submit job: %v", err)
	}

	wg.Wait()

	// Проверяем метрики
	metrics := pool.GetMetrics()
	if metrics.failedJobs != 1 {
		t.Errorf("Expected 1 failed job, got %d", metrics.failedJobs)
	}
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	pool := NewWorkerPool(1, 5, logger)

	pool.Start()

	// Останавливаем пул
	pool.Stop()

	// Пытаемся отправить задачу после остановки
	job := Job{
		UpdateID: 1,
		UserID:   1,
		Command:  "test",
		Handler: func() error {
			return nil
		},
	}

	if err := pool.Submit(job); err == nil {
		t.Error("Expected error when submitting job to stopped pool")
	}
}

func TestWorkerPoolQueueFull(t *testing.T) {
	logger := zap.NewNop()
	pool := NewWorkerPool(1, 1, logger) // Очень маленькая очередь

	pool.Start()
	defer pool.Stop()

	time.Sleep(100 * time.Millisecond)

	// Создаем канал для синхронизации
	jobStarted := make(chan struct{})
	jobFinished := make(chan struct{})

	// Заполняем очередь долгой задачей
	job1 := Job{
		UpdateID: 1,
		UserID:   1,
		Command:  "test1",
		Handler: func() error {
			close(jobStarted)                  // Сигнализируем, что задача началась
			time.Sleep(500 * time.Millisecond) // Очень долгая задача
			close(jobFinished)                 // Сигнализируем, что задача закончилась
			return nil
		},
	}

	if err := pool.Submit(job1); err != nil {
		t.Errorf("Failed to submit first job: %v", err)
	}

	// Ждем, пока первая задача начнет выполняться
	<-jobStarted

	// Проверяем, что очередь заполнена
	if pool.GetQueueSize() != 0 {
		t.Errorf("Expected queue size 0 after job started, got %d", pool.GetQueueSize())
	}

	// Пытаемся отправить еще одну задачу - должна получить ErrQueueFull
	job2 := Job{
		UpdateID: 2,
		UserID:   2,
		Command:  "test2",
		Handler: func() error {
			return nil
		},
	}

	if err := pool.Submit(job2); err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}

	// Ждем завершения первой задачи
	<-jobFinished
}
