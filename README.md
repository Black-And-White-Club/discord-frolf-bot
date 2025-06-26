# Discord Frolf Bot

A production-ready Discord bot for managing frolf (frisbee golf) events, scoring, and community interaction. Designed for simple deployment with Docker containers.

## Features

- **Automated Guild Setup**: Command-line setup that auto-configures your Discord server
- **Event Management**: Create and manage frolf events with RSVP functionality
- **Score Tracking**: Track scores and maintain leaderboards
- **Role Management**: Automatic role assignment based on performance
- **File-based Configuration**: Simple YAML config with auto-updates
- **Container Ready**: Dockerfile included for easy deployment

## Quick Start

### Local Development

1. **Clone and Setup**:
   ```bash
   git clone https://github.com/your-org/discord-frolf-bot.git
   cd discord-frolf-bot
   go mod tidy
   ```

2. **Configuration**:
   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your Discord bot token
   ```

3. **Guild Setup** (Auto-configures your Discord server):
   ```bash
   go run main.go setup YOUR_GUILD_ID
   # This will automatically update your config.yaml
   ```

4. **Run Locally**:
   ```bash
   go run main.go
   ```

### Container Deployment

1. **Build Container**:
   ```bash
   docker build -t discord-frolf-bot:latest .
   ```

2. **Run Container**:
   ```bash
   docker run -e DISCORD_TOKEN=your_token \
              -e DISCORD_GUILD_ID=your_guild_id \
              -p 8080:8080 \
              discord-frolf-bot:latest
   ```

## Configuration

### Environment Variables

The following environment variables override config file values:

- `DISCORD_TOKEN` - Discord bot token (required)
- `DISCORD_GUILD_ID` - Primary Discord guild/server ID (required for setup)
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)

### Config File

Copy `config.example.yaml` to `config.yaml` and customize:

```yaml
discord:
  token: "YOUR_BOT_TOKEN"
  guild_id: ""  # Auto-populated by setup command
  app_id: "YOUR_APP_ID"
  # Channel/role IDs auto-populated by setup
  signup_channel_id: ""
  event_channel_id: ""
  leaderboard_channel_id: ""
  # ... other settings

# Service configuration
service:
  name: "discord-frolf-bot"
  version: "1.0.0"

# Observability
observability:
  environment: "development"
  loki_url: ""
  metrics_address: ":8080"
```

## Guild Setup

Run the setup command to automatically configure your Discord server:

1. Invite the bot to your server with admin permissions
2. Run setup from command line:
   ```bash
   go run main.go setup YOUR_GUILD_ID
   ```
3. The setup will automatically:
   - Create required channels (signup, events, leaderboard)
   - Create required roles (Rattler, Editor, Admin)
   - Set up channel permissions
   - Create signup reaction message
   - **Update your config.yaml with all the generated IDs**

No manual config editing required! ✅

## Production Deployment

### Health Checks

The bot includes health check endpoints (port 8080) for container orchestration:
- `GET /health` - Application health status
- `GET /ready` - Readiness check for load balancers

*Note: Health endpoints are available in code but not currently integrated into main application.*

### Container Deployment

The bot is designed for simple container deployment:

```bash
# Build and run with Docker
docker build -t discord-frolf-bot:latest .
docker run -e DISCORD_TOKEN=your_token discord-frolf-bot:latest
```

### CI/CD

GitHub Actions workflow automatically:
- Builds and tests on every push
- Creates Docker images tagged with Git SHA
- Pushes to GitHub Container Registry
- Ready for GitOps deployment

## Architecture

### Current Mode: Single-Server

- **File-based Configuration**: All settings stored in `config.yaml`
- **Auto-setup**: Command-line setup automatically configures Discord and updates config
- **No Database Required**: Everything runs from file-based configuration
- **Simple Deployment**: Single container with minimal dependencies

### Future Expansion: Multi-Tenant Ready

The codebase is designed to support future multi-tenant deployment:
- Database schema available for guild configurations
- Convenience methods abstract config access
- Ready for horizontal scaling when needed

### Components

- **Discord Bot**: Core bot functionality and command handling
- **Event System**: NATS-based event bus for decoupled components
- **Storage**: In-memory interaction state management
- **Observability**: Structured logging, metrics, and tracing
- **Health**: HTTP endpoints available for container health checks (not integrated)

## Commands

- `go run main.go setup <guild_id>` - Automated server setup and configuration
- `/create-round` - Create new frolf event
- `/score-round` - Submit scores for completed round
- `/leaderboard` - View current standings
- And many more...

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/frolf-bot .
```

### Project Structure

```
app/
├── bot/           # Core bot logic
├── discordgo/     # Discord integration
├── events/        # Event handlers
├── health/        # Health check endpoints
├── leaderboard/   # Leaderboard management
├── round/         # Round/event management
├── score/         # Score tracking
└── user/          # User management
config/            # Configuration management
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

This project is licensed under the MIT License.
