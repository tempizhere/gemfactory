package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Client представляет клиент для работы с LLM API
type Client struct {
	baseURL     string
	apiKey      string
	httpClient  *http.Client
	logger      *zap.Logger
	delay       time.Duration
	lastRequest time.Time
	mu          sync.Mutex
	// Метрики
	requestCount    int64
	successCount    int64
	errorCount      int64
	lastRequestTime time.Time
}

// Config конфигурация для LLM клиента
type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
	Delay   time.Duration
}

// MultiReleaseData структура для одного релиза из мультирелиза
type MultiReleaseData struct {
	Date       string `json:"date"`    // "11.08.25"
	Artist     string `json:"artist"`  // "CORTIS"
	Track      string `json:"track"`   // "GO!"
	Album      string `json:"album"`   // "1st EP COLOR OUTSIDE THE LINES"
	YouTubeURL string `json:"youtube"` // "https://youtu.be/..."
}

// MultiReleaseResponse ответ от LLM с мультирелизами
type MultiReleaseResponse struct {
	Releases []MultiReleaseData `json:"releases"`
}

// Request структура запроса к LLM
type Request struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
	Reasoning   bool      `json:"reasoning,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Message сообщение в чате
type Message struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// Response ответ от LLM
type Response struct {
	Choices []Choice `json:"choices"`
}

// Choice выбор из ответа
type Choice struct {
	Message Message `json:"message"`
}

// NewClient создает новый LLM клиент
func NewClient(config Config, logger *zap.Logger) *Client {
	return &Client{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:      logger,
		delay:       config.Delay,
		lastRequest: time.Time{},
	}
}

// ParseMultiRelease парсит мультирелиз через LLM (устаревший метод)
func (c *Client) ParseMultiRelease(ctx context.Context, htmlBlock string, month string) (*MultiReleaseResponse, error) {
	prompt := c.createComplexBlockPrompt(htmlBlock, month)

	c.logger.Info("Sending request to LLM",
		zap.String("prompt_length", fmt.Sprintf("%d", len(prompt))),
		zap.String("prompt_full", prompt))

	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		c.incrementError()
		return nil, fmt.Errorf("failed to send request to LLM: %w", err)
	}

	c.logger.Info("Received response from LLM",
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_full", response))

	multiReleaseResponse, err := c.parseResponse(response)
	if err != nil {
		c.incrementError()
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.incrementSuccess()
	c.logger.Info("Successfully parsed multi-release response", zap.Int("releases_count", len(multiReleaseResponse.Releases)))

	return multiReleaseResponse, nil
}

// ParseSingleBlock парсит один HTML блок с мультирелизами через LLM с rate limiting
func (c *Client) ParseSingleBlock(ctx context.Context, htmlBlock string, month string) (*MultiReleaseResponse, error) {
	if err := c.enforceRateLimit(); err != nil {
		return nil, fmt.Errorf("rate limit enforcement failed: %w", err)
	}

	prompt := c.createComplexBlockPrompt(htmlBlock, month)

	c.logger.Info("Sending multi-release block request to LLM",
		zap.String("prompt_length", fmt.Sprintf("%d", len(prompt))),
		zap.String("prompt_full", prompt),
		zap.String("month", month))

	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		c.incrementError()
		return nil, fmt.Errorf("failed to send request to LLM: %w", err)
	}

	c.logger.Info("Received response from LLM for multi-release block",
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_full", response))

	multiReleaseResponse, err := c.parseResponse(response)
	if err != nil {
		c.incrementError()
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.incrementSuccess()
	c.logger.Info("Successfully parsed multi-release block response",
		zap.Int("releases_count", len(multiReleaseResponse.Releases)))

	return multiReleaseResponse, nil
}

// enforceRateLimit применяет задержку между запросами
func (c *Client) enforceRateLimit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if !c.lastRequest.IsZero() {
		elapsed := now.Sub(c.lastRequest)
		if elapsed < c.delay {
			sleepDuration := c.delay - elapsed
			c.logger.Debug("Rate limiting: sleeping",
				zap.Duration("sleep_duration", sleepDuration),
				zap.Duration("delay", c.delay))
			time.Sleep(sleepDuration)
		}
	}

	c.lastRequest = time.Now()
	c.requestCount++
	c.lastRequestTime = now
	return nil
}

// GetMetrics возвращает метрики LLM клиента
func (c *Client) GetMetrics() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return map[string]interface{}{
		"total_requests":      c.requestCount,
		"successful_requests": c.successCount,
		"failed_requests":     c.errorCount,
		"last_request_time":   c.lastRequestTime,
		"delay_ms":            c.delay.Milliseconds(),
	}
}

// incrementSuccess увеличивает счетчик успешных запросов
func (c *Client) incrementSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.successCount++
}

// incrementError увеличивает счетчик неудачных запросов
func (c *Client) incrementError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorCount++
}

// createComplexBlockPrompt создает промпт для парсинга сложного блока
func (c *Client) createComplexBlockPrompt(htmlBlock string, month string) string {
	return fmt.Sprintf(`Извлеки все релизы одного артиста из HTML-блока в JSON-массив в формате:
[
  {"artist": "NAME", "date": "DD.MM.YY", "track": "NAME", "album": "NAME", "youtube": "URL"},
  ...
]

