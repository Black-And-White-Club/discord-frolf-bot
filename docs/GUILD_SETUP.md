# Guild Setup Implementation

This document describes the new guild setup functionality that allows the Discord Frolf Bot to be configured using a `/frolf-setup` slash command.

## Overview

The guild setup system provides a scalable foundation for the Frolf Bot that can start simple and evolve into a multi-tenant system. It consists of:

1. **Discord Setup Command** - `/frolf-setup` slash command that auto-creates channels and roles
2. **Backend Event Processing** - NATS-based event handling to persist configuration  
3. **Database Storage** - PostgreSQL storage for guild configurations
4. **Future-Proof Architecture** - Designed to support multi-guild deployments

## Architecture

```
Discord Bot → /frolf-setup → NATS Event → Backend Handler → PostgreSQL
```

### Components

- **`app/events/guild/`** - Event definitions for guild setup/config events
- **`app/guild/discord/setup/`** - Discord slash command handler
- **`app/guild/watermill/handlers/`** - Backend event processors  
- **`app/guild/storage/`** - Database service for guild configs
- **`app/guild/`** - Module integration and setup

## Current Status

✅ **Implemented:**
- Event structures for guild setup/updates/removal
- Discord setup command handler with auto-creation of channels/roles
- Backend handlers for processing setup events
- Database service with PostgreSQL support
- SQL migration for guild_configs table
- Module integration pattern following existing code structure

⏳ **Pending Integration:**
- Database connection setup in main application
- Full end-to-end testing
- Error handling and validation improvements

## How It Works

### 1. Setup Command (`/frolf-setup`)

When an admin runs `/frolf-setup` in Discord:

1. **Permission Check** - Verifies user has Administrator permissions
2. **Auto-Setup** - Creates/finds required channels and roles:
   - `#frolf-events` - For round management
   - `#frolf-leaderboard` - For rankings and results  
   - `#frolf-signup` - For player registration
   - `@Frolf Player` role - For participants
   - `@Frolf Admin` role - For administrators
3. **Event Publishing** - Sends `GuildSetupEvent` to NATS
4. **User Response** - Shows success message with created resources

### 2. Backend Processing

The backend handler receives the `GuildSetupEvent` and:

1. **Converts** event data to database format
2. **Saves** guild configuration to PostgreSQL
3. **Logs** successful completion

### 3. Database Schema

The `guild_configs` table stores:
- Guild ID and name
- Channel IDs and names for events, leaderboard, signup
- Role mappings and IDs  
- Timestamps for created/updated

## Installation & Setup

### Basic Setup (No Database)

The bot works without database connectivity. The setup command will be available but won't persist data:

```yaml
# config.yaml
discord:
  token: "${DISCORD_TOKEN}"
  app_id: "${DISCORD_APP_ID}" 
  primary_guild_id: "${DISCORD_GUILD_ID}"
```

### Full Setup (With Database)

1. **Set up PostgreSQL database:**
   ```bash
   createdb frolf_bot
   ```

2. **Run migrations:**
   ```bash
   psql frolf_bot < migrations/002_guild_configs.sql
   ```

3. **Configure database connection:**
   ```yaml
   # config.yaml
   infrastructure:
     mode: "single-server"
     database_url: "postgres://user:pass@localhost:5432/frolf_bot"
   ```

4. **Uncomment guild module initialization in `app/bot/bot.go`:**
   ```go
   // Remove /* and */ around the guild module code
   ```

### Environment Variables

```bash
export DISCORD_TOKEN="your_bot_token"
export DISCORD_APP_ID="your_app_id"  
export DISCORD_GUILD_ID="your_guild_id"
export DATABASE_URL="postgres://user:pass@localhost:5432/frolf_bot"  # Optional
```

## Usage

1. **Invite bot to your Discord server** with Administrator permissions
2. **Run setup command** in any channel: `/frolf-setup`
3. **Bot responds** with created channels and roles
4. **Configuration persisted** to database (if configured)

## Future Enhancements

This implementation provides a foundation for:

- **Multi-Guild Support** - Add guild_id indexing and per-guild isolation
- **Advanced Permissions** - Role-based setup restrictions  
- **Configuration Updates** - Commands to modify existing setup
- **Premium Features** - Tier-based functionality
- **Backup/Restore** - Export/import guild configurations
- **Web Dashboard** - GUI for configuration management

## Error Handling

The system includes:
- Permission validation before setup
- Database transaction safety
- Graceful fallbacks when resources already exist
- Comprehensive logging for troubleshooting

## Testing

To test the setup flow:

1. Ensure bot has Administrator permissions in your test server
2. Run `/frolf-setup` in any channel
3. Verify channels and roles are created
4. Check database for saved configuration (if DB enabled)
5. Test that existing resources are reused on repeat runs
