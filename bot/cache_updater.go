package bot

import (
	"time"

	"new_parser/parser"
	"new_parser/utils"

	"go.uber.org/zap"
)

// StartCacheUpdater runs a background task to periodically update the cache
func StartCacheUpdater(logger *zap.Logger) {
	ticker := time.NewTicker(utils.GetCacheCheckInterval())
	defer ticker.Stop()
	for range ticker.C {
		parser.UpdateCache(logger)
	}
}
