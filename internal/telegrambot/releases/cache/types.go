package cache

import (
	"sync"
	"time"

	"gemfactory/internal/telegrambot/releases/release"
)

// CacheEntry holds cached releases
type CacheEntry struct {
	Releases  []release.Release
	Timestamp time.Time
}

// cache stores the releases with their timestamps
var cache = make(map[string]CacheEntry)

// cacheMu protects the cache map
var cacheMu sync.RWMutex

// cacheDuration holds the parsed CACHE_DURATION value
var cacheDuration time.Duration
var cacheDurationOnce sync.Once

// activeUpdates tracks the number of active cache updates
var activeUpdates int
var activeUpdatesMu sync.Mutex
