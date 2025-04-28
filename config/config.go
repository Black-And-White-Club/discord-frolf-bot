package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	obs "github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"gopkg.in/yaml.v3"
)

// Config struct to hold the configuration settings

type Config struct {
	Discord       DiscordConfig       `yaml:"discord"`
	NATS          NATSConfig          `yaml:"nats"`
	Observability ObservabilityConfig `yaml:"observability"`
	Service       ServiceConfig       `yaml:"service"`
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

// NATSConfig holds NATS configuration.

type NATSConfig struct {
	URL string `yaml:"url"`
}

// ObservabilityConfig holds configuration for observability components

type ObservabilityConfig struct {
	LokiURL         string  `yaml:"loki_url"`
	LokiTenantID    string  `yaml:"loki_tenant_id"`
	MetricsAddress  string  `yaml:"metrics_address"`
	TempoEndpoint   string  `yaml:"tempo_endpoint"`
	TempoInsecure   bool    `yaml:"tempo_insecure"`
	TempoSampleRate float64 `yaml:"tempo_sample_rate"`
	Environment     string  `yaml:"environment"`
	Version         string  `yaml:"version"`
}

// ServiceConfig holds general service configuration

type ServiceConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
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
	// Merge with environment variables
	return mergeEnvVars(&cfg)
}

// loadConfigFromEnv loads the configuration from environment variables.

func loadConfigFromEnv() (*Config, error) {
	cfg := &Config{}
	// Discord
	cfg.Discord.Token = os.Getenv("DISCORD_TOKEN")
	cfg.Discord.RegisteredRoleID = os.Getenv("DISCORD_REGISTERED_ROLE_ID")
	cfg.Discord.GuildID = os.Getenv("DISCORD_GUILD_ID")
	cfg.Discord.DiscordAppID = os.Getenv("DISCORD_APP_ID")
	cfg.Discord.SignupChannelID = os.Getenv("DISCORD_SIGNUP_CHANNEL_ID")
	cfg.Discord.SignupMessageID = os.Getenv("DISCORD_SIGNUP_MESSAGE_ID")
	cfg.Discord.SignupEmoji = os.Getenv("DISCORD_SIGNUP_EMOJI")
	cfg.Discord.AdminRoleID = os.Getenv("DISCORD_ADMIN_ROLE_ID")
	cfg.Discord.ChannelID = os.Getenv("DISCORD_EVENT_CHANNEL_ID")
	// Role mappings
	cfg.Discord.RoleMappings = make(map[string]string)
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "DISCORD_ROLE_MAPPING_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimPrefix(parts[0], "DISCORD_ROLE_MAPPING_")
			cfg.Discord.RoleMappings[key] = parts[1]
		}
	}
	// NATS
	cfg.NATS.URL = os.Getenv("NATS_URL")
	// Observability
	cfg.Observability.LokiURL = os.Getenv("LOKI_URL")
	cfg.Observability.LokiTenantID = os.Getenv("LOKI_TENANT_ID")
	cfg.Observability.MetricsAddress = os.Getenv("METRICS_ADDRESS")
	cfg.Observability.TempoEndpoint = os.Getenv("TEMPO_ENDPOINT")
	cfg.Observability.Environment = os.Getenv("ENV")
	// Service
	cfg.Service.Name = os.Getenv("SERVICE_NAME")
	cfg.Service.Version = os.Getenv("SERVICE_VERSION")
	// Parse booleans and floats
	var parseErr error
	cfg.Observability.TempoInsecure, parseErr = strconv.ParseBool(os.Getenv("TEMPO_INSECURE"))
	if parseErr != nil {
		cfg.Observability.TempoInsecure = false
	}
	cfg.Observability.TempoSampleRate, parseErr = strconv.ParseFloat(os.Getenv("TEMPO_SAMPLE_RATE"), 64)
	if parseErr != nil {
		cfg.Observability.TempoSampleRate = 0.1
	}
	// Validate required fields
	if cfg.Discord.Token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable not set")
	}
	if cfg.NATS.URL == "" {
		return nil, fmt.Errorf("NATS_URL environment variable not set")
	}
	if cfg.Observability.MetricsAddress == "" {
		return nil, fmt.Errorf("METRICS_ADDRESS environment variable not set")
	}
	return cfg, nil
}

