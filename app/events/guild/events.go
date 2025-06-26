package guild

import (
	"time"
)

// Topic constants for guild events
const (
	GuildSetupEventTopic        = "guild.setup"
	GuildConfigUpdateEventTopic = "guild.config.update"
	GuildRemovedEventTopic      = "guild.removed"
)

// GuildSetupEvent is published when a guild completes initial setup
type GuildSetupEvent struct {
	GuildID                string            `json:"guild_id"`
	GuildName              string            `json:"guild_name"`
	AdminUserID            string            `json:"admin_user_id"`
	EventChannelID         string            `json:"event_channel_id"`
	EventChannelName       string            `json:"event_channel_name"`
	LeaderboardChannelID   string            `json:"leaderboard_channel_id"`
	LeaderboardChannelName string            `json:"leaderboard_channel_name"`
	SignupChannelID        string            `json:"signup_channel_id"`
	SignupChannelName      string            `json:"signup_channel_name"`
	RoleMappings           map[string]string `json:"role_mappings"` // role_name -> role_id
	RegisteredRoleID       string            `json:"registered_role_id"`
	AdminRoleID            string            `json:"admin_role_id"`
	SetupCompletedAt       time.Time         `json:"setup_completed_at"`
}

// GuildConfigUpdateEvent is published when guild settings change
type GuildConfigUpdateEvent struct {
	GuildID     string      `json:"guild_id"`
	UpdatedBy   string      `json:"updated_by"`
	ConfigField string      `json:"config_field"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// GuildRemovedEvent is published when the bot is removed from a guild
type GuildRemovedEvent struct {
	GuildID   string    `json:"guild_id"`
	GuildName string    `json:"guild_name"`
	RemovedAt time.Time `json:"removed_at"`
}
