package scraper

import (
	"context"
	"gemfactory/internal/external/llm"
)

// LLMClientInterface определяет интерфейс для LLM клиента
type LLMClientInterface interface {
	ParseMultiRelease(ctx context.Context, htmlBlock string, month string) (*llm.MultiReleaseResponse, error)
	ParseSingleBlock(ctx context.Context, htmlBlock string, month string) (*llm.MultiReleaseResponse, error)
	GetMetrics() map[string]interface{}
}
