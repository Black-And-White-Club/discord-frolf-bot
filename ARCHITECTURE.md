# Discord Frolf Bot - Architecture

## Overview

The Discord Frolf Bot is designed as a **single-server bot** with a **file-based configuration system** and **automatic setup capabilities**. The architecture is built to be simple and maintainable while remaining extensible for future multi-tenant deployment.

## Current Architecture (Single-Server Mode)

### Core Design Principles

1. **File-based Configuration** - No database dependency for core functionality
2. **Automatic Setup** - Command-line setup with auto-config updates
3. **Simple Deployment** - Single container, minimal dependencies
4. **Future-proof** - Ready for multi-tenant expansion without rewrites

### Configuration Structure

The bot uses a file-based configuration with automatic updates:

```yaml
discord:
  # Core credentials
  token: ""              # Set via DISCORD_TOKEN env var
  app_id: ""             # Set via DISCORD_APP_ID env var
  guild_id: ""           # Auto-populated by setup command
  
  # Auto-populated by setup command
  signup_channel_id: ""
  event_channel_id: ""
  leaderboard_channel_id: ""
  signup_message_id: ""
  registered_role_id: ""
  admin_role_id: ""
  role_mappings: {}
  
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

### Setup Process

The bot includes an automated setup process:

```bash
# Run setup for your Discord server
go run main.go setup YOUR_GUILD_ID
```

This command:
1. Connects to your Discord server
2. Creates required channels (signup, events, leaderboard)
3. Creates required roles (User, Editor, Admin)
4. Sets up channel permissions
5. Creates signup reaction message
6. **Automatically updates your config.yaml** with all generated IDs

### Environment Variables

Required environment variables:

```bash
# Required
DISCORD_TOKEN=your_bot_token
DISCORD_APP_ID=your_app_id

# Optional (with defaults)
NATS_URL=nats://localhost:4222
ENVIRONMENT=development
LOKI_URL=http://loki:3100
METRICS_ADDRESS=:8080
```

### Code Design

All existing code works through convenience methods:
- `config.GetGuildID()` → returns guild ID
- `config.GetEventChannelID()` → returns event channel ID
- `config.GetSignupChannelID()` → returns signup channel ID
- etc.

This abstraction allows for future database-backed configuration without code changes.

## Future Expansion: Multi-Tenant Support

The codebase is designed to support future multi-tenant deployment with minimal changes:

### Migration Path

To switch to multi-tenant mode in the future:

1. **Add database integration:**
   - Use the existing `database/schema.sql` and `migrations/` files
   - Implement database-backed config loading
   - Add uptrace/bun ORM integration

2. **Update deployment:**
   - Add database connection via `DATABASE_URL`
   - Implement guild onboarding flow
   - Add per-guild observability

### Multi-Tenant Benefits (Future)

**Per-Guild Configuration:**
- Stored in database, not config files
- Auto-discovery when bot joins new servers
- Subscription tier management
- Feature flag toggles

**Scalability:**
- Horizontal scaling with multiple bot instances
- Per-guild metrics and logs
- Usage analytics for potential premium features

## Current Architecture Benefits

### Single-Server Mode (Current)
- **Zero database dependency** - everything in config files
- **Simple deployment** - single container, minimal resources
- **Fast startup** - no database queries needed
- **Easy development** - local testing with file-based config
- **Automatic configuration** - setup command updates config automatically

### Multi-Tenant Ready (Future)
- **Horizontal scaling** - multiple bot instances
- **Per-guild isolation** - configurations stored separately
- **Database integration** - PostgreSQL with uptrace/bun ORM
- **Auto-onboarding** - automatic setup when joining new servers

## Development Workflow

### Current (Single-Server)
1. Set `DISCORD_TOKEN` and `DISCORD_APP_ID` environment variables
2. Run `go run main.go setup <guild_id>` to configure your Discord server
3. Config is automatically updated with all Discord IDs
4. Deploy single container - bot serves your Discord server

### Future (Multi-Tenant)
1. Add database connection via `DATABASE_URL`
2. Implement database-backed config loading
3. Bot automatically discovers/creates resources on guild join
4. Per-guild config stored in database
5. Bot serves multiple Discord servers simultaneously

## Infrastructure

### Current Files
- ✅ **`Dockerfile`** - Container build configuration
- ✅ **`app/health/`** - Health check endpoints (available but not integrated)
- ✅ **`config.yaml`** - Auto-updated by setup command
- ✅ **`main.go`** - Includes setup command and config auto-update

### Future Files (Available)
- 📁 **`database/schema.sql`** - Multi-tenant database structure
- 📁 **`migrations/`** - Database migration files

### Removed/Not Needed
- ❌ **k8s files** - Removed, not using Kubernetes
- ❌ **Multi-tenant config** - Available but not currently used

## Next Steps

### Immediate (Current Workflow)
1. ✅ File-based configuration system implemented
2. ✅ Automatic setup command with config updates
3. ✅ Single-server deployment ready
4. 🎯 Deploy to your container environment
5. 🎯 Test automated setup process

### Future Expansion (Optional)
1. Integrate health endpoints into main application
2. Add database integration with uptrace/bun ORM
3. Implement guild onboarding flow for multi-tenant support
4. Add per-guild observability and metrics
5. Deploy as public Discord bot (if desired)

The architecture is now optimized for your current single-server use case while remaining ready for future expansion.
