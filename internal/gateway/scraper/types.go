package scraper

import (
	"context"
	"gemfactory/internal/domain/release"
)

// Fetcher defines the interface for scraping operations
type Fetcher interface {
	FetchMonthlyLinks(ctx context.Context, months []string) ([]string, error)
	ParseMonthlyPage(ctx context.Context, url, month string, whitelist map[string]struct{}) ([]release.Release, error)
}
