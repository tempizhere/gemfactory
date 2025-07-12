// Package worker реализует пул воркеров для асинхронной обработки задач Telegram-бота.
package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Pool пул воркеров для обработки обновлений
type Pool struct {
	workers  int
	jobQueue chan Job
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *zap.Logger
	metrics  *Metrics
	stopOnce sync.Once
	stopped  bool
	mu       sync.RWMutex
}

// Убеждаемся, что Pool реализует PoolInterface
var _ PoolInterface = (*Pool)(nil)

// Job представляет задачу для обработки
type Job struct {
	UpdateID int
	Handler  func() error
	UserID   int64
	Command  string
}

// Metrics метрики воркер пула
type Metrics struct {
	mu             sync.RWMutex
	processedJobs  int64
	failedJobs     int64
	processingTime time.Duration
	queueSize      int
}

// NewWorkerPool создает новый пул воркеров
func NewWorkerPool(workers int, queueSize int, logger *zap.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:  workers,
		jobQueue: make(chan Job, queueSize),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		metrics:  &Metrics{},
	}
}

// Start запускает пул воркеров
func (wp *Pool) Start() {
	wp.logger.Info("Starting worker pool", zap.Int("workers", wp.workers))

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop останавливает пул воркеров
func (wp *Pool) Stop() {
	wp.logger.Info("Stopping worker pool")
	wp.cancel()

	// Безопасное закрытие jobQueue
	wp.stopOnce.Do(func() {
		wp.mu.Lock()
		wp.stopped = true
		wp.mu.Unlock()
		close(wp.jobQueue)
	})

	wp.wg.Wait()
	wp.logger.Info("Worker pool stopped")
}

// Submit добавляет задачу в очередь
func (wp *Pool) Submit(job Job) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.stopped {
		return ErrQueueFull // Or a more specific error indicating the pool is stopped
	}

	select {
	case wp.jobQueue <- job:
		wp.metrics.mu.Lock()
		wp.metrics.queueSize = len(wp.jobQueue)
		wp.metrics.mu.Unlock()
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		return ErrQueueFull
	}
}

// worker основной цикл воркера
func (wp *Pool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("Worker started", zap.Int("worker_id", id))

	for {
		select {
		case job, ok := <-wp.jobQueue:
			if !ok {
				wp.logger.Debug("Worker stopping", zap.Int("worker_id", id))
				return
			}

			wp.processJob(job, id)

		case <-wp.ctx.Done():
			wp.logger.Debug("Worker context cancelled", zap.Int("worker_id", id))
			return
		}
	}
}

// processJob обрабатывает задачу
func (wp *Pool) processJob(job Job, workerID int) {
	startTime := time.Now()

	wp.logger.Debug("Processing job",
		zap.Int("worker_id", workerID),
		zap.Int("update_id", job.UpdateID),
		zap.String("command", job.Command),
		zap.Int64("user_id", job.UserID))

	if err := job.Handler(); err != nil {
		wp.logger.Error("Job processing failed",
			zap.Int("worker_id", workerID),
			zap.Int("update_id", job.UpdateID),
			zap.String("command", job.Command),
			zap.Int64("user_id", job.UserID),
			zap.Error(err))

		wp.metrics.mu.Lock()
		wp.metrics.failedJobs++
		wp.metrics.mu.Unlock()
	} else {
		wp.metrics.mu.Lock()
		wp.metrics.processedJobs++
		wp.metrics.processingTime += time.Since(startTime)
		wp.metrics.mu.Unlock()

		wp.logger.Debug("Job processed successfully",
			zap.Int("worker_id", workerID),
			zap.Int("update_id", job.UpdateID),
			zap.Duration("duration", time.Since(startTime)))
	}
}

// GetMetrics возвращает текущие метрики
func (wp *Pool) GetMetrics() Metrics {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()

	return Metrics{
		processedJobs:  wp.metrics.processedJobs,
		failedJobs:     wp.metrics.failedJobs,
		processingTime: wp.metrics.processingTime,
		queueSize:      wp.metrics.queueSize,
	}
}

// GetProcessedJobs возвращает количество обработанных задач
func (wp *Pool) GetProcessedJobs() int64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.processedJobs
}

// GetFailedJobs возвращает количество неудачных задач
func (wp *Pool) GetFailedJobs() int64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.failedJobs
}

// GetProcessingTime возвращает общее время обработки
func (wp *Pool) GetProcessingTime() time.Duration {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.processingTime
}

// GetQueueSize возвращает текущий размер очереди
func (wp *Pool) GetQueueSize() int {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.queueSize
}

// Ошибки
var (
	ErrQueueFull = &Error{msg: "job queue is full"}
)

// Error ошибка воркера
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
