package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client представляет клиент для работы с LLM API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

// Config конфигурация для LLM клиента
type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// MultiReleaseData структура для одного релиза из мультирелиза
type MultiReleaseData struct {
	Date       string `json:"date"`        // "11.08.25"
	Artist     string `json:"artist"`      // "CORTIS"
	Track      string `json:"track"`       // "GO!"
	Album      string `json:"album"`       // "1st EP COLOR OUTSIDE THE LINES"
	YouTubeURL string `json:"youtube_url"` // "https://youtu.be/..."
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
		logger: logger,
	}
}

// ParseMultiRelease парсит мультирелиз через LLM
func (c *Client) ParseMultiRelease(ctx context.Context, htmlBlock string, month string) (*MultiReleaseResponse, error) {
	// Создаем промпт для парсинга мультирелиза
	prompt := c.createMultiReleasePrompt(htmlBlock, month)

	// Показываем полный промпт
	c.logger.Info("Sending request to LLM",
		zap.String("prompt_length", fmt.Sprintf("%d", len(prompt))),
		zap.String("prompt_full", prompt))

	// Отправляем запрос к LLM
	response, err := c.sendRequest(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to LLM: %w", err)
	}

	// Показываем полный ответ
	c.logger.Info("Received response from LLM",
		zap.String("response_length", fmt.Sprintf("%d", len(response))),
		zap.String("response_full", response))

	// Парсим ответ
	multiReleaseResponse, err := c.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	c.logger.Info("Successfully parsed multi-release response", zap.Int("releases_count", len(multiReleaseResponse.Releases)))

	return multiReleaseResponse, nil
}

// createMultiReleasePrompt создает промпт для парсинга мультирелиза
func (c *Client) createMultiReleasePrompt(htmlBlocks string, month string) string {
	return fmt.Sprintf(`Extract ALL releases from HTML blocks. Each block may contain multiple releases. Return ONLY JSON.

IMPORTANT:
1. ALWAYS preserve YouTube links! If you see <a href="https://youtube.com/..."> or <a href="https://youtu.be/...">, extract the full URL.
2. Date format: "Month Day, Year" (e.g., "October 8, 2025") - NO day of week like "Wednesday" or "Monday"!
3. Structure: <event> contains date info, <releases> contains artist and track info
4. FILTER: Only return releases for %s month - ignore releases from other months
5. DEDUPLICATION: Remove duplicate releases (same artist + same date + same track)

HTML blocks (separated by semicolons):
%s

REQUIRED FORMAT (exactly this structure):
{
  "releases": [
    {
      "date": "August 11, 2025",
      "artist": "CORTIS",
      "track": "GO!",
      "album": "1st EP COLOR OUTSIDE THE LINES",
      "youtube_url": "https://youtu.be/..."
    }
  ]
}

CRITICAL RULES:
- HTML blocks are separated by semicolons (;)
- Each HTML block may contain MULTIPLE releases (e.g., one artist with several release dates)
- Extract EVERY release as a separate entry in the releases array
- Look for multiple dates in the same block (e.g., "January 15, 2025", "January 20, 2025", "January 25, 2025")
- Look for multiple tracks in the same block (marked with "• " or "– ")
- Each date + track combination = separate release entry
- If multiple tracks have the same date, create separate entries for each track
   - Date format: "January 1, 2025" (full month name, day, year) - NO day of week (Monday, Tuesday, etc.)
- Artist: exact name (first word/name in the block)
- Track: extract from "Title Track:" field, track names after dates, or bullet points (• / –)
- Album: extract from "Album:" field if available, empty string if not
   - YouTube URL: extract from <a href="https://youtu.be/..."> or <a href="https://www.youtube.com/..."> links, even if the <a> tag is empty (no text content), ALWAYS extract the full URL if found, never leave empty if YouTube link exists

DATE RULES:
- If <releases> contains specific dates (e.g., "January 15, 2025", "January 20, 2025"), use <event> date
- If <releases> has NO specific dates, use the date from <event> for ALL releases in that block
- If <releases> contains multiple dates, extract release only for date from <event>
- YouTube links always come AFTER the track name in the same line

DEDUPLICATION RULES:
- If you find multiple releases with the same artist + same date + same track, keep only ONE
- Prefer releases with YouTube URLs over those without
- If multiple releases have the same data, keep the first one found

EXAMPLES:
- Block with 1 release: 1 entry in releases array
- Block with multiple releases: multiple entries in releases array (one per date+track)
- Block with multiple dates: create separate entry for each date+track combination

DATE EXAMPLES:
- <event>January 15, 2025</event><releases>Artist • Track 1 • Track 2</releases> → 2 releases with date "January 15, 2025"
- <event>January 15, 2025</event><releases>Artist • January 15: Track 1 • January 25: Track 2</releases> → 1 release with date "January 15, 2025"
- <event>January 15, 2025 at 6 PM KST</event><releases>Artist • Track 1</releases> → 1 release with date "January 15, 2025"

Return ONLY valid JSON, no explanations, no markdown.`, month, htmlBlocks)
}

