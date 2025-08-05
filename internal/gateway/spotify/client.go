// Package spotify реализует клиент для работы с Spotify Web API.
package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zmb3/spotify/v2"
	"go.uber.org/zap"
)

// tokenTransport добавляет токен к каждому запросу
type tokenTransport struct {
	base      http.RoundTripper
	token     string
	tokenType string
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.tokenType+" "+t.token)

	// Используем DefaultTransport если base равен nil
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	return base.RoundTrip(req)
}

// Client представляет клиент для работы с Spotify API
type Client struct {
	clientID     string
	clientSecret string
	logger       *zap.Logger
}

// NewClient создает новый Spotify клиент с использованием Client Credentials Flow
func NewClient(clientID, clientSecret string, logger *zap.Logger) (*Client, error) {
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("spotify client ID and secret are required")
	}

	logger.Info("Spotify client created successfully with client credentials flow")

	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		logger:       logger,
	}, nil
}

// createSpotifyClient создает новый Spotify клиент для каждого запроса
func (c *Client) createSpotifyClient() (*spotify.Client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем HTTP клиент
	httpClient := &http.Client{}

	// Подготавливаем данные для запроса токена согласно документации Spotify
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Создаем запрос
	req, err := http.NewRequestWithContext(ctx, "POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	// Устанавливаем заголовки согласно документации
	credentials := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+credentials)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Выполняем запрос
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("Failed to close response body", zap.Error(closeErr))
		}
	}()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Парсим ответ
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("no access token received")
	}

	// Создаем HTTP клиент с токеном в заголовках
	tokenClient := &http.Client{
		Transport: &tokenTransport{
			base:      http.DefaultTransport, // Используем DefaultTransport вместо nil
			token:     tokenResponse.AccessToken,
			tokenType: tokenResponse.TokenType,
		},
	}

	// Создаем Spotify клиент с HTTP клиентом, который автоматически добавляет токен
	client := spotify.New(tokenClient)

	c.logger.Debug("Created new Spotify client for request")

	return client, nil
}

// ExtractPlaylistID извлекает ID плейлиста из URL
func (c *Client) ExtractPlaylistID(playlistURL string) (string, error) {
	// Поддерживаем разные форматы URL:
	// https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
	// spotify:playlist:37i9dQZF1DXcBWIGoYBM5M

	if strings.HasPrefix(playlistURL, "spotify:playlist:") {
		return strings.TrimPrefix(playlistURL, "spotify:playlist:"), nil
	}

	if strings.Contains(playlistURL, "open.spotify.com/playlist/") {
		parts := strings.Split(playlistURL, "/playlist/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid playlist URL format")
		}
		// Убираем возможные параметры после ID
		playlistID := strings.Split(parts[1], "?")[0]
		return playlistID, nil
	}

	return "", fmt.Errorf("unsupported playlist URL format")
}

// GetPlaylistTracks получает треки из публичного плейлиста
func (c *Client) GetPlaylistTracks(playlistURL string) ([]*Track, error) {
	playlistID, err := c.ExtractPlaylistID(playlistURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract playlist ID: %w", err)
	}

	// Создаем новый Spotify клиент для каждого запроса
	c.logger.Debug("Creating new Spotify client for playlist tracks request")
	client, err := c.createSpotifyClient()
	if err != nil {
		c.logger.Error("Failed to create Spotify client", zap.Error(err))
		return nil, fmt.Errorf("failed to create spotify client: %w", err)
	}

	ctx := context.Background()

	var allTracks []*Track
	offset := 0
	limit := 100 // Максимальный размер страницы для Spotify API

	c.logger.Debug("Starting pagination to get all playlist tracks",
		zap.String("playlist_id", playlistID))

	for {
		// Получаем страницу треков плейлиста
		c.logger.Debug("Requesting playlist items page",
			zap.String("playlist_id", playlistID),
			zap.Int("offset", offset),
			zap.Int("limit", limit))

		tracks, err := client.GetPlaylistItems(ctx, spotify.ID(playlistID), spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			c.logger.Error("Spotify API request failed",
				zap.String("playlist_id", playlistID),
				zap.Int("offset", offset),
				zap.Error(err))
			return nil, fmt.Errorf("failed to get playlist tracks at offset %d: %w", offset, err)
		}

		c.logger.Debug("Retrieved playlist items page",
			zap.String("playlist_id", playlistID),
			zap.Int("offset", offset),
			zap.Int("items_in_page", len(tracks.Items)),
			zap.Int("total_items", int(tracks.Total)))

		// Обрабатываем треки на текущей странице
		for _, item := range tracks.Items {
			// Проверяем, что это трек, а не эпизод
			if item.Track.Track == nil {
				continue
			}

			artistName := "Unknown Artist"
			if len(item.Track.Track.Artists) > 0 {
				artistName = item.Track.Track.Artists[0].Name
			}

			allTracks = append(allTracks, &Track{
				ID:     string(item.Track.Track.ID),
				Title:  item.Track.Track.Name,
				Artist: artistName,
			})
		}

		// Проверяем, есть ли еще страницы
		if offset+len(tracks.Items) >= int(tracks.Total) {
			break
		}

		offset += len(tracks.Items)
	}

	c.logger.Info("Successfully retrieved all tracks from playlist",
		zap.String("playlist_id", playlistID),
		zap.Int("total_tracks", len(allTracks)))

	return allTracks, nil
}

// GetPlaylistInfo получает информацию о плейлисте
func (c *Client) GetPlaylistInfo(playlistURL string) (*PlaylistInfo, error) {
	playlistID, err := c.ExtractPlaylistID(playlistURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract playlist ID: %w", err)
	}

	// Создаем новый Spotify клиент для каждого запроса
	c.logger.Debug("Creating new Spotify client for playlist info request")
	client, err := c.createSpotifyClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create spotify client: %w", err)
	}

	ctx := context.Background()

	c.logger.Debug("Requesting playlist info from Spotify API")
	playlist, err := client.GetPlaylist(ctx, spotify.ID(playlistID))
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist: %w", err)
	}

	return &PlaylistInfo{
		ID:          string(playlist.ID),
		Name:        playlist.Name,
		Description: playlist.Description,
		TotalTracks: int(playlist.Tracks.Total),
		Public:      playlist.IsPublic,
		Owner:       playlist.Owner.DisplayName,
	}, nil
}