Требования:
1. Извлекай "artist" из тега <artist>.
2. Извлекай релизы из блока <need_unparse> ТОЛЬКО за %s месяц. Если в блоке есть даты: используй их для соответствующих релизов, но включай только релизы за %s. Если дат нет: назначь дату из тега <date> для всех релизов.
3. Название трека берется из кавычек (' ' или " " или другие) без суффиксов "MV Release", "Release", "(Title Track)", "PRE-RELEASE" и т.д, сохраняя версии (ENG Ver.) и коллаборации "feat.". ПРИОРИТЕТ: всегда ищи треки в кавычках перед применением других правил.
4. Если в <need_unparse> есть поле "Album" или "OST", извлекай название альбома из него и применяй ко ВСЕМ релизам в <need_unparse>.
7. YouTube-ссылки берутся из тегов <a href=...>YouTube</a> рядом с треком. Ссылка либо встроена в название релиза, либо находится сразу на следующей строке.
8. Возвращай валидный JSON без объяснений и без не ASCII символов.

ВАЖНЫЕ ПРИМЕРЫ:

Пример 1 - фильтрация по месяцу (извлекай ТОЛЬКО за august):
<event><date>August 22, 2025</date><need_unparse><artist>GROUP A</artist>
July 15: "OLD SONG" MV Release
August 11: "SONG 1" MV Release
August 18: "SONG 2" MV Release
August 22: "SONG 3" MV Release
September 5: "FUTURE SONG" MV Release
Album: 2nd Album GROUP A <ALBUM_TITLE></need_unparse></event>
→ [{"artist": "GROUP A", "date": "11.08.25", "track": "SONG 1", "album": "2nd Album GROUP A ALBUM_TITLE", "youtube": ""}, {"artist": "GROUP A", "date": "18.08.25", "track": "SONG 2", "album": "2nd Album GROUP A ALBUM_TITLE", "youtube": ""}, {"artist": "GROUP A", "date": "22.08.25", "track": "SONG 3", "album": "2nd Album GROUP A ALBUM_TITLE", "youtube": ""}]
(ТОЛЬКО релизы за august, релизы за июль и сентябрь игнорируются)

Пример 2 - многотрековый релиз без дат:
<event><date>August 11, 2025</date><need_unparse><artist>GROUP B</artist> (subunit)
Album: Debut EP <ALBUM_NAME>
Title Track & MV:
– <a href="https://youtu.be/abc123">"TRACK 1"</a>
– <a href="https://youtu.be/def456">"TRACK 2"</a>
– <a href="https://youtu.be/ghi789">"TRACK 3"</a></need_unparse></event>
→ [{"artist": "GROUP B", "date": "11.08.25", "track": "TRACK 1", "album": "Debut EP ALBUM_NAME", "youtube": "https://youtu.be/abc123"}, {"artist": "GROUP B", "date": "11.08.25", "track": "TRACK 2", "album": "Debut EP ALBUM_NAME", "youtube": "https://youtu.be/def456"}, {"artist": "GROUP B", "date": "11.08.25", "track": "TRACK 3", "album": "Debut EP ALBUM_NAME", "youtube": "https://youtu.be/ghi789"}]
(Все треки получают дату из <date>, создаются отдельные релизы для каждого трека)

HTML-блок:
%s`, month, month, htmlBlock)
}

// sendRequest отправляет запрос к LLM API
func (c *Client) sendRequest(ctx context.Context, prompt string) (string, error) {
	request := Request{
		Model: "qwen/qwen2.5-7b-instruct",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a JSON extraction tool for K-pop releases. Extract releases from HTML blocks and return ONLY valid JSON array in this exact format:\n\nExtract releases from the provided block, filtering by the specified month. Use dates specified within the block or the <date> tag as fallback.\n\n[\n  {\n    \"artist\": \"ARTIST NAME\",\n    \"date\": \"DD.MM.YY\",\n    \"track\": \"TRACK NAME\",\n    \"album\": \"ALBUM NAME\",\n    \"youtube\": \"https://youtu.be/...\"\n  }\n]\n\nCRITICAL: Return ONLY valid JSON array with standard ASCII characters. No explanations, no reasoning, no markdown, no code blocks, no special Unicode characters like â, é, ñ, etc. Use only standard JSON format.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,
		TopP:        0.7,
		MaxTokens:   8192,
		Stream:      false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	c.logger.Debug("Sending request to LLM", zap.String("url", req.URL.String()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn("Failed to close response body", zap.Error(err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Info("LLM API response",
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_body", string(body)))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	message := response.Choices[0].Message

	return message.Content, nil
}

// parseResponse парсит ответ от LLM в структуру MultiReleaseResponse
func (c *Client) parseResponse(response string) (*MultiReleaseResponse, error) {
	cleanedResponse := response

	// Убираем markdown блоки ```json
	if bytes.Contains([]byte(response), []byte("```json")) {
		start := bytes.Index([]byte(response), []byte("```json"))
		if start != -1 {
			start += 7
			end := bytes.LastIndex([]byte(response), []byte("```"))
			if end != -1 && end > start {
				cleanedResponse = string([]byte(response)[start:end])
			}
		}
	}

	// Ищем последний валидный JSON массив
	lastBracket := bytes.LastIndex([]byte(cleanedResponse), []byte("]"))
	if lastBracket != -1 {
		bracketCount := 0
		startBracket := -1
		for i := lastBracket; i >= 0; i-- {
			if cleanedResponse[i] == ']' {
				bracketCount++
			} else if cleanedResponse[i] == '[' {
				bracketCount--
				if bracketCount == 0 {
					startBracket = i
					break
				}
			}
		}

		if startBracket != -1 {
			cleanedResponse = cleanedResponse[startBracket : lastBracket+1]
		}
	}

	c.logger.Info("Cleaned response for parsing", zap.String("cleaned", cleanedResponse))

	var releases []MultiReleaseData
	if err := json.Unmarshal([]byte(cleanedResponse), &releases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal releases array: %w", err)
	}

	multiReleaseResponse := &MultiReleaseResponse{
		Releases: releases,
	}

	c.logger.Info("Successfully parsed multi-release response",
		zap.Int("releases_count", len(multiReleaseResponse.Releases)))

	return multiReleaseResponse, nil
}
