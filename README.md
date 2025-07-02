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

2. **Run Container** (Standalone Mode):
   ```bash
   docker run -e DISCORD_TOKEN=your_token \
              -p 8080:8080 \
              discord-frolf-bot:latest
   ```

3. **Multi-Pod Production Deployment**:
   ```bash
   # Gateway pod (exactly 1 replica)
   docker run -e DISCORD_TOKEN=your_token \
              -e BOT_MODE=gateway \
              -e NATS_URL=nats://nats:4222 \
              -p 8080:8080 \
              discord-frolf-bot:latest

   # Worker pods (scale as needed)
   docker run -e BOT_MODE=worker \
              -e NATS_URL=nats://nats:4222 \
              -p 8081:8080 \
              discord-frolf-bot:latest
   ```

## Configuration

### Environment Variables

The following environment variables override config file values:

- `DISCORD_TOKEN` - Discord bot token (required)
- `DISCORD_GUILD_ID` - Primary Discord guild/server ID (optional, for single-guild mode)
- `NATS_URL` - NATS server URL (default: nats://localhost:4222)
- `BOT_MODE` - Deployment mode: `standalone`, `gateway`, or `worker` (default: `standalone`)
- `METRICS_ADDRESS` - Metrics server address (default: :8080)
- `LOKI_URL` - Loki logging endpoint (optional)
- `ENVIRONMENT` - Environment name for queue group isolation (default: development)

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
   - Create required roles (User, Editor, Admin)
   - Set up channel permissions
   - Create signup reaction message
   - **Update your config.yaml with all the generated IDs**

No manual config editing required! ✅

## Production Deployment

### Multi-Pod Architecture

The bot supports three deployment modes via the `BOT_MODE` environment variable:

1. **Standalone Mode** (Default):
   ```bash
   # Single pod handling both Discord interactions and backend processing
   docker run -e DISCORD_TOKEN=your_token discord-frolf-bot:latest
   ```

2. **Gateway Mode** (Production):
   ```bash
   # Single pod handling Discord interactions only
   docker run -e DISCORD_TOKEN=your_token \
              -e BOT_MODE=gateway \
              discord-frolf-bot:latest
   ```

3. **Worker Mode** (Production):
   ```bash
   # Multiple pods handling backend event processing
   docker run -e DISCORD_TOKEN=your_token \
              -e BOT_MODE=worker \
              discord-frolf-bot:latest
   ```

### Health Checks

The bot includes health check endpoints (port 8080) for container orchestration:
- `GET /health` - Application health status
- `GET /ready` - Readiness check for load balancers

### Kubernetes Deployment

For production deployments, use the gateway/worker pattern:

```yaml
# Gateway deployment (exactly 1 replica)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-frolf-bot-gateway
spec:
  replicas: 1  # MUST be 1 - Discord allows only one connection per bot
  template:
    spec:
      containers:
      - name: gateway
        image: discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "gateway"
        - name: DISCORD_TOKEN
          valueFrom:
            secretKeyRef:
              name: discord-secrets
              key: token
---
# Worker deployment (scale as needed)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: discord-frolf-bot-worker
spec:
  replicas: 3  # Scale based on load
  template:
    spec:
      containers:
      - name: worker
        image: discord-frolf-bot:latest
        env:
        - name: BOT_MODE
          value: "worker"
```

### CI/CD

GitHub Actions workflow automatically:
- Builds and tests on every push
- Creates Docker images tagged with Git SHA
- Pushes to GitHub Container Registry
- Ready for GitOps deployment

## Architecture

The Discord Frolf Bot is built using a modern, multi-tenant event-driven architecture (EDA) designed for production scale and reliability.

### Multi-Pod Deployment Modes

The bot supports three deployment modes optimized for different scenarios:

**Standalone Mode** (Development/Small deployments):
- Single process handles both Discord interactions and backend processing
- Simplest deployment, good for development and testing
- Set `BOT_MODE=standalone` or leave unset (default)

**Gateway Mode** (Production - Discord Interface):
- Dedicated pod handling only Discord interactions and event publishing
- **MUST run exactly 1 replica** (Discord allows only one bot connection)
- Handles slash commands, user interactions, and Discord events
- Set `BOT_MODE=gateway`

**Worker Mode** (Production - Backend Processing):
- Dedicated pods handling event processing and business logic
- **Can scale horizontally** (multiple replicas supported)
- Processes guild setup, scoring, leaderboards, and database operations
- Set `BOT_MODE=worker`

### Multi-Tenant Support

The bot supports unlimited Discord servers (guilds) simultaneously:

- **Guild Isolation**: All events include `guild_id` for proper tenant separation
- **Dynamic Command Registration**: Commands auto-register per guild based on setup status
- **Runtime Configuration**: Guild configs loaded dynamically without restarts
- **Event-driven Setup**: New guilds trigger automated setup flows

### Event-Driven Architecture

- **No Direct Database Access**: Discord bot only handles interactions and event publishing
- **NATS/JetStream Integration**: Reliable message processing with Watermill
- **Guild-scoped Events**: All events properly isolated by `guild_id`
- **Queue Groups**: Environment-specific processing with exclusive message handling
- **Backend Separation**: Business logic completely isolated from Discord interactions

### Modules

- **Guild Module**: Multi-tenant guild management, setup, and lifecycle
- **Round Module**: Event creation, RSVP management, and scheduling
- **User Module**: User registration, role management, and profiles
- **Score Module**: Score tracking and leaderboard management
- **Leaderboard Module**: Tag claiming and ranking systems

Each module follows the same EDA patterns with:
- Discord interaction handlers
- Event publishers/subscribers
- Watermill routers and handlers
- Multi-tenant context propagation

## Commands

### Setup Commands
- `go run main.go setup <guild_id>` - Automated server setup and configuration
- `make setup GUILD_ID=<guild_id>` - Setup using Makefile

### Bot Commands (Discord)
- `/frolf-setup` - Initial guild setup (auto-registered for new guilds)
- `/create-round` - Create new frolf event
- `/score-round` - Submit scores for completed round
- `/leaderboard` - View current standings
- `/claim-tag` - Claim leaderboard position tags
- And many more... (commands auto-register after guild setup)

### Development Commands
- `make test-all` - Run all tests with summary
- `make build` - Build the application
- `make run` - Run in development mode
- `BOT_MODE=gateway make run` - Run in gateway mode
- `BOT_MODE=worker make run` - Run in worker mode

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
├── bot/              # Core bot logic and mode management
├── discordgo/        # Discord interaction handlers
├── events/           # Event definitions and schemas
├── guild/            # Multi-tenant guild management
│   ├── discord/      # Discord event handlers and setup
│   └── watermill/    # Event processing and handlers
├── health/           # Health check endpoints
├── leaderboard/      # Leaderboard and tag management
├── round/            # Round/event management
├── score/            # Score tracking and validation
├── shared/           # Shared utilities and storage
└── user/             # User management and roles
config/               # Configuration management
cmd/                  # Command-line utilities
migrations/           # Database migrations
scripts/              # Build and deployment scripts
```

### Module Architecture

Each module follows the same EDA pattern:
- **Discord handlers**: Handle Discord interactions and publish events
- **Watermill routers**: Route events to appropriate handlers
- **Event handlers**: Process events and manage business logic
- **Storage interfaces**: Abstract data persistence
- **Mock implementations**: Enable comprehensive testing

## Gateway/Worker Implementation Guide

The framework for multi-pod deployment exists but needs completion. This section documents what needs to be implemented for production gateway/worker separation.

### Current State

✅ **Framework Complete**:
- `BOT_MODE` environment variable support
- Health endpoints for each mode
- Graceful fallback to standalone mode
- Event-driven architecture with NATS

🚧 **Implementation Needed**:
- Gateway-only logic (currently falls back to standalone)
- Worker-only logic (currently falls back to standalone)

### Implementation Requirements

#### 1. Gateway Mode (`main.go:runGatewayMode`)

**Purpose**: Handle Discord WebSocket connection and forward events to workers

**Required Implementation**:
```go
func runGatewayMode(ctx context.Context) {
    // ✅ Config and observability already implemented
    
    // 🚧 TODO: Discord session (WebSocket only)
    session, err := discordgo.New("Bot " + cfg.Discord.Token)
    
    // 🚧 TODO: NATS publisher setup
    publisher := setupNATSPublisher(cfg.NATS.URL)
    
    // 🚧 TODO: Discord event handlers → NATS publishers
    session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
        publishInteractionToNATS(publisher, i) // NO local processing
    })
    
    // 🚧 TODO: NATS subscriber for worker responses
    subscriber.Subscribe("discord.response.*", handleWorkerResponse)
    
    // 🚧 TODO: Open Discord connection
    session.Open()
}
```

#### 2. Worker Mode (`main.go:runWorkerMode`)

**Purpose**: Process business logic from NATS events (no Discord connection)

**Required Implementation**:
```go
func runWorkerMode(ctx context.Context) {
    // ✅ Config and observability already implemented
    
    // 🚧 TODO: NATS subscriber (NO Discord session)
    subscriber := setupNATSSubscriber(cfg.NATS.URL)
    
    // 🚧 TODO: Initialize business logic modules
    // Extract from existing standalone mode:
    // - User module initialization
    // - Round module initialization  
    // - Guild module initialization
    // - All Watermill routers
    
    // 🚧 TODO: Subscribe to gateway events
    subscriber.Subscribe("discord.interaction.*", processInteraction)
    
    // 🚧 TODO: Start all business logic routers
}
```

#### 3. Event Schema (New File: `app/events/gateway/`)

**Required Events**:
```go
// Gateway → Workers
type DiscordInteractionEvent struct {
    GuildID     string                      `json:"guild_id"`
    Interaction *discordgo.InteractionCreate `json:"interaction"`
    Timestamp   time.Time                   `json:"timestamp"`
}

