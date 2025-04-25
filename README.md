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
- `/remove_artist [female/male] [artists]`: Remove artists from the whitelist (admin only).
- `/clearwhitelists`: Clear all whitelists (admin only).

For whitelist-related questions, contact the admin (e.g., `@fullofsarang`).

## Prerequisites

- **Go**: Version 1.24 or higher (for development).
- **Docker**: For containerized deployment.
- **Docker Compose**: For orchestrating the bot with persistent storage.
- **Telegram Bot Token**: Obtain from BotFather in Telegram.
- **GitHub Account**: For accessing the repository and Docker Hub integration.
- **Docker Hub Account**: For pulling the pre-built image.

## Installation

### Clone the Repository

```bash
git clone https://github.com/tempizhere/gemfactory.git
cd gemfactory
```

### Set Up Environment Variables

Create a `.env` file in the project root with the following variables:

```env
BOT_TOKEN=your_telegram_bot_token
ADMIN_USERNAME=your_admin_username
CACHE_DURATION=8h
MAX_RETRIES=3
REQUEST_DELAY=10s
WHITELIST_DIR=data
LOG_LEVEL=info
```

Replace `your_telegram_bot_token` with your Telegram Bot Token and `your_admin_username` with the Telegram username of the admin (e.g., `fullofsarang`).

### Initialize Whitelists

Create the `data/` directory and add initial whitelist files:

```bash
mkdir -p data
```

**data/female_whitelist.json**:

```json
[]
```

**data/male_whitelist.json**:

```json
[]
```

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
    env_file:
      - .env
    volumes:
      - whitelist_data:/app/data
    networks:
      - gemfactory_network

volumes:
  whitelist_data:
    name: whitelist_data

networks:
  gemfactory_network:
    name: gemfactory_network
```
You can also use the environment section instead of env_file.

2. Start the bot:

```bash
docker-compose up -d
```

3. Check the container logs:

```bash
docker logs gemfactory
```

### Using Go (Development)

1. Install dependencies:

```bash
go mod tidy
```

2. Run the bot:

```bash
go run cmd/bot/main.go
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
- `internal/features/releasesbot/`: Core bot logic, including scraping, caching, and command handling.
- `pkg/`: Shared utilities (logging, configuration).
- `data/`: Directory for whitelist JSON files.

### Key Components

- **Scraper**: Fetches K-pop release schedules from external sources (e.g., `kpopofficial.com`).
- **Cache**: Stores release data to reduce API calls, updated every `CACHE_DURATION` (default: 8 hours).
- **Whitelists**: Managed via `female_whitelist.json` and `male_whitelist.json`, editable by admins.
- **Commands**: Handled via Telegram Bot API, with support for filtering and formatting.

### Building the Docker Image

```bash
docker build -t tempizhere/gemfactory:latest 
```

### Linting and Testing

Run linter:

```bash
golangci-lint run
```

Run tests (if available):

```bash
go test ./...
```

## Contributing

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/your-feature`).
3. Commit changes (`git commit -m "Add your feature"`).
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a Pull Request.
