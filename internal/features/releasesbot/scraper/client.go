package scraper

import (
	"context"
	"strings"
	"sync"
	"time"
	"fmt"

	"gemfactory/internal/features/releasesbot/release"
	"gemfactory/pkg/config"
	"github.com/gocolly/colly/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GetMonthlyLinksWithContext retrieves links to monthly schedules with context
func GetMonthlyLinksWithContext(ctx context.Context, months []string, config *config.Config, logger *zap.Logger) ([]string, error) {
	if logger.Core().Enabled(zapcore.DebugLevel) {
		logger.Debug("Starting GetMonthlyLinksWithContext", zap.Strings("months", months))
	}
	var monthlyLinks []string
	uniqueLinks := make(map[string]struct{})
	var err error

	// Пробуем до трёх раз при таймаутах
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-ctx.Done():
			logger.Warn("GetMonthlyLinks stopped due to context cancellation", zap.Error(ctx.Err()))
			return nil, ctx.Err()
		default:
			if logger.Core().Enabled(zapcore.DebugLevel) {
				logger.Debug("GetMonthlyLinks attempt", zap.Int("attempt", attempt+1))
			}
			collector := NewCollector(config, logger)
			var mu sync.Mutex
			collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
				select {
				case <-ctx.Done():
					logger.Warn("OnHTML processing stopped due to context cancellation", zap.String("url", e.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					link := e.Attr("href")
					if strings.Contains(link, "kpop-comeback-schedule-") &&
						strings.Contains(link, "https://kpopofficial.com/") &&
						strings.Contains(link, release.CurrentYear()) {
						if len(months) > 0 {
							for _, month := range months {
								if strings.Contains(strings.ToLower(link), strings.ToLower(month)) {
									mu.Lock()
									if _, exists := uniqueLinks[link]; !exists {
										uniqueLinks[link] = struct{}{}
										monthlyLinks = append(monthlyLinks, link)
										if logger.Core().Enabled(zapcore.DebugLevel) {
											logger.Debug("Added link", zap.String("link", link), zap.String("month", month))
										}
									}
									mu.Unlock()
								}
							}
						} else {
							mu.Lock()
							if _, exists := uniqueLinks[link]; !exists {
								uniqueLinks[link] = struct{}{}
								monthlyLinks = append(monthlyLinks, link)
								if logger.Core().Enabled(zapcore.DebugLevel) {
									logger.Debug("Added link", zap.String("link", link))
								}
							}
							mu.Unlock()
						}
					}
				}
			})

			collector.OnRequest(func(r *colly.Request) {
				select {
				case <-ctx.Done():
					logger.Warn("OnRequest stopped due to context cancellation", zap.String("url", r.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Visiting URL", zap.String("url", r.URL.String()))
					}
					r.Ctx.Put("start_time", time.Now())
				}
			})

			collector.OnResponse(func(r *colly.Response) {
				select {
				case <-ctx.Done():
					logger.Warn("OnResponse stopped due to context cancellation", zap.String("url", r.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					startTime, _ := r.Ctx.GetAny("start_time").(time.Time)
					if logger.Core().Enabled(zapcore.DebugLevel) {
						logger.Debug("Received response", zap.String("url", r.Request.URL.String()), zap.Duration("duration", time.Since(startTime)))
					}
				}
			})

			collector.OnError(func(r *colly.Response, err error) {
				select {
				case <-ctx.Done():
					logger.Warn("Error processing stopped due to context cancellation", zap.String("url", r.Request.URL.String()), zap.Error(ctx.Err()))
					return
				default:
					logger.Error("Failed to scrape links", zap.String("url", r.Request.URL.String()), zap.Error(err))
				}
			})

			url := "https://kpopofficial.com/category/kpop-comeback-schedule/"
			requestCtx, requestCancel := context.WithTimeout(ctx, 90*time.Second)
			defer requestCancel()

			err = collector.Visit(url)
			collector.Wait()

			select {
			case <-requestCtx.Done():
				logger.Warn("Scraping links timed out", zap.String("url", url), zap.Error(requestCtx.Err()))
				if attempt < 2 {
					logger.Info("Retrying GetMonthlyLinks due to timeout", zap.Int("attempt", attempt+1))
					monthlyLinks = nil
					uniqueLinks = make(map[string]struct{})
					continue
				}
				return nil, requestCtx.Err()
			default:
				if err != nil {
					logger.Error("Failed to visit main page", zap.String("url", url), zap.Error(err))
					if attempt < 2 {
						logger.Info("Retrying GetMonthlyLinks due to error", zap.Int("attempt", attempt+1), zap.Error(err))
						monthlyLinks = nil
						uniqueLinks = make(map[string]struct{})
						continue
					}
					return nil, fmt.Errorf("failed to visit main page: %v", err)
				}
				if logger.Core().Enabled(zapcore.DebugLevel) {
					logger.Debug("Completed scraping links", zap.String("url", url), zap.Int("link_count", len(monthlyLinks)))
				}
				return monthlyLinks, nil
			}
		}
	}

	return monthlyLinks, nil
}
