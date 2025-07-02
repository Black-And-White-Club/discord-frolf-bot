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

## Event Flow

### Guild Setup Flow
1. Bot joins new guild → Discord `GuildCreate` event
2. Setup command automatically registered for that guild
3. Admin runs `/frolf-setup` → Setup interaction handled
4. Setup event published to backend → Backend processes and stores config
5. Backend responds with success/failure → Discord module handles response
6. On success: Full command set could be unlocked (future enhancement)

### Multi-Tenant Context Propagation
- All events include `guild_id` in metadata
- Guild ID extracted from Discord interaction context
- Backend handlers validate and scope operations by guild
- Response events maintain guild context

## Deployment Modes

### Development/Testing (Single Guild)
- Set `discord.guild_id` to specific guild
- Commands registered per-guild
- Simplified testing and development

### Production (Multi-Tenant)
- Leave `discord.guild_id` empty
- Commands registered globally
- Bot can serve unlimited guilds
- Proper queue groups and scaling

## Future Enhancements

1. **Dead Letter Queues**: Add DLQ support for failed guild events
2. **Circuit Breakers**: Add resilience patterns for backend communication
3. **Idempotency**: Add idempotency keys for event deduplication
4. **Dynamic Command Management**: Unlock commands post-setup per guild
5. **Metrics**: Guild-specific metrics and monitoring
6. **Rate Limiting**: Per-guild rate limiting and quotas

The guild module serves as a reference implementation for multi-tenant Discord bot architecture using event-driven patterns.