// Workers → Gateway  
type DiscordResponseEvent struct {
    GuildID       string                         `json:"guild_id"`
    InteractionID string                         `json:"interaction_id"`
    Response      *discordgo.InteractionResponse `json:"response"`
    Timestamp     time.Time                     `json:"timestamp"`
}
```

#### 4. NATS Topics

**Topic Structure**:
- `discord.interaction.{guild_id}` - Gateway publishes interactions
- `discord.response.{guild_id}` - Workers publish responses
- `discord.guild.joined.{guild_id}` - Gateway publishes guild events
- `discord.guild.left.{guild_id}` - Gateway publishes guild events

#### 5. Files to Modify/Create

**Existing Files**:
- `main.go` - Complete gateway/worker functions
- `app/bot/bot.go` - Extract business logic for worker mode

**New Files Needed**:
- `app/gateway/gateway.go` - Gateway-specific Discord handling
- `app/events/gateway/events.go` - Gateway/worker event schemas
- `app/gateway/publisher.go` - NATS event publishing
- `app/worker/subscriber.go` - NATS event consumption

#### 6. Implementation Benefits

**When Complete**:
- True zero-downtime deployments (update workers without Discord disconnect)
- Independent scaling (scale business logic separately from Discord handling)
- Fault isolation (business logic crashes don't affect Discord connection)
- Resource optimization (lightweight gateway, heavy-processing workers)

**Scaling Pattern**:
```yaml
# Production deployment
gateway:   replicas: 1    # MUST be exactly 1
workers:   replicas: 3-N  # Scale based on processing load
```

### Implementation Priority

1. **Phase 1**: Create event schemas and NATS topic structure
2. **Phase 2**: Extract business logic from standalone mode for worker mode
3. **Phase 3**: Implement gateway-only Discord handling
4. **Phase 4**: Test full gateway ↔ worker communication
5. **Phase 5**: Update deployment documentation and examples

This implementation maintains backward compatibility - standalone mode continues working while gateway/worker modes are developed.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

## License

This project is licensed under the MIT License.
