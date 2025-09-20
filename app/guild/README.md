# Guild Module - Multi-Tenant Architecture

The Guild module is designed from the ground up to support multi-tenant Discord bot deployments where a single bot instance can serve multiple Discord servers (guilds) simultaneously.

## Architecture Overview

The guild module follows the same event-driven architecture (EDA) pattern as other modules (round, user) but with enhanced multi-tenant capabilities:

### Key Components

1. **Discord Event Handlers** (`app/guild/discord/events.go`)
   - Handles Discord guild lifecycle events (join, leave)
   - Automatically registers setup commands for new guilds
   - Publishes backend events for guild management

2. **Setup Manager** (`app/guild/discord/setup/`)
   - Manages guild configuration and setup flow
   - Uses `guild_id` from interaction context (not config)
   - Publishes events to backend for persistence

3. **Watermill Router** (`app/guild/watermill/router.go`)
   - Environment-specific queue groups for scaling
   - Guild ID propagation in all event flows
   - Multi-tenant message routing

4. **Event Handlers** (`app/guild/watermill/handlers/`)
   - Backend event response processing
   - Guild-scoped validation and processing
   - Multi-tenant context preservation

## Multi-Tenant Features

### Dynamic Command Registration
- **New Guilds**: Only `/frolf-setup` command is registered initially
- **Configured Guilds**: All commands available after setup completion
- **Global vs Guild-specific**: Supports both deployment modes

### Event-Driven Backend Communication
- All operations are event-driven (no direct DB access)
- Guild ID included in all event metadata
- Backend handles persistence and business logic

### Scalability
- **Gateway/Worker separation**: Discord connection isolated from business logic
- **Single Discord connection**: One gateway prevents Discord API conflicts
- **Horizontal worker scaling**: Add workers for increased processing capacity
- NATS queue groups for exclusive message processing
- Environment-specific queue naming
- Guild-scoped event routing

### Guild Lifecycle Management
- Automatic setup command registration on guild join
- Cleanup events on guild leave/role/channel deletion
- Backend-driven configuration management

## Configuration

### Multi-Tenant Deployment
```yaml
discord:
  guild_id: ""  # Empty for multi-tenant (global commands)
```

### Single-Guild Deployment  
```yaml
discord:
  guild_id: "123456789"  # Specific guild ID
```

## Multi-Pod Event Flow

### Gateway Pod Responsibilities
1. **Discord Connection**: Single WebSocket connection to Discord
2. **Command Registration**: Manages global slash command registration
3. **Event Publishing**: Forwards Discord events to NATS
4. **Response Handling**: Sends worker responses back to Discord

### Worker Pod Responsibilities  
1. **Event Processing**: Subscribes to Discord events from NATS
2. **Business Logic**: Handles guild setup, configuration management
3. **Backend Communication**: Database operations and API calls
4. **Response Generation**: Creates Discord responses via NATS

### Guild Setup Flow (Multi-Pod)
1. **Gateway**: Bot joins new guild → Publishes `guild.joined` to NATS
2. **Worker**: Processes join event → Registers setup command availability
3. **Gateway**: Admin runs `/frolf-setup` → Publishes interaction to NATS
4. **Worker**: Processes setup → Creates channels/roles → Publishes to backend
5. **Worker**: Backend confirms → Publishes success response to NATS
6. **Gateway**: Receives response → Sends confirmation embed to Discord

### Multi-Tenant Context Propagation
- All events include `guild_id` in metadata
- Guild ID extracted from Discord interaction context
- Backend handlers validate and scope operations by guild
- Response events maintain guild context

## Deployment Modes

### Development/Testing (Standalone Mode)
- Single pod deployment (default behavior)
- Set `discord.guild_id` to specific guild for testing
- Full bot functionality in one process
- Use: `BOT_MODE=""` or `BOT_MODE="standalone"`

### Production (Multi-Pod Architecture)

#### Gateway Mode (1 replica only)
- Handles Discord WebSocket connection exclusively
- Registers/manages slash commands globally
- Publishes Discord events to NATS
- **Critical**: Only ONE gateway pod per bot token
- Use: `BOT_MODE="gateway"`

#### Worker Mode (N replicas)
- Processes business logic and backend operations
- Subscribes to NATS events from gateway
- Handles database operations and API calls
- Horizontally scalable (multiple workers safe)
- Use: `BOT_MODE="worker"`

### Multi-Tenant Production Setup
```yaml
# Gateway deployment - ALWAYS 1 replica
discord-gateway:
  replicas: 1
  env:
    BOT_MODE: "gateway"
    DISCORD_TOKEN: "required"
    
# Worker deployment - scale as needed  
discord-workers:
  replicas: 3
  env:
    BOT_MODE: "worker"
    # No Discord token needed
```

## Completed Features

### ✅ Multi-Tenant Architecture
- **Event-Driven Design**: No direct database access from Discord bot
- **Guild Isolation**: All events properly scoped by `guild_id`
- **Runtime Configuration**: Dynamic guild config loading without restarts
- **Multi-Guild Support**: Unlimited Discord servers with proper isolation

### ✅ Dynamic Command Management
- **Setup-Based Registration**: Commands unlock automatically after guild setup
- **Global Command Support**: Bot can handle commands across all guilds
- **Automatic Cleanup**: Commands unregistered when guild is removed
- **Setup Flow**: New guilds get `/frolf-setup` command automatically

### ✅ Multi-Pod Architecture Framework
- **Three Deployment Modes**: Standalone, Gateway, and Worker modes defined
- **Mode Selection**: `BOT_MODE` environment variable controls deployment
- **Health Endpoints**: Proper health/ready checks for each mode
- **Graceful Fallback**: Gateway/Worker modes fall back to standalone if incomplete

### ✅ Event-Driven Communication
- **NATS Integration**: Reliable event processing with JetStream
- **Queue Groups**: Environment-specific processing isolation
- **Guild Context**: All events maintain guild_id for proper routing
- **Backend Separation**: Complete isolation between Discord and business logic

## Future Enhancements

1. **Complete Gateway Implementation**: Finish gateway-only mode (currently falls back to standalone)
2. **Complete Worker Implementation**: Finish worker-only mode (currently falls back to standalone)  
3. **Dead Letter Queues**: Add DLQ support for failed guild events
4. **Circuit Breakers**: Add resilience patterns for backend communication
5. **Idempotency**: Add idempotency keys for event deduplication
6. **Advanced Metrics**: Guild-specific metrics and monitoring dashboards
7. **Rate Limiting**: Per-guild rate limiting and quotas
8. **Auto-scaling**: Worker pod auto-scaling based on event queue depth
9. **Command Permissions**: Role-based command access control per guild
10. **Event Replay**: Ability to replay failed events for debugging

## Architecture Benefits

### Multi-Pod Architecture
- **Fault Isolation**: Discord connection issues don't affect business logic
- **Independent Scaling**: Scale Discord handling vs business logic separately  
- **Zero Downtime Deployments**: Update workers without affecting Discord connection
- **Resource Optimization**: Gateway pods lightweight, workers handle heavy processing
- **Development Flexibility**: Test business logic without Discord complexity

### Multi-Tenant Support
- **Guild Isolation**: Each guild's data and operations properly scoped
- **Horizontal Scaling**: Support unlimited Discord servers
- **Event-Driven**: No direct database dependencies in Discord layer
- **Backend Agnostic**: Discord bot works with any backend implementing events

The guild module serves as a reference implementation for multi-tenant Discord bot architecture using event-driven patterns.
