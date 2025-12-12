# Slash Command Registration (Multi-Tenant)

This document explains **when** and **where** the bot registers Discord slash commands, and how new commands roll out to existing servers after deployments.

## Goals

- Before a server is configured, **only** `/frolf-setup` should be visible.
- After setup completes, all guild-scoped commands (e.g. `/createround`, `/claimtag`, `/set-udisc-name`, …) should be visible.
- When new commands are added in code and the bot is redeployed, existing setup-complete guilds should automatically receive the new commands **without re-running setup**.

## Key idea: global vs guild commands

Discord has two relevant “scopes” for application commands:

- **Global commands**: registered with an empty guild ID.
  - Pros: available everywhere.
  - Cons: can take longer to propagate.
- **Guild commands**: registered against a specific guild ID.
  - Pros: show up quickly.
  - Cons: must be registered per guild.

This bot uses:

- `/frolf-setup` as a **global** command.
- All other commands as **guild** commands, gated on “setup complete”.

## Where commands are defined

Commands are registered by:

- `app/discordgo/commands.go` → `RegisterCommands(session, logger, guildID)`

### Idempotent registration

`RegisterCommands` first lists the currently registered commands:

- `ApplicationCommands(appID, guildID)`

…and then **creates only the commands that are missing**. This makes registration safe to call:

- at startup
- after setup
- repeatedly across deploys

## When commands are registered

### 1) Deploy/startup

On startup the bot:

1. Registers global commands:
   - `discord.RegisterCommands(bot.Session, bot.Logger, "")`
   - In multi-tenant mode, this registers **only** `/frolf-setup` globally.
2. When Discord fires the Ready event, the bot runs a one-time reconciliation:
   - `(*DiscordBot).syncGuildCommands(r.Guilds)`

The reconciliation intentionally **skips** guilds that are not setup-complete:

- `GuildConfigResolver.IsGuildSetupComplete(guildID)`

This preserves the intended UX: pre-setup guilds only see `/frolf-setup`.

Code:

- `app/bot/bot.go` → Ready handler + `syncGuildCommands`

### 2) First-time setup completion

When a guild completes setup successfully, the bot receives a guild-config success event:

- `guildevents.GuildConfigCreated`

…and then registers all guild commands for that guild:

- `GuildDiscord.RegisterAllCommands(guildID)`

Code:

- `app/guild/watermill/handlers/guild_config_creation_handler.go` → `HandleGuildConfigCreated`
- `app/guild/discord/discord.go` → `RegisterAllCommands`
- `app/guild/watermill/router.go` → wires `GuildConfigCreated` → `HandleGuildConfigCreated`

If setup fails, command registration does _not_ occur (guild remains setup-only):

- `HandleGuildConfigCreationFailed`

## Why this fixes “new commands don’t appear”

Discord will not automatically add your newly-defined commands to existing guilds. If your bot only registered commands during the initial setup, then a newly deployed command (like `/set-udisc-name`) won’t appear until something triggers registration again.

With this design:

- On **deploy/startup**: setup-complete guilds get any missing commands via reconciliation.
- On **setup completion**: the guild immediately gets all guild commands.

## Operational notes

- **Discord propagation**: global commands can take longer to appear than guild commands.
- **Rate limiting**: the Ready reconciliation sleeps briefly between guild registrations to reduce rate-limit risk.
- **Multi-pod**: only a single “gateway” instance should maintain the Discord websocket + register commands. If multiple replicas connect to Discord, you can get duplicate registration attempts and/or gateway disconnects.
