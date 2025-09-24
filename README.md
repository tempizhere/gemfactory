# gemfactory

Telegram bot for tracking K-pop releases and homework assignments. Fetches data from external sources, filters by artist lists, and presents in a user-friendly format.

## Features

- **K-pop Release Schedules**: Get releases for specified months
- **Gender Filtering**: Show releases only for female (`-f`) or male (`-m`) artists
- **Homework Assignments**: Random tracks from Spotify playlist for listening
- **LLM Parsing**: AI-powered data extraction from websites
- **Automatic Updates**: Task scheduler for playlist updates

## Commands

### User Commands

- `/start` - Start interaction with the bot
- `/help` - Show available commands
- `/month [month]` - Show releases for month (e.g., `/month april`)
- `/month [month] -f` - Female artists only
- `/month [month] -m` - Male artists only
- `/search [artist]` - Search releases by artist
- `/artists` - Show active artists lists
- `/homework` - Get homework assignment
- `/playlist` - Playlist information

### Admin Commands

- `/add_artist [name] [-f|-m]` - Add artist to list
- `/remove_artist [name]` - Remove artist from list
- `/config [key] [value]` - Set configuration
- `/config_list` - Show configuration
- `/config_reset` - Reset configuration
- `/tasks_list` - Show task list
- `/reload_playlist` - Reload playlist
- `/parse_releases [month/year]` - Parse releases for specific month/year
- `/export` - Export all artists

### Environment Variables

Copy `env.example` to `.env` and fill in:

```bash
# Required
DB_DSN=postgres://username:password@host:port/gemfactory?sslmode=disable
BOT_TOKEN=your_telegram_bot_token
ADMIN_USERNAME=your_telegram_username

# Optional
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret
PLAYLIST_URL=https://open.spotify.com/playlist/your_playlist_id
LLM_API_KEY=your_llm_api_key
LLM_BASE_URL=https://integrate.api.nvidia.com/v1
```

## Architecture

- **BUN ORM** - PostgreSQL database operations
- **LLM Integration** - AI-powered data parsing
- **Task Scheduler** - Automatic updates
- **Spotify API** - Playlist integration
- **Telegram Bot API** - User interface

## Project Structure

```
gemfactory/
├── cmd/bot/                 # Application entry point
├── internal/
│   ├── config/             # Configuration
│   ├── model/              # Data models
│   ├── service/            # Business logic
│   ├── storage/            # Database layer
│   ├── handlers/           # Command handlers
│   ├── external/           # External APIs
│   └── app/                # Component factory
├── migrations/             # Database migrations
└── env.example            # Configuration example
```

## License

MIT License
