# Gemfactory

Gemfactory is a Telegram bot designed to provide users with schedules of K-pop comebacks and releases for specified months. It fetches data from external sources, filters releases based on predefined whitelists of artists, and presents them in a user-friendly format. The bot is built with Go, containerized using Docker, and deployed via Docker Compose.

## Features

- **K-pop Release Schedules**: Retrieve upcoming K-pop comebacks and releases for a specific month
- **Whitelist Filtering**: Filter releases by female (`-gg`) or male (`-mg`) artists using curated whitelists
- **Dynamic Whitelists**: Admins can manage artist whitelists with commands like `/add_artist` and `/remove_artist`
- **Interactive Interface**: User-friendly keyboard interface for month selection with current, previous, and next month quick access
- **Smart Caching**: Efficient caching system with automatic updates and retry mechanisms
- **Fault Tolerance**: Automatic restart on failures, request rate limiting, and API error handling
- **Timezone Support**: Configurable timezone settings (default: Asia/Seoul)
- **Persistent Storage**: Docker volume-based storage for whitelist data
- **Health Monitoring**: Built-in health check endpoints for container orchestration
- **Metrics Collection**: Comprehensive system metrics and performance monitoring
- **Graceful Shutdown**: Proper resource cleanup and configurable shutdown timeouts

## Commands

### User Commands

- `/start`: Start interaction with the bot and display month selection keyboard
- `/help`: Display available commands and admin contact information
- `/month [month]`: Show K-pop releases for the specified month (e.g., `/month april`)
- `/month [month] -gg`: Show releases only for female artists
- `/month [month] -mg`: Show releases only for male artists
- `/whitelists`: Display lists of female and male artists in a multi-column format
- `/metrics`: Display system metrics including user activity, cache stats, and performance data
- `/homework`: Get a random homework assignment with a track from the playlist and number of times to listen

### Admin Commands

Whitelist management commands are only available to the user specified in `ADMIN_USERNAME` environment variable:

- `/add_artist [female/male] [artists]`: Add artists to the whitelist
- `/remove_artist [artists]`: Remove artists from the whitelist
- `/clearwhitelists`: Clear all whitelists
- `/clearcache`: Clear and reinitialize the cache
- `/export`: Export whitelists
- `/import_playlist [file_path]`: Import playlist from CSV file (use [Chosic Spotify Playlist Exporter](https://www.chosic.com/spotify-playlist-exporter/) to export playlists). **Note**: This will replace the current playlist completely and save it to persistent storage.
- **File Upload**: You can also send a CSV file directly to the bot (only for admin). The bot will automatically detect and load the playlist.

## Prerequisites

- **Go**: Version 1.21 or higher (for development)
- **Docker**: For containerized deployment
- **Docker Compose**: For orchestrating the bot with persistent storage
- **Telegram Bot Token**: Obtain from BotFather in Telegram

## Running the Bot

### Using Docker Compose (Recommended)

1. Create a `docker-compose.yml` file in your project directory:

```yaml
services:
  gemfactory:
    image: tempizhere/gemfactory:latest
    container_name: gemfactory
    pull_policy: always
    restart: unless-stopped
    environment:
      - BOT_TOKEN=your_bot_token
      - ADMIN_USERNAME=your_telegram_username
      - CACHE_DURATION=8h
      - MAX_RETRIES=3
      - REQUEST_DELAY=10s

      - LOG_LEVEL=info
      - TZ=Europe/Moscow
    volumes:
      - app_data:/app/data
volumes:
  app_data:
    name: app_data
```

2. Run the bot:

```bash
docker-compose up -d
```

### Using Docker Container

You can also run the bot directly using Docker:

```bash
docker run -d \
  --name gemfactory \
  --restart unless-stopped \
  -e BOT_TOKEN=your_bot_token \
  -e ADMIN_USERNAME=your_telegram_username \
  -e CACHE_DURATION=8h \
  -e MAX_RETRIES=3 \
  -e REQUEST_DELAY=10s \

  -e LOG_LEVEL=info \
  -e TZ=Europe/Moscow \
  -v app_data:/app/data \
  tempizhere/gemfactory:latest
```

## Environment Variables

### Required Variables

- `BOT_TOKEN`: Telegram bot token from BotFather
- `ADMIN_USERNAME`: Admin's Telegram username (default: fullofsarang)

### Core Settings

- `CACHE_DURATION`: Duration to cache data (default: 24h)
- `MAX_RETRIES`: Maximum number of retries on failures (default: 3)
- `REQUEST_DELAY`: Delay between requests (default: 3s)

- `LOG_LEVEL`: Logging level (default: info)
- `TZ`: Timezone (default: Asia/Seoul)

### Performance Settings

- `MAX_CONCURRENT_REQUESTS`: Maximum concurrent requests (default: 5)
- `RETRY_MAX_ATTEMPTS`: Maximum retry attempts (default: 3)
- `RETRY_INITIAL_DELAY`: Initial retry delay (default: 1s)
- `RETRY_MAX_DELAY`: Maximum retry delay (default: 30s)
- `RETRY_BACKOFF_MULTIPLIER`: Backoff multiplier (default: 2.0)

### HTTP Client Configuration

- `HTTP_MAX_IDLE_CONNS`: Maximum idle connections (default: 100)
- `HTTP_MAX_IDLE_CONNS_PER_HOST`: Maximum idle connections per host (default: 10)
- `HTTP_IDLE_CONN_TIMEOUT`: Idle connection timeout (default: 90s)
- `HTTP_TLS_HANDSHAKE_TIMEOUT`: TLS handshake timeout (default: 10s)
- `HTTP_RESPONSE_HEADER_TIMEOUT`: Response header timeout (default: 30s)
- `HTTP_DISABLE_KEEP_ALIVES`: Disable HTTP keep-alives (default: false)

### Health Check Configuration

- `HEALTH_CHECK_ENABLED`: Enable health check server (default: true)
- `HEALTH_CHECK_PORT`: Port for health check server (default: 8080)
- `HEALTH_CHECK_INTERVAL`: Health check interval (default: 30s)

### Rate Limiting Configuration

- `RATE_LIMIT_ENABLED`: Enable rate limiting (default: true)
- `RATE_LIMIT_REQUESTS`: Rate limit requests per window (default: 10)
- `RATE_LIMIT_WINDOW`: Rate limit window duration (default: 60s)

### Command Cache Configuration

- `COMMAND_CACHE_ENABLED`: Enable command cache (default: true)
- `COMMAND_CACHE_TTL`: Command cache TTL (default: 5m)

### Additional Settings

- `METRICS_ENABLED`: Enable metrics collection (default: false)
- `GRACEFUL_SHUTDOWN_TIMEOUT`: Graceful shutdown timeout (default: 10s)
- `PLAYLIST_CSV_PATH`: Path to the playlist CSV file (optional, if not set playlist will be loaded from persistent storage or via `/import_playlist` command)

  **Playlist Format**: Use [Chosic Spotify Playlist Exporter](https://www.chosic.com/spotify-playlist-exporter/) to export your Spotify playlists to CSV format. The bot expects columns: Song (column 2), Artist (column 3), Spotify Track Id (column 21).

  **Persistence**: Playlists are automatically saved to persistent storage (same directory as whitelists) and will be restored on bot restart.

### Application Data Structure

The bot stores all its data in the `data/` directory:

```
data/
├── female_whitelist.json    # Female artist whitelist
├── male_whitelist.json      # Male artist whitelist
└── playlist.csv             # Music playlist
```

## Deployment

The bot can be deployed using Docker Compose or as a standalone Docker container. Both methods provide the same functionality and persistent storage for whitelist data.

**Docker Compose** is the recommended approach as it simplifies configuration management and provides better container orchestration.

## Project Structure

```
gemfactory/
├── bin/                           # Compiled binaries
├── build/                         # Build artifacts and scripts
├── cmd/                           # Application entry points
│   └── bot/
├── deploy/                        # Deployment configurations
├── internal/                     # Private application code
│   ├── bot/                      # Bot core components
│   │   ├── handlers/            # Command handlers
│   │   ├── keyboard/            # Keyboard management
│   │   ├── middleware/          # Middleware components
│   │   ├── router/              # Request routing
│   │   └── service/             # Business logic services
│   ├── config/                   # Configuration management
│   ├── domain/                   # Domain models and business logic
│   │   ├── artist/              # Artist domain
│   │   ├── release/             # Release domain
│   │   ├── service/             # Service domain
│   │   └── types/               # Common domain types
│   ├── gateway/                  # External service integrations
│   │   ├── scraper/             # Web scraping gateway
│   │   └── telegram/            # Telegram API gateway
│   │       └── botapi/
│   └── infrastructure/          # Infrastructure components
│       ├── cache/               # Caching system
│       ├── debounce/            # Request debouncing
│       ├── health/              # Health monitoring
│       ├── metrics/             # Metrics collection
│       ├── middleware/          # Infrastructure middleware
│       ├── updater/             # Cache updating system
│       └── worker/              # Worker pool system
├── pkg/                          # Public library code
│   └── log/                      # Logging utilities
├── data/                         # Runtime data storage (created at runtime)
└── README.md                     # Project documentation
```

## Development

### Architecture Overview

The project follows Clean Architecture principles with clear separation of concerns:

- **Domain Layer** (`internal/domain/`): Core business logic and entities
- **Application Layer** (`internal/bot/`): Application-specific business rules
- **Infrastructure Layer** (`internal/infrastructure/`): External concerns (database, cache, etc.)
- **Interface Layer** (`internal/gateway/`): External API integrations

### Key Components

#### Bot Core

- **Router**: Command routing and middleware pipeline
- **Handlers**: Command processing logic
- **Services**: Business logic implementation
- **Keyboard Manager**: Dynamic inline keyboard generation

#### Infrastructure

- **Cache System**: Intelligent caching with automatic updates and concurrent protection
- **Worker Pool**: Concurrent job processing with configurable workers
- **Health Check**: Container health monitoring endpoints
- **Metrics**: System performance and usage tracking
- **Rate Limiter**: Request throttling and abuse prevention

#### External Integrations

- **Telegram Bot API**: Official Telegram API wrapper with optimizations
- **Web Scraper**: Resilient HTTP client with retry logic and connection pooling
- **Data Persistence**: JSON-based whitelist storage with atomic operations

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following the project's architecture principles
4. Add tests for new functionality
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to your fork (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
