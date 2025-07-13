package router

import (
	"fmt"
	"gemfactory/internal/domain/types"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Router manages command routes and middleware
type Router struct {
	routes      map[string]types.HandlerFunc
	middlewares []types.Middleware
	metrics     *RouterMetrics
	mu          sync.RWMutex
}

// RouterMetrics содержит метрики роутера
type RouterMetrics struct {
	mu               sync.RWMutex
	totalRequests    int64
	totalErrors      int64
	totalDuration    time.Duration
	commandRequests  map[string]int64
	commandErrors    map[string]int64
	commandDurations map[string]time.Duration
}

// NewRouterMetrics создает новые метрики роутера
func NewRouterMetrics() *RouterMetrics {
	return &RouterMetrics{
		commandRequests:  make(map[string]int64),
		commandErrors:    make(map[string]int64),
		commandDurations: make(map[string]time.Duration),
	}
}

var _ Interface = (*Router)(nil)

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		routes:  make(map[string]types.HandlerFunc),
		metrics: NewRouterMetrics(),
	}
}

// Use adds a middleware to the router
func (r *Router) Use(middleware types.Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if middleware == nil {
		return
	}

	r.middlewares = append(r.middlewares, middleware)
}

// Handle registers a command handler
func (r *Router) Handle(command string, handler types.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if command == "" || handler == nil {
		return
	}

	r.routes[command] = handler
}

// Dispatch dispatches a command to its handler
func (r *Router) Dispatch(ctx types.Context) error {
	startTime := time.Now()
	command := ctx.Message.Command()

	// Обновляем метрики в defer
	defer func() {
		duration := time.Since(startTime)
		r.updateMetrics(command, duration, false)
	}()

	r.mu.RLock()
	handler, ok := r.routes[command]
	middlewares := make([]types.Middleware, len(r.middlewares))
	copy(middlewares, r.middlewares)
	r.mu.RUnlock()

	if !ok {
		err := types.NewCommandError(command, ctx.Message.From.ID, ctx.Message.Chat.ID, types.ErrCommandNotFound)
		ctx.Deps.Logger.Warn("Unknown command",
			zap.String("command", command),
			zap.Int("update_id", ctx.UpdateID),
			zap.Error(err))

		// Обновляем метрики с ошибкой
		r.updateMetrics(command, time.Since(startTime), true)

		// Отправляем пользователю сообщение об ошибке
		if sendErr := ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Неизвестная команда. Используйте /help"); sendErr != nil {
			ctx.Deps.Logger.Error("Failed to send error message", zap.Error(sendErr))
		}

		return err
	}

	// Проверяем, не был ли уже выполнен обработчик
	if ctx.HandlerExecuted {
		ctx.Deps.Logger.Warn("Handler already executed",
			zap.String("command", command),
			zap.Int("update_id", ctx.UpdateID))
		return nil
	}

	// Создаем цепочку middleware с защитой от паники
	currentHandler := r.wrapHandlerWithPanicRecovery(handler, ctx.Deps.Logger)
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		currentHandler = r.wrapWithMiddleware(currentHandler, mw)
	}

	ctx.HandlerExecuted = true
	err := currentHandler(ctx)

	if err != nil {
		r.updateMetrics(command, time.Since(startTime), true)
		return types.NewCommandError(command, ctx.Message.From.ID, ctx.Message.Chat.ID, err)
	}

	return nil
}

// wrapHandlerWithPanicRecovery оборачивает обработчик для защиты от паники
func (r *Router) wrapHandlerWithPanicRecovery(handler types.HandlerFunc, logger *zap.Logger) types.HandlerFunc {
	return func(ctx types.Context) (err error) {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				logger.Error("Handler panic recovered",
					zap.String("command", ctx.Message.Command()),
					zap.Int64("user_id", ctx.Message.From.ID),
					zap.Int64("chat_id", ctx.Message.Chat.ID),
					zap.Any("panic", panicErr))

				err = types.NewBotError("PANIC", "handler panicked",
					fmt.Errorf("panic: %v", panicErr))
			}
		}()

		return handler(ctx)
	}
}

// wrapWithMiddleware оборачивает обработчик в middleware
func (r *Router) wrapWithMiddleware(handler types.HandlerFunc, mw types.Middleware) types.HandlerFunc {
	return func(ctx types.Context) error {
		return mw(ctx, handler)
	}
}

// updateMetrics обновляет метрики роутера
func (r *Router) updateMetrics(command string, duration time.Duration, isError bool) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()

	r.metrics.totalRequests++
	r.metrics.totalDuration += duration
	r.metrics.commandRequests[command]++
	r.metrics.commandDurations[command] += duration

	if isError {
		r.metrics.totalErrors++
		r.metrics.commandErrors[command]++
	}
}

// GetMetrics возвращает копию метрик роутера
func (r *Router) GetMetrics() *RouterMetrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	// Создаем глубокую копию
	metrics := RouterMetrics{
		totalRequests:    r.metrics.totalRequests,
		totalErrors:      r.metrics.totalErrors,
		totalDuration:    r.metrics.totalDuration,
		commandRequests:  make(map[string]int64),
		commandErrors:    make(map[string]int64),
		commandDurations: make(map[string]time.Duration),
	}

	for k, v := range r.metrics.commandRequests {
		metrics.commandRequests[k] = v
	}

	for k, v := range r.metrics.commandErrors {
		metrics.commandErrors[k] = v
	}

	for k, v := range r.metrics.commandDurations {
		metrics.commandDurations[k] = v
	}

	return &metrics
}

// GetRegisteredCommands возвращает список зарегистрированных команд
func (r *Router) GetRegisteredCommands() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commands := make([]string, 0, len(r.routes))
	for command := range r.routes {
		commands = append(commands, command)
	}

	return commands
}

// ResetMetrics сбрасывает метрики роутера
func (r *Router) ResetMetrics() {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()

	r.metrics.totalRequests = 0
	r.metrics.totalErrors = 0
	r.metrics.totalDuration = 0
	r.metrics.commandRequests = make(map[string]int64)
	r.metrics.commandErrors = make(map[string]int64)
	r.metrics.commandDurations = make(map[string]time.Duration)
}
