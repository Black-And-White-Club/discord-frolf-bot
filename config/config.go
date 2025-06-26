package config

import (
	"context"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	NATS          NATSConfig          `yaml:"nats"`
	Discord       DiscordConfig       `yaml:"discord"`
	Service       ServiceConfig       `yaml:"service"`
	Observability ObservabilityConfig `yaml:"observability"`
	DatabaseURL   string              `yaml:"database_url"` // PostgreSQL connection string

	// Internal state management
	mu       sync.RWMutex // For thread-safe access
	isFromDB bool         // Track if config came from database
}

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URL string `yaml:"url"`
}

// DiscordConfig holds Discord bot configuration
type DiscordConfig struct {
	Token                string            `yaml:"token"`
	SignupChannelID      string            `yaml:"signup_channel_id"`
	SignupMessageID      string            `yaml:"signup_message_id"`
	SignupEmoji          string            `yaml:"signup_emoji"`
	RegisteredRoleID     string            `yaml:"registered_role_id"`
	EventChannelID       string            `yaml:"event_channel_id"`
	LeaderboardChannelID string            `yaml:"leaderboard_channel_id"`
	GuildID              string            `yaml:"guild_id"`
	AppID                string            `yaml:"app_id"`
	URL                  string            `yaml:"url"`
	RoleMappings         map[string]string `yaml:"role_mappings"`
	AdminRoleID          string            `yaml:"admin_role_id"`
}

// ServiceConfig holds service metadata
type ServiceConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	LokiURL         string  `yaml:"loki_url"`
	MetricsAddress  string  `yaml:"metrics_address"`
	TempoEndpoint   string  `yaml:"tempo_endpoint"`
	TempoInsecure   bool    `yaml:"tempo_insecure"`
	TempoSampleRate float64 `yaml:"tempo_sample_rate"`
	Environment     string  `yaml:"environment"`
}

// LoadConfig loads configuration from the specified file path
func LoadConfig(configPath string) (*Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	guildID := os.Getenv("DISCORD_GUILD_ID")

	// Try database-backed config first if available
	if databaseURL != "" && guildID != "" {
		if cfg, err := LoadConfigFromDatabase(context.Background(), databaseURL, guildID); err == nil {
			cfg.isFromDB = true
			return cfg, nil
		}
		// If database fails, fall back to file (for initial setup or migration)
		fmt.Printf("Database config failed, falling back to file: %s\n", configPath)
	}

	// Load from file
	cfg := &Config{}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables if present
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		cfg.Discord.Token = token
	}
	if guildID := os.Getenv("DISCORD_GUILD_ID"); guildID != "" {
		cfg.Discord.GuildID = guildID
	}
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		cfg.NATS.URL = natsURL
	}
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	}

	cfg.isFromDB = false
	return cfg, nil
}

// Getter methods for backward compatibility
func (c *Config) GetGuildID() string {
	return c.Discord.GuildID
}

func (c *Config) GetSignupChannelID() string {
	return c.Discord.SignupChannelID
}

func (c *Config) GetSignupMessageID() string {
	return c.Discord.SignupMessageID
}

func (c *Config) GetSignupEmoji() string {
	return c.Discord.SignupEmoji
}

func (c *Config) GetEventChannelID() string {
	return c.Discord.EventChannelID
}

func (c *Config) GetLeaderboardChannelID() string {
	return c.Discord.LeaderboardChannelID
}

func (c *Config) GetRegisteredRoleID() string {
	return c.Discord.RegisteredRoleID
}

func (c *Config) GetAdminRoleID() string {
	return c.Discord.AdminRoleID
}

func (c *Config) GetRoleMappings() map[string]string {
	return c.Discord.RoleMappings
}

// UpdateConfigFromDatabase refreshes config from database if available
func (c *Config) UpdateConfigFromDatabase() error {
	if !c.isFromDB || c.DatabaseURL == "" {
		return fmt.Errorf("config is not database-backed")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	guildID := c.GetGuildID()
	updatedConfig, err := LoadConfigFromDatabase(context.Background(), c.DatabaseURL, guildID)
	if err != nil {
		return fmt.Errorf("failed to reload config from database: %w", err)
	}

	// Update current config with new values
	c.Discord = updatedConfig.Discord
	c.Service = updatedConfig.Service
	c.Observability = updatedConfig.Observability
	c.NATS = updatedConfig.NATS

	return nil
}

// SaveToDatabase saves current config to database (if database-backed)
func (c *Config) SaveToDatabase(guildName string) error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("no database URL configured")
	}

	return SaveConfigToDatabase(context.Background(), c.DatabaseURL, c, guildName)
}

// IsConfigFromDB returns true if the config was loaded from the database
func (c *Config) IsConfigFromDB() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isFromDB
}
