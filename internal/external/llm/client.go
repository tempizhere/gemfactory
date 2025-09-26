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
	prompt := c.createMultiReleasePrompt(htmlBlock, month)

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

	prompt := c.createSingleBlockPrompt(htmlBlock, month)

	c.logger.Info("Sending multi-release block request to LLM",
		zap.String("prompt_length", fmt.Sprintf("%d", len(prompt))),
		zap.String("month", month))

	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		c.incrementError()
		return nil, fmt.Errorf("failed to send request to LLM: %w", err)
	}

	c.logger.Info("Received response from LLM for multi-release block",
		zap.String("response_length", fmt.Sprintf("%d", len(response))))

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

// createMultiReleasePrompt создает промпт для парсинга мультирелиза
func (c *Client) createMultiReleasePrompt(htmlBlocks string, month string) string {
	return fmt.Sprintf(`Extract release data from HTML blocks into JSON array. Each <event> contains <date>, <artist>, <need_unparse>.

CRITICAL RULES:
1. MONTH FILTER: Only return releases for %s month - ignore ALL other months.
2. JSON ONLY: Return valid JSON array, no explanations, ASCII characters only.
3. YOUTUBE LINKS: Preserve all YouTube URLs, never mix between different artists/blocks.
4. INDEPENDENT PROCESSING: Each <event> block must be processed separately - NEVER copy data between different artists.
5. COMPLETE EXTRACTION: For multi-date blocks, extract ALL releases that match the target month.

DATE ASSIGNMENT RULES:
- If <need_unparse> contains multiple dates: extract releases ONLY for dates matching <date> tag
- If <need_unparse> contains NO dates but multiple releases: assign <date> to ALL releases
- If multiple dates in <need_unparse>: releases may have NO YouTube links (don't borrow links from other releases)

MULTI-TRACK PROCESSING:
- Multi-track releases: create separate releases for each track under "Title Track:" with bullet points ("–")
- Title Track lists: if "Title Track:" contains semicolon-separated tracks ("Track1"; "Track2"), create separate releases for each
- Album: extract from "Album:" field, otherwise empty string
- Track: clean names after "–" or "Title Track", remove 'MV', 'Title Track' (keep 'feat')

CRITICAL: Each artist block must be processed independently - NEVER copy track names between different artists!

ALBUM-ONLY RELEASES:
- If only "Album:" present (no "Title Track:"): use YouTube link text as track name
- Example: "Album: Single – <Club Icarus Remix>" + "Music Video: <a href="...">YouTube</a>" → track: "YouTube"

OUTPUT FORMAT:
[
  {"artist": "NAME", "date": "DD.MM.YY", "track": "NAME", "album": "NAME", "youtube": "URL"},
  ...
]

EXAMPLES:

Multiple dates in <need_unparse> (extract only matching <date>, no borrowed links):
<event><date>October 20, 2025</date><artist>Artist A</artist><need_unparse>September 15, 2025: Track 1: <a href="https://youtu.be/abc123">YouTube</a>
October 20, 2025: Track 2: Album Release</need_unparse></event>
→ [{"artist": "Artist A", "date": "20.10.25", "track": "Track 2", "album": "", "youtube": ""}]

Multiple releases without dates (assign <date> to all) in <need_unparse>:
<event><date>August 13, 2025</date><artist>Artist B</artist><need_unparse>Title Track: – "Song 1" – "Song 2"
Album: Studio Album</need_unparse></event>
→ [{"artist": "Artist B", "date": "13.08.25", "track": "Song 1", "album": "Studio Album", "youtube": ""}, {"artist": "Artist B", "date": "13.08.25", "track": "Song 2", "album": "Studio Album", "youtube": ""}]

Album-only release without Title Track in <need_unparse>:
<event><date>August 13, 2025</date><artist>ARTIST NAME</artist><need_unparse>Album: ALBUM NAME
Music Video: <a href="https://youtube.com/playlist">YouTube</a></need_unparse></event>
→ [{"artist": "ARTIST NAME", "date": "13.08.25", "track": "YouTube", "album": "ALBUM NAME", "youtube": "https://youtube.com/playlist"}]

Multi-date releases (extract ALL matching dates) in <need_unparse>:
<event><date>August 11, 2025</date><artist>ARTIST C</artist><need_unparse>August 11, 2025: Track 1: <a href="https://youtu.be/abc">YouTube</a>
August 18, 2025: Track 2: <a href="https://youtu.be/def">YouTube</a>
August 22, 2025: Track 3: <a href="https://youtu.be/ghi">YouTube</a>
Album: Studio Album</need_unparse></event>
→ [{"artist": "ARTIST C", "date": "11.08.25", "track": "Track 1", "album": "Studio Album", "youtube": "https://youtu.be/abc"}, {"artist": "ARTIST C", "date": "18.08.25", "track": "Track 2", "album": "Studio Album", "youtube": "https://youtu.be/def"}, {"artist": "ARTIST C", "date": "22.08.25", "track": "Track 3", "album": "Studio Album", "youtube": "https://youtu.be/ghi"}]

HTML blocks:
%s`, month, htmlBlocks)
}

