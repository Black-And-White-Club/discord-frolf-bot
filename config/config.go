package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config struct to hold the configuration settings
type Config struct {
	NATS    NATSConfig    `yaml:"nats"`
	Discord DiscordConfig `yaml:"discord"`
	Service ServiceConfig `yaml:"service"` // Add ServiceConfig
	Loki    LokiConfig    `yaml:"loki"`    // Add LokiConfig
	Tempo   TempoConfig   `yaml:"tempo"`
	// ... other configuration fields ...
}

// NATSConfig holds NATS configuration.
type NATSConfig struct {
	URL string `yaml:"url"`
}

// DiscordConfig holds Discord configuration.
type DiscordConfig struct {
	Token            string            `yaml:"token"`
	RegisteredRoleID string            `yaml:"registered_role_id"`
	GuildID          string            `yaml:"guild_id"`
	DiscordAppID     string            `yaml:"discord_app_id"`
	RoleMappings     map[string]string `yaml:"role_mappings"`
	SignupChannelID  string            `yaml:"signup_channel_id"`
	SignupMessageID  string            `yaml:"signup_message_id"`
	SignupEmoji      string            `yaml:"signup_emoji"`
	AdminRoleID      string            `yaml:"admin_role_id"`
	ChannelID        string            `yaml:"event_channel_id"`
}

// ServiceConfig holds general service configuration
type ServiceConfig struct {
	Name string `yaml:"name"`
}

// LokiConfig holds Loki configuration.
type LokiConfig struct {
	URL      string `yaml:"url"`
	TenantID string `yaml:"tenant_id"`
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type TempoConfig struct {
	Endpoint    string  `yaml:"url"`
	Insecure    bool    `yaml:"insecure"`
	ServiceName string  `yaml:"service_name"`
	ServiceVer  string  `yaml:"service_version"`
	SampleRate  float64 `yaml:"sample_rate"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		fmt.Println("Trying to load configuration from environment variables...")
		cfg := &Config{} // Create a new Config to pass to loadConfigFromEnv
		return loadConfigFromEnv(cfg)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	// After unmarshaling, load from environment if values are missing
	loadConfigFromEnv(&cfg)
	fmt.Println("Loaded Discord Token:", cfg.Discord.Token)
	fmt.Println("Loaded NATS:", cfg.NATS.URL)
	fmt.Println("Loaded Loki:", cfg.Loki.URL)
	fmt.Println("Loaded GuildID:", cfg.Discord.GuildID)
	return &cfg, nil
}

// loadConfigFromEnv loads the configuration from environment variables.
func loadConfigFromEnv(cfg *Config) (*Config, error) {
	// Only load from environment variables if the value is not already set.
	if cfg.NATS.URL == "" {
		cfg.NATS.URL = os.Getenv("NATS_URL")
		if cfg.NATS.URL == "" {
			return nil, fmt.Errorf("NATS_URL environment variable not set")
		}
	}
	if cfg.Service.Name == "" {
		cfg.Service.Name = os.Getenv("SERVICE_NAME")
		if cfg.Service.Name == "" {
			return nil, fmt.Errorf("SERVICE_NAME environment variable not set")
		}
	}
	if cfg.Loki.URL == "" {
		cfg.Loki.URL = os.Getenv("LOKI_URL")
		if cfg.Loki.URL == "" {
			return nil, fmt.Errorf("LOKI_URL environment variable not set")
		}
	}
	if cfg.Loki.TenantID == "" {
		cfg.Loki.TenantID = os.Getenv("LOKI_TENANT_ID")
		if cfg.Loki.TenantID == "" {
			return nil, fmt.Errorf("LOKI_TENANT_ID environment variable not set")
		}
	}
	if cfg.Discord.Token == "" {
		cfg.Discord.Token = os.Getenv("DISCORD_TOKEN")
		if cfg.Discord.Token == "" {
			return nil, fmt.Errorf("DISCORD_TOKEN environment variable not set")
		}
	}
	if cfg.Discord.RegisteredRoleID == "" {
		cfg.Discord.RegisteredRoleID = os.Getenv("DISCORD_REGISTERED_ROLE_ID")
	}
	if cfg.Discord.GuildID == "" {
		cfg.Discord.GuildID = os.Getenv("DISCORD_GUILD_ID")
	}
	if cfg.Discord.DiscordAppID == "" {
		cfg.Discord.DiscordAppID = os.Getenv("DISCORD_APP_ID")
	}
	if cfg.Discord.SignupChannelID == "" {
		cfg.Discord.SignupChannelID = os.Getenv("DISCORD_SIGNUP_CHANNEL_ID")
	}
	if cfg.Discord.SignupMessageID == "" {
		cfg.Discord.SignupMessageID = os.Getenv("DISCORD_SIGNUP_MESSAGE_ID")
	}
	// Load role mappings from environment variables (special handling)
	if cfg.Discord.RoleMappings == nil {
		cfg.Discord.RoleMappings = make(map[string]string)
		for _, envVar := range os.Environ() {
			if strings.HasPrefix(envVar, "DISCORD_ROLE_MAPPING_") {
				parts := strings.SplitN(envVar, "=", 2)
				if len(parts) != 2 {
					continue
				}
				key := strings.TrimPrefix(parts[0], "DISCORD_ROLE_MAPPING_")
				value := parts[1]
				cfg.Discord.RoleMappings[key] = value
			}
		}
	}
	return cfg, nil
}
