package router

import (
	"gemfactory/internal/telegrambot/bot/types"

	"go.uber.org/zap"
)

// Router manages command routes and middleware
type Router struct {
	routes      map[string]types.HandlerFunc
	middlewares []types.Middleware
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]types.HandlerFunc),
	}
}

// Use adds a middleware to the router
func (r *Router) Use(middleware types.Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

// Handle registers a command handler
func (r *Router) Handle(command string, handler types.HandlerFunc) {
	r.routes[command] = handler
}

// Dispatch dispatches a command to its handler
func (r *Router) Dispatch(ctx types.Context) error {
	command := ctx.Message.Command()
	handler, ok := r.routes[command]
	if !ok {
		ctx.Deps.Logger.Warn("Unknown command", zap.String("command", command), zap.Int("update_id", ctx.UpdateID))
		return ctx.Deps.BotAPI.SendMessage(ctx.Message.Chat.ID, "Неизвестная команда. Используйте /help")
	}

	// Create a chain of middleware
	currentHandler := handler
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		mw := r.middlewares[i]
		currentHandler = func(h types.HandlerFunc) types.HandlerFunc {
			return func(c types.Context) error {
				return mw(c, h)
			}
		}(currentHandler)
	}

	if ctx.HandlerExecuted {
		ctx.Deps.Logger.Warn("Handler already executed", zap.String("command", command), zap.Int("update_id", ctx.UpdateID))
		return nil
	}
	ctx.HandlerExecuted = true
	return currentHandler(ctx)
}
