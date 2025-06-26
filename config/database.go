package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// DatabaseConfig represents the configuration stored in the database
type DatabaseConfig struct {
	GuildID                string            `db:"guild_id"`
	GuildName              string            `db:"guild_name"`
	EventChannelID         string            `db:"event_channel_id"`
	EventChannelName       string            `db:"event_channel_name"`
	LeaderboardChannelID   string            `db:"leaderboard_channel_id"`
	LeaderboardChannelName string            `db:"leaderboard_channel_name"`
	SignupChannelID        string            `db:"signup_channel_id"`
	SignupChannelName      string            `db:"signup_channel_name"`
	RoleMappings           map[string]string `db:"role_mappings"`
	RegisteredRoleID       string            `db:"registered_role_id"`
	AdminRoleID            string            `db:"admin_role_id"`
}

// LoadConfigFromDatabase loads configuration from PostgreSQL database
func LoadConfigFromDatabase(ctx context.Context, databaseURL string, guildID string) (*Config, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required for database-backed config")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	var dbConfig DatabaseConfig
	var roleMappingsJSON []byte

	query := `
		SELECT guild_id, guild_name, event_channel_id, event_channel_name,
		       leaderboard_channel_id, leaderboard_channel_name,
		       signup_channel_id, signup_channel_name, role_mappings,
		       registered_role_id, admin_role_id
		FROM guild_configs 
		WHERE guild_id = $1`

	err = db.QueryRowContext(ctx, query, guildID).Scan(
		&dbConfig.GuildID,
		&dbConfig.GuildName,
		&dbConfig.EventChannelID,
		&dbConfig.EventChannelName,
		&dbConfig.LeaderboardChannelID,
		&dbConfig.LeaderboardChannelName,
		&dbConfig.SignupChannelID,
		&dbConfig.SignupChannelName,
		&roleMappingsJSON,
		&dbConfig.RegisteredRoleID,
		&dbConfig.AdminRoleID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no configuration found for guild ID %s", guildID)
		}
		return nil, fmt.Errorf("failed to query guild config: %w", err)
	}

	// Parse role mappings JSON
	if err := json.Unmarshal(roleMappingsJSON, &dbConfig.RoleMappings); err != nil {
		return nil, fmt.Errorf("failed to parse role mappings: %w", err)
	}

	// Convert database config to application config
	config := &Config{
		Discord: DiscordConfig{
			Token:                getEnvOrDefault("DISCORD_TOKEN", ""),
			AppID:                getEnvOrDefault("DISCORD_APP_ID", ""),
			GuildID:              dbConfig.GuildID,
			SignupChannelID:      dbConfig.SignupChannelID,
			EventChannelID:       dbConfig.EventChannelID,
			LeaderboardChannelID: dbConfig.LeaderboardChannelID,
			RegisteredRoleID:     dbConfig.RegisteredRoleID,
			AdminRoleID:          dbConfig.AdminRoleID,
			RoleMappings:         dbConfig.RoleMappings,
			SignupEmoji:          "🐍",
		},
		Service: ServiceConfig{
			Name:    "discord-frolf-bot",
			Version: "1.0.0",
		},
		Observability: ObservabilityConfig{
			Environment:     getEnvOrDefault("ENVIRONMENT", "production"),
			LokiURL:         getEnvOrDefault("LOKI_URL", ""),
			MetricsAddress:  getEnvOrDefault("METRICS_ADDRESS", ":8080"),
			TempoEndpoint:   getEnvOrDefault("TEMPO_ENDPOINT", ""),
			TempoInsecure:   getEnvOrDefault("TEMPO_INSECURE", "true") == "true",
			TempoSampleRate: 1.0,
		},
		NATS: NATSConfig{
			URL: getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		},
	}

	return config, nil
}

// SaveConfigToDatabase saves the current configuration to the database
func SaveConfigToDatabase(ctx context.Context, databaseURL string, config *Config, guildName string) error {
	if databaseURL == "" {
		return fmt.Errorf("database URL is required")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	roleMappingsJSON, err := json.Marshal(config.Discord.RoleMappings)
	if err != nil {
		return fmt.Errorf("failed to marshal role mappings: %w", err)
	}

	query := `
		INSERT INTO guild_configs (
			guild_id, guild_name, event_channel_id, event_channel_name,
			leaderboard_channel_id, leaderboard_channel_name,
			signup_channel_id, signup_channel_name, role_mappings,
			registered_role_id, admin_role_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (guild_id) DO UPDATE SET
			guild_name = EXCLUDED.guild_name,
			event_channel_id = EXCLUDED.event_channel_id,
			event_channel_name = EXCLUDED.event_channel_name,
			leaderboard_channel_id = EXCLUDED.leaderboard_channel_id,
			leaderboard_channel_name = EXCLUDED.leaderboard_channel_name,
			signup_channel_id = EXCLUDED.signup_channel_id,
			signup_channel_name = EXCLUDED.signup_channel_name,
			role_mappings = EXCLUDED.role_mappings,
			registered_role_id = EXCLUDED.registered_role_id,
			admin_role_id = EXCLUDED.admin_role_id,
			updated_at = NOW()`

	_, err = db.ExecContext(ctx, query,
		config.GetGuildID(),
		guildName,
		config.GetEventChannelID(),
		"events", // channel name
		config.GetLeaderboardChannelID(),
		"leaderboard", // channel name
		config.GetSignupChannelID(),
		"signup", // channel name
		roleMappingsJSON,
		config.GetRegisteredRoleID(),
		config.GetAdminRoleID(),
	)
	if err != nil {
		return fmt.Errorf("failed to save guild config: %w", err)
	}

	return nil
}