// sendRequest отправляет запрос к LLM API
func (c *Client) sendRequest(ctx context.Context, prompt string) (string, error) {
	request := Request{
		Model: "qwen/qwen2.5-7b-instruct",
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a JSON extraction tool. Extract ALL releases from cleaned text blocks and return ONLY valid JSON in this exact format:\n\n{\n  \"releases\": [\n    {\n      \"date\": \"August 11, 2025\",\n      \"artist\": \"CORTIS\",\n      \"track\": \"GO!\",\n      \"album\": \"1st EP COLOR OUTSIDE THE LINES\",\n      \"youtube_url\": \"https://youtu.be/...\"\n    }\n  ]\n}\n\nReturn ONLY JSON, no explanations, no reasoning, no markdown.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.2,   // Температура как в примере
		TopP:        0.7,   // top_p как в примере
		MaxTokens:   8192,  // Увеличиваем лимит токенов для обработки большого количества блоков
		Stream:      false, // Отключаем streaming для простоты
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

	// Для qwen/qwen2.5-7b-instruct JSON должен быть в content
	c.logger.Info("Extracting JSON from content (qwen2.5-7b-instruct model)")
	return message.Content, nil
}

// parseResponse парсит ответ от LLM в структуру MultiReleaseResponse
func (c *Client) parseResponse(response string) (*MultiReleaseResponse, error) {
	// Очищаем ответ от возможных markdown блоков
	cleanedResponse := response

	// Убираем ```json и ``` если есть
	if bytes.Contains([]byte(response), []byte("```json")) {
		start := bytes.Index([]byte(response), []byte("```json"))
		if start != -1 {
			start += 7 // длина "```json"
			end := bytes.LastIndex([]byte(response), []byte("```"))
			if end != -1 && end > start {
				cleanedResponse = string([]byte(response)[start:end])
			}
		}
	}

	// Ищем последний валидный JSON объект
	lastBrace := bytes.LastIndex([]byte(cleanedResponse), []byte("}"))
	if lastBrace != -1 {
		// Ищем соответствующую открывающую скобку {
		braceCount := 0
		startBrace := -1
		for i := lastBrace; i >= 0; i-- {
			if cleanedResponse[i] == '}' {
				braceCount++
			} else if cleanedResponse[i] == '{' {
				braceCount--
				if braceCount == 0 {
					startBrace = i
					break
				}
			}
		}

		if startBrace != -1 {
			cleanedResponse = cleanedResponse[startBrace : lastBrace+1]
		}
	}

	c.logger.Info("Cleaned response for parsing", zap.String("cleaned", cleanedResponse))

	var multiReleaseResponse MultiReleaseResponse
	if err := json.Unmarshal([]byte(cleanedResponse), &multiReleaseResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal multi-release response: %w", err)
	}

	c.logger.Info("Successfully parsed multi-release response",
		zap.Int("releases_count", len(multiReleaseResponse.Releases)))

	return &multiReleaseResponse, nil
}
