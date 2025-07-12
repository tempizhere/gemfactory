# Gemfactory

Gemfactory is a Telegram bot designed to provide users with schedules of K-pop comebacks and releases for specified months. It fetches data from external sources, filters releases based on predefined whitelists of artists, and presents them in a user-friendly format. The bot is built with Go, containerized using Docker, and deployed via GitHub Actions to Docker Hub.

![2025-06-21_01-05](https://github.com/user-attachments/assets/3f1ba3d7-1084-498b-b8e5-18ce49de32a3)


## Features

- **K-pop Release Schedules**: Retrieve upcoming K-pop comebacks and releases for a specific month
- **Whitelist Filtering**: Filter releases by female (`-gg`) or male (`-mg`) artists using curated whitelists
- **Dynamic Whitelists**: Admins can manage artist whitelists with commands like `/add_artist` and `/remove_artist`
- **Interactive Interface**: User-friendly keyboard interface for month selection with current, previous, and next month quick access
- **Smart Caching**: Efficient caching system with automatic updates and retry mechanisms
- **Fault Tolerance**: Automatic restart on failures, request rate limiting, and API error handling
- **Timezone Support**: Configurable timezone settings (default: Asia/Seoul)
- **Persistent Storage**: Docker volume-based storage for whitelist data

## Commands

### User Commands

- `/start`: Start interaction with the bot and display month selection keyboard
- `/help`: Display available commands and admin contact information
- `/month [month]`: Show K-pop releases for the specified month (e.g., `/month april`)
- `/month [month] -gg`: Show releases only for female artists
- `/month [month] -mg`: Show releases only for male artists
- `/whitelists`: Display lists of female and male artists in a multi-column format

### Admin Commands

Whitelist management commands are only available to the user specified in `ADMIN_USERNAME` environment variable:

- `/add_artist [female/male] [artists]`: Add artists to the whitelist
- `/remove_artist [artists]`: Remove artists from the whitelist
- `/clearwhitelists`: Clear all whitelists
- `/clearcache`: Clear and reinitialize the cache
- `/export`: Export whitelists

## Prerequisites

- **Go**: Version 1.21 or higher (for development)
- **Docker**: For containerized deployment
- **Docker Compose**: For orchestrating the bot with persistent storage
- **Telegram Bot Token**: Obtain from BotFather in Telegram

## Running the Bot

### Using Docker Compose (Recommended)

1. Ensure you have a `docker-compose.yml` file in the project root:

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
      - WHITELIST_DIR=data
      - LOG_LEVEL=info
      - TZ=Asia/Seoul
    volumes:
      - whitelist_data:/app/data
volumes:
  whitelist_data:
    name: whitelist_data
```

### Environment Variables

- `BOT_TOKEN`: Telegram bot token
- `ADMIN_USERNAME`: Admin's Telegram username
- `CACHE_DURATION`: Duration to cache data (default: 24h)
- `MAX_RETRIES`: Maximum number of retries on failures (default: 3)
- `REQUEST_DELAY`: Delay between requests (default: 3s)
- `WHITELIST_DIR`: Directory for whitelist storage (default: data)
- `LOG_LEVEL`: Logging level (default: info)
- `TZ`: Timezone (default: Asia/Seoul)
- `MAX_CONCURRENT_REQUESTS`: Maximum concurrent requests (default: 5)

## Deployment

The bot is automatically built and published to Docker Hub (`tempizhere/gemfactory:latest`) via GitHub Actions on every push to the `main` branch. The CI/CD pipeline includes:

1. Automated testing
2. Code linting
3. Docker image building and publishing
4. Automated deployment support

To deploy on your server:

1. Use the provided `docker-compose.yml` (see above)
2. Deploy using a tool like **Dockge**:
   - Copy the `docker-compose.yml` into Dockge
   - Create a stack and deploy it
3. Ensure the `.env` file is present on the server with the correct environment variables

## Development

### Project Structure

- `cmd/bot/main.go`: Entry point with configuration loading and bot initialization
- `internal/`:
  - `telegrambot/`:
    - `bot/`: Core bot logic with command handling and keyboard management
    - `releases/`: Release management, caching, and scraping logic
  - `debounce/`: Rate limiting utilities
- `pkg/`:
  - `config/`: Configuration management
  - `log/`: Structured logging with zap
- `data/`: Whitelist JSON files storage

### Key Components

- **Bot Core**: Handles Telegram API interaction and command routing
- **Release Manager**: Manages K-pop release data and filtering
- **Cache System**:
  - Intelligent caching with periodic updates
  - Configurable cache duration
  - Concurrent update protection
- **Whitelist Manager**:
  - Separate female and male artist lists
  - JSON-based persistent storage
  - Thread-safe operations
- **Keyboard Manager**:
  - Dynamic month selection interface
  - Automatic monthly updates
  - Context-aware navigation
- **Error Handling**:
  - Comprehensive logging
  - Automatic retries
  - Rate limiting protection

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to your fork (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
