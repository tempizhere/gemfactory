# Gemfactory

Gemfactory is a Telegram bot designed to provide users with schedules of K-pop comebacks and releases for specified months. It fetches data from external sources, filters releases based on predefined whitelists of artists, and presents them in a user-friendly format. The bot is built with Go, containerized using Docker, and deployed via GitHub Actions to Docker Hub.

## Features

- **K-pop Release Schedules**: Retrieve upcoming K-pop comebacks and releases for a specific month.
- **Whitelist Filtering**: Filter releases by female (`-gg`) or male (`-mg`) artists using curated whitelists.
- **Dynamic Whitelists**: Admins can manage artist whitelists with commands like `/add_artist` and `/remove_artist`.
- **User-Friendly Output**: Formatted release lists and whitelists displayed in Telegram with HTML and monospaced fonts.
- **Caching**: Efficient caching of release data to reduce external API calls.
- **Dockerized Deployment**: Easy deployment with Docker and Docker Compose, supporting persistent storage for whitelists.

## Commands

- `/help`: Display available commands and contact info for whitelist inquiries.
- `/month [month]`: Show K-pop releases for the specified month (e.g., `/month april`).
- `/month [month] -gg`: Show releases only for female artists.
- `/month [month] -mg`: Show releases only for male artists.
- `/whitelists`: Display lists of female and male artists in a multi-column format.
- `/add_artist [female/male] [artists]`: Add artists to the female or male whitelist (admin only).
- `/remove_artist [artists]`: Remove artists from the whitelist (admin only).
- `/clearwhitelists`: Clear all whitelists (admin only).
- `/clearcache`: Clears the cache and reinitializes it (admin only).
- `/export`: Exports whitelists (admin only).

For whitelist-related questions, contact the admin (e.g., `@fullofsarang`).

## Prerequisites

- **Go**: Version 1.24 or higher (for development).
- **Docker**: For containerized deployment.
- **Docker Compose**: For orchestrating the bot with persistent storage.
- **Telegram Bot Token**: Obtain from BotFather in Telegram.

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
    volumes:
      - whitelist_data:/app/data
volumes:
  whitelist_data:
    name: whitelist_data
```


## Deployment

The bot is automatically built and published to Docker Hub (`tempizhere/gemfactory:latest`) via GitHub Actions on every push to the `main` branch. To deploy on your server:

1. Use the provided `docker-compose.yml` (see above).
2. Deploy using a tool like **Dockge**:
   - Copy the `docker-compose.yml` into Dockge.
   - Create a stack and deploy it.
3. Ensure the `.env` file is present on the server with the correct `BOT_TOKEN` and other variables.

## Development

### Project Structure

- `cmd/bot/main.go`: Entry point for the bot.
- `internal/telegrambot/`: Core bot logic, including scraping, caching, and command handling.
- `pkg/`: Shared utilities (logging, configuration).
- `data/`: Directory for whitelist JSON files.

### Key Components

- **Scraper**: Fetches K-pop release schedules from external sources (e.g., `kpopofficial.com`).
- **Cache**: Stores release data to reduce API calls, updated every `CACHE_DURATION` (default: 24 hours).
- **Whitelists**: Managed via `female_whitelist.json` and `male_whitelist.json`, editable by admins.

