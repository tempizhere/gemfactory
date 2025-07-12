package worker

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// WorkerPool пул воркеров для обработки обновлений
type WorkerPool struct {
	workers  int
	jobQueue chan Job
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	logger   *zap.Logger
	metrics  *Metrics
}

// Убеждаемся, что WorkerPool реализует WorkerPoolInterface
var _ WorkerPoolInterface = (*WorkerPool)(nil)

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
func NewWorkerPool(workers int, queueSize int, logger *zap.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:  workers,
		jobQueue: make(chan Job, queueSize),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		metrics:  &Metrics{},
	}
}

// Start запускает пул воркеров
func (wp *WorkerPool) Start() {
	wp.logger.Info("Starting worker pool", zap.Int("workers", wp.workers))

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// Stop останавливает пул воркеров
func (wp *WorkerPool) Stop() {
	wp.logger.Info("Stopping worker pool")
	wp.cancel()
	close(wp.jobQueue)
	wp.wg.Wait()
	wp.logger.Info("Worker pool stopped")
}

// Submit добавляет задачу в очередь
func (wp *WorkerPool) Submit(job Job) error {
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
func (wp *WorkerPool) worker(id int) {
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
func (wp *WorkerPool) processJob(job Job, workerID int) {
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
func (wp *WorkerPool) GetMetrics() Metrics {
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
func (wp *WorkerPool) GetProcessedJobs() int64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.processedJobs
}

// GetFailedJobs возвращает количество неудачных задач
func (wp *WorkerPool) GetFailedJobs() int64 {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.failedJobs
}

// GetProcessingTime возвращает общее время обработки
func (wp *WorkerPool) GetProcessingTime() time.Duration {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.processingTime
}

// GetQueueSize возвращает текущий размер очереди
func (wp *WorkerPool) GetQueueSize() int {
	wp.metrics.mu.RLock()
	defer wp.metrics.mu.RUnlock()
	return wp.metrics.queueSize
}

// Ошибки
var (
	ErrQueueFull = &WorkerError{msg: "job queue is full"}
)

// WorkerError ошибка воркера
type WorkerError struct {
	msg string
}

func (e *WorkerError) Error() string {
	return e.msg
}
