package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config struct to hold the configuration settings
type Config struct {
	NATS    NATSConfig    `yaml:"nats"`
	Discord DiscordConfig `yaml:"discord"`
	// ... other configuration fields ...
}

// NATSConfig holds NATS configuration.
type NATSConfig struct {
	URL string `yaml:"url"`
}

// DiscordConfig holds Discord configuration.
type DiscordConfig struct {
	Token            string `yaml:"token"`
	OptInMessageID   string `yaml:"opt_in_message_id"`
	OptInChannelID   string `yaml:"opt_in_channel_id"`
	RegisteredRoleID string `yaml:"registered_role_id"`
	GuildID          string `yaml:"guild_id"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(filename string) (*Config, error) {
	// Try reading configuration from the file first
	data, err := os.ReadFile(filename)
	if err != nil {
		// If the file is not found, try loading from environment variables
		fmt.Printf("Failed to read config file: %v\n", err)
		fmt.Println("Trying to load configuration from environment variables...")

		return loadConfigFromEnv()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// loadConfigFromEnv loads the configuration from environment variables.
func loadConfigFromEnv() (*Config, error) {
	var cfg Config

	// Load NATS URL
	cfg.NATS.URL = os.Getenv("NATS_URL")
	if cfg.NATS.URL == "" {
		return nil, fmt.Errorf("NATS_URL environment variable not set")
	}
	// Load Discord token
	cfg.Discord.Token = os.Getenv("DISCORD_TOKEN")
	if cfg.Discord.Token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable not set")
	}

	// Load other Discord configuration fields
	cfg.Discord.OptInMessageID = os.Getenv("DISCORD_OPT_IN_MESSAGE_ID")
	cfg.Discord.OptInChannelID = os.Getenv("DISCORD_OPT_IN_CHANNEL_ID")
	cfg.Discord.RegisteredRoleID = os.Getenv("DISCORD_REGISTERED_ROLE_ID")
	cfg.Discord.GuildID = os.Getenv("DISCORD_GUILD_ID")

	return &cfg, nil
}
