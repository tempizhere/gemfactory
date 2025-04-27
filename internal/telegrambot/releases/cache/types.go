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

var cache = make(map[string]CacheEntry)
var cacheMu sync.RWMutex

// cacheDuration holds the parsed CACHE_DURATION value
var cacheDuration time.Duration
var cacheDurationOnce sync.Once

var activeUpdates int
var activeUpdatesMu sync.Mutex