// mergeEnvVars merges environment variables into existing config

func mergeEnvVars(cfg *Config) (*Config, error) {
	// Discord
	if token := os.Getenv("DISCORD_TOKEN"); token != "" {
		cfg.Discord.Token = token
	}
	if roleID := os.Getenv("DISCORD_REGISTERED_ROLE_ID"); roleID != "" {
		cfg.Discord.RegisteredRoleID = roleID
	}
	if guildID := os.Getenv("DISCORD_GUILD_ID"); guildID != "" {
		cfg.Discord.GuildID = guildID
	}
	if appID := os.Getenv("DISCORD_APP_ID"); appID != "" {
		cfg.Discord.DiscordAppID = appID
	}
	if channelID := os.Getenv("DISCORD_SIGNUP_CHANNEL_ID"); channelID != "" {
		cfg.Discord.SignupChannelID = channelID
	}
	if messageID := os.Getenv("DISCORD_SIGNUP_MESSAGE_ID"); messageID != "" {
		cfg.Discord.SignupMessageID = messageID
	}
	if emoji := os.Getenv("DISCORD_SIGNUP_EMOJI"); emoji != "" {
		cfg.Discord.SignupEmoji = emoji
	}
	if adminRoleID := os.Getenv("DISCORD_ADMIN_ROLE_ID"); adminRoleID != "" {
		cfg.Discord.AdminRoleID = adminRoleID
	}
	if eventChannelID := os.Getenv("DISCORD_EVENT_CHANNEL_ID"); eventChannelID != "" {
		cfg.Discord.ChannelID = eventChannelID
	}
	// Role mappings
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "DISCORD_ROLE_MAPPING_") {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimPrefix(parts[0], "DISCORD_ROLE_MAPPING_")
			cfg.Discord.RoleMappings[key] = parts[1]
		}
	}
	// NATS
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		cfg.NATS.URL = natsURL
	}
	// Observability
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		cfg.Observability.LokiURL = lokiURL
	}
	if tenantID := os.Getenv("LOKI_TENANT_ID"); tenantID != "" {
		cfg.Observability.LokiTenantID = tenantID
	}
	if metricsAddr := os.Getenv("METRICS_ADDRESS"); metricsAddr != "" {
		cfg.Observability.MetricsAddress = metricsAddr
	}
	if tempoEndpoint := os.Getenv("TEMPO_ENDPOINT"); tempoEndpoint != "" {
		cfg.Observability.TempoEndpoint = tempoEndpoint
	}
	if env := os.Getenv("ENV"); env != "" {
		cfg.Observability.Environment = env
	}
	// Service
	if name := os.Getenv("SERVICE_NAME"); name != "" {
		cfg.Service.Name = name
	}
	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		cfg.Service.Version = version
	}
	// Parse booleans and floats
	if tempoInsecure := os.Getenv("TEMPO_INSECURE"); tempoInsecure != "" {
		cfg.Observability.TempoInsecure, _ = strconv.ParseBool(tempoInsecure)
	}
	if sampleRate := os.Getenv("TEMPO_SAMPLE_RATE"); sampleRate != "" {
		cfg.Observability.TempoSampleRate, _ = strconv.ParseFloat(sampleRate, 64)
	}
	// Validate required fields
	if cfg.Discord.Token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN must be set in config or environment")
	}
	if cfg.NATS.URL == "" {
		return nil, fmt.Errorf("NATS_URL must be set in config or environment")
	}
	if cfg.Observability.MetricsAddress == "" {
		return nil, fmt.Errorf("METRICS_ADDRESS must be set in config or environment")
	}
	return cfg, nil
}

// ToObsConfig converts application config to observability config

func ToObsConfig(appCfg *Config) obs.Config {
	return obs.Config{
		ServiceName:     appCfg.Service.Name,
		Environment:     appCfg.Observability.Environment,
		Version:         "1.2.3",
		LokiURL:         appCfg.Observability.LokiURL,
		MetricsAddress:  appCfg.Observability.MetricsAddress,
		TempoEndpoint:   appCfg.Observability.TempoEndpoint,
		TempoInsecure:   appCfg.Observability.TempoInsecure,
		TempoSampleRate: appCfg.Observability.TempoSampleRate,
	}
}
