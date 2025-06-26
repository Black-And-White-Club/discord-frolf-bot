package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	NATS          NATSConfig          `yaml:"nats"`
	Discord       DiscordConfig       `yaml:"discord"`
	Service       ServiceConfig       `yaml:"service"`
	Observability ObservabilityConfig `yaml:"observability"`
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

	return cfg, nil
}