// createSingleBlockPrompt создает промпт для парсинга одного блока с мультирелизами
func (c *Client) createSingleBlockPrompt(htmlBlock string, month string) string {
	return fmt.Sprintf(`Extract MULTIPLE releases from HTML block into JSON array. This is a MULTI-RELEASE block for one artist that may contain multiple tracks/releases.

CRITICAL RULES:
1. MONTH FILTER: Only return releases for %s month - ignore ALL other months.
2. JSON ONLY: Return valid JSON array, no explanations, ASCII characters only.
3. YOUTUBE LINKS: Preserve all YouTube URLs.
4. MULTI-RELEASE EXTRACTION: Extract ALL releases from this single artist block that match the target month.

DATE ASSIGNMENT RULES:
- If <need_unparse> contains multiple dates: extract releases ONLY for dates matching <date> tag
- If <need_unparse> contains NO dates but multiple releases: assign <date> to ALL releases
- If multiple dates in <need_unparse>: releases may have NO YouTube links

MULTI-TRACK PROCESSING:
- Multi-track releases: create separate releases for each track under "Title Track:" with bullet points ("–")
- Title Track lists: if "Title Track:" contains semicolon-separated tracks ("Track1"; "Track2"), create separate releases for each
- Album: extract from "Album:" field, otherwise empty string
- Track: clean names after "–" or "Title Track", remove 'MV', 'Title Track' (keep 'feat')

ALBUM-ONLY RELEASES:
- If only "Album:" present (no "Title Track:"): use YouTube link text as track name
- Example: "Album: Single – <Club Icarus Remix>" + "Music Video: <a href="...">YouTube</a>" → track: "YouTube"

OUTPUT FORMAT (JSON array of releases):
[
  {"artist": "NAME", "date": "DD.MM.YY", "track": "NAME", "album": "NAME", "youtube": "URL"},
  {"artist": "NAME", "date": "DD.MM.YY", "track": "NAME", "album": "NAME", "youtube": "URL"},
  ...
]

EXAMPLES:

Single track release:
<event><date>August 13, 2025</date><artist>ARTIST NAME</artist><need_unparse>Title Track: "Song Name"
Album: Studio Album
Music Video: <a href="https://youtu.be/abc">YouTube</a></need_unparse></event>
→ [{"artist": "ARTIST NAME", "date": "13.08.25", "track": "Song Name", "album": "Studio Album", "youtube": "https://youtu.be/abc"}]

Multi-track release:
<event><date>August 13, 2025</date><artist>ARTIST NAME</artist><need_unparse>Title Track: – "Song 1" – "Song 2"
Album: Studio Album</need_unparse></event>
→ [{"artist": "ARTIST NAME", "date": "13.08.25", "track": "Song 1", "album": "Studio Album", "youtube": ""}, {"artist": "ARTIST NAME", "date": "13.08.25", "track": "Song 2", "album": "Studio Album", "youtube": ""}]

Multi-date releases:
<event><date>August 11, 2025</date><artist>ARTIST NAME</artist><need_unparse>August 11, 2025: Track 1: <a href="https://youtu.be/abc">YouTube</a>
August 18, 2025: Track 2: <a href="https://youtu.be/def">YouTube</a>
Album: Studio Album</need_unparse></event>
→ [{"artist": "ARTIST NAME", "date": "11.08.25", "track": "Track 1", "album": "Studio Album", "youtube": "https://youtu.be/abc"}, {"artist": "ARTIST NAME", "date": "18.08.25", "track": "Track 2", "album": "Studio Album", "youtube": "https://youtu.be/def"}]

HTML block:
%s`, month, htmlBlock)
}

// sendRequest отправляет запрос к LLM API
func (c *Client) sendRequest(ctx context.Context, prompt string) (string, error) {
	request := Request{
		Model: "qwen/qwen2.5-7b-instruct",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a JSON extraction tool for K-pop releases. Extract ALL releases from HTML blocks and return ONLY valid JSON array in this exact format:\n\n[\n  {\n    \"artist\": \"ARTIST NAME\",\n    \"date\": \"DD.MM.YY\",\n    \"track\": \"TRACK NAME\",\n    \"album\": \"ALBUM NAME\",\n    \"youtube\": \"https://youtu.be/...\"\n  }\n]\n\nCRITICAL: Return ONLY valid JSON array with standard ASCII characters. No explanations, no reasoning, no markdown, no code blocks, no special Unicode characters like â, é, ñ, etc. Use only standard JSON format.",
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
