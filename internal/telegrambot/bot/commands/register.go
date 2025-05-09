package commands

import (
	"gemfactory/internal/telegrambot/bot/router"
	"gemfactory/internal/telegrambot/bot/types"
)

// RegisterRoutes registers all command routes
func RegisterRoutes(r *router.Router, deps *types.Dependencies) {
	deps.Logger.Debug("Registering command routes")
	RegisterUserRoutes(r, deps)
	RegisterAdminRoutes(r, deps)
	deps.Logger.Debug("Command routes registered successfully")
}
