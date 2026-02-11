package config

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	NATS          NATSConfig          `yaml:"nats"`
	Discord       DiscordConfig       `yaml:"discord"`
	Service       ServiceConfig       `yaml:"service"`
	Observability ObservabilityConfig `yaml:"observability"`
	PWA           PWAConfig           `yaml:"pwa"`
	DatabaseURL   string              `yaml:"database_url"` // PostgreSQL connection string

	// Internal state management
	mu       sync.RWMutex // For thread-safe access
	isFromDB bool         // Track if config came from database
}

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URL string `yaml:"url"`
}

// DiscordConfig holds Discord bot configuration.
// Fields other than Token and AppID are treated as global defaults
// and should be resolved per-guild via GuildConfigResolver whenever possible.
type DiscordConfig struct {
	Token                string            `yaml:"token"`
	SignupChannelID      string            `yaml:"signup_channel_id"`      // Default only
	SignupMessageID      string            `yaml:"signup_message_id"`      // Default only
	SignupEmoji          string            `yaml:"signup_emoji"`           // Default only
	RegisteredRoleID     string            `yaml:"registered_role_id"`     // Default only
	EventChannelID       string            `yaml:"event_channel_id"`       // Default only
	LeaderboardChannelID string            `yaml:"leaderboard_channel_id"` // Default only
	GuildID              string            `yaml:"guild_id"`              // Default/Main guild only; deprecated for multi-tenant scoping
	AppID                string            `yaml:"app_id"`
	URL                  string            `yaml:"url"`
	RoleMappings         map[string]string `yaml:"role_mappings"`
	AdminRoleID          string            `yaml:"admin_role_id"` // Default only
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
	OTLPEndpoint    string  `yaml:"otlp_endpoint"`
	OTLPTransport   string  `yaml:"otlp_transport"`
	OTLPLogsEnabled bool    `yaml:"otlp_logs_enabled"`
}

// PWAConfig holds PWA integration configuration
type PWAConfig struct {
	BaseURL        string `yaml:"base_url"`        // PWA frontend URL (for logging/display only)
	RequestTimeout int    `yaml:"request_timeout"` // Timeout in seconds for magic link requests
}

// LoadConfigFromEnvironment loads configuration from environment variables only
func LoadConfigFromEnvironment() (*Config, error) {
	cfg := &Config{}

	// Required environment variable: token
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("DISCORD_TOKEN environment variable is required")
	}

	// GuildID is optional for global command registration
	guildID := os.Getenv("DISCORD_GUILD_ID")
	if guildID == "" {
		fmt.Println("Warning: DISCORD_GUILD_ID not provided. Bot will register commands globally and work in any server.")
	}

	// Set required fields
	cfg.Discord.Token = token
	cfg.Discord.GuildID = guildID

	// Set optional fields with defaults
	cfg.NATS.URL = getEnvOrDefault("NATS_URL", "nats://localhost:4222")
	cfg.DatabaseURL = os.Getenv("DATABASE_URL") // Can be empty

	// Discord optional fields
	cfg.Discord.SignupChannelID = os.Getenv("DISCORD_SIGNUP_CHANNEL_ID")
	cfg.Discord.SignupMessageID = os.Getenv("DISCORD_SIGNUP_MESSAGE_ID")
	cfg.Discord.SignupEmoji = getEnvOrDefault("DISCORD_SIGNUP_EMOJI", "✅")
	cfg.Discord.RegisteredRoleID = os.Getenv("DISCORD_REGISTERED_ROLE_ID")
	cfg.Discord.EventChannelID = os.Getenv("DISCORD_EVENT_CHANNEL_ID")
	cfg.Discord.LeaderboardChannelID = os.Getenv("DISCORD_LEADERBOARD_CHANNEL_ID")
	cfg.Discord.AppID = os.Getenv("DISCORD_APP_ID")
	cfg.Discord.URL = os.Getenv("DISCORD_URL")
	cfg.Discord.AdminRoleID = os.Getenv("DISCORD_ADMIN_ROLE_ID")

	// Service config
	cfg.Service.Name = getEnvOrDefault("SERVICE_NAME", "discord-frolf-bot")
	cfg.Service.Version = getEnvOrDefault("SERVICE_VERSION", "1.0.0")

	// Observability config
	cfg.Observability.LokiURL = os.Getenv("LOKI_URL")
	cfg.Observability.MetricsAddress = getEnvOrDefault("METRICS_ADDRESS", ":8080")
	cfg.Observability.TempoEndpoint = os.Getenv("TEMPO_ENDPOINT")
	cfg.Observability.OTLPEndpoint = os.Getenv("OTLP_ENDPOINT")
	cfg.Observability.OTLPTransport = os.Getenv("OTLP_TRANSPORT")
	cfg.Observability.OTLPLogsEnabled = os.Getenv("OTLP_LOGS_ENABLED") == "true"
	cfg.Observability.Environment = getEnvOrDefault("ENVIRONMENT", "development")

	// Handle boolean and float environment variables
	if tempoInsecure := os.Getenv("TEMPO_INSECURE"); tempoInsecure != "" {
		cfg.Observability.TempoInsecure = (tempoInsecure == "true")
	}
	if sampleRate := os.Getenv("TEMPO_SAMPLE_RATE"); sampleRate != "" {
		if rate, err := strconv.ParseFloat(sampleRate, 64); err == nil {
			cfg.Observability.TempoSampleRate = rate
		}
	}

	// PWA config
	cfg.PWA.BaseURL = getEnvOrDefault("PWA_BASE_URL", "https://pwa.frolf-bot.com")
	cfg.PWA.RequestTimeout = getIntEnvOrDefault("PWA_REQUEST_TIMEOUT", 5)

	// Role mappings from environment variables (JSON format)
	// This could be extended to parse JSON if needed
	cfg.Discord.RoleMappings = make(map[string]string)

	cfg.isFromDB = false
	return cfg, nil
}

// LoadBaseConfig loads only the base bot configuration (tokens, URLs, etc.)
// Guild-specific configurations will be loaded from the backend via events
func LoadBaseConfig() (*Config, error) {
	// Initialize base config with defaults
	config := &Config{
		Discord: DiscordConfig{
			Token:       getEnvOrError("DISCORD_TOKEN"),
			AppID:       getEnvOrError("DISCORD_APP_ID"),
			SignupEmoji: "🐍", // Default emoji
			// Guild-specific fields will be populated from backend
		},
		Service: ServiceConfig{
			Name:    getEnvOrDefault("SERVICE_NAME", "discord-frolf-bot"),
			Version: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		},
		Observability: ObservabilityConfig{
			Environment:     getEnvOrDefault("ENVIRONMENT", "production"),
			LokiURL:         getEnvOrDefault("LOKI_URL", ""),
			MetricsAddress:  getEnvOrDefault("METRICS_ADDRESS", ":8080"),
			TempoEndpoint:   getEnvOrDefault("TEMPO_ENDPOINT", ""),
			TempoInsecure:   getEnvOrDefault("TEMPO_INSECURE", "true") == "true",
			TempoSampleRate: 1.0,
			OTLPEndpoint:    getEnvOrDefault("OTLP_ENDPOINT", ""),
			OTLPTransport:   getEnvOrDefault("OTLP_TRANSPORT", ""),
			OTLPLogsEnabled: getEnvOrDefault("OTLP_LOGS_ENABLED", "false") == "true",
		},
		NATS: NATSConfig{
			URL: getEnvOrDefault("NATS_URL", "nats://localhost:4222"),
		},
		PWA: PWAConfig{
			BaseURL:        getEnvOrDefault("PWA_BASE_URL", "https://pwa.frolf-bot.com"),
			RequestTimeout: getIntEnvOrDefault("PWA_REQUEST_TIMEOUT", 5),
		},
	}

	// Parse float for sample rate
	if sampleRate := os.Getenv("TEMPO_SAMPLE_RATE"); sampleRate != "" {
		if rate, err := strconv.ParseFloat(sampleRate, 64); err == nil {
			config.Observability.TempoSampleRate = rate
		}
	}

	return config, nil
}

// getEnvOrError returns environment variable value or returns an error if missing
func getEnvOrError(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// For required values, we'll panic since the bot can't function without them
	panic(fmt.Sprintf("Required environment variable %s is not set", key))
}

// Helper function to get environment variable with default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to get integer environment variable with default
func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// LoadConfig loads configuration from the specified file path with fallbacks
func LoadConfig(configPath string) (*Config, error) {
	// Try to load from file
	cfg := &Config{}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist and we have essential env vars, try environment-only mode
		if os.IsNotExist(err) && os.Getenv("DISCORD_TOKEN") != "" {
			fmt.Printf("Config file %s not found, but essential environment variables are present. Using environment-only configuration.\n", configPath)
			return LoadConfigFromEnvironment()
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(cfg)

	cfg.isFromDB = false
	return cfg, nil
}

// applyEnvironmentOverrides applies environment variable overrides to a config
func applyEnvironmentOverrides(cfg *Config) {
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

	// Discord overrides
	if signupChannelID := os.Getenv("DISCORD_SIGNUP_CHANNEL_ID"); signupChannelID != "" {
		cfg.Discord.SignupChannelID = signupChannelID
	}
	if signupMessageID := os.Getenv("DISCORD_SIGNUP_MESSAGE_ID"); signupMessageID != "" {
		cfg.Discord.SignupMessageID = signupMessageID
	}
	if signupEmoji := os.Getenv("DISCORD_SIGNUP_EMOJI"); signupEmoji != "" {
		cfg.Discord.SignupEmoji = signupEmoji
	}
	if registeredRoleID := os.Getenv("DISCORD_REGISTERED_ROLE_ID"); registeredRoleID != "" {
		cfg.Discord.RegisteredRoleID = registeredRoleID
	}
	if eventChannelID := os.Getenv("DISCORD_EVENT_CHANNEL_ID"); eventChannelID != "" {
		cfg.Discord.EventChannelID = eventChannelID
	}
	if leaderboardChannelID := os.Getenv("DISCORD_LEADERBOARD_CHANNEL_ID"); leaderboardChannelID != "" {
		cfg.Discord.LeaderboardChannelID = leaderboardChannelID
	}
	if appID := os.Getenv("DISCORD_APP_ID"); appID != "" {
		cfg.Discord.AppID = appID
	}
	if url := os.Getenv("DISCORD_URL"); url != "" {
		cfg.Discord.URL = url
	}
	if adminRoleID := os.Getenv("DISCORD_ADMIN_ROLE_ID"); adminRoleID != "" {
		cfg.Discord.AdminRoleID = adminRoleID
	}

	// Service overrides
	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		cfg.Service.Name = serviceName
	}
	if serviceVersion := os.Getenv("SERVICE_VERSION"); serviceVersion != "" {
		cfg.Service.Version = serviceVersion
	}

	// Override observability settings with environment variables
	if lokiURL := os.Getenv("LOKI_URL"); lokiURL != "" {
		cfg.Observability.LokiURL = lokiURL
	}
	if metricsAddr := os.Getenv("METRICS_ADDRESS"); metricsAddr != "" {
		cfg.Observability.MetricsAddress = metricsAddr
	}
	if tempoEndpoint := os.Getenv("TEMPO_ENDPOINT"); tempoEndpoint != "" {
		cfg.Observability.TempoEndpoint = tempoEndpoint
	}
	if otlpEndpoint := os.Getenv("OTLP_ENDPOINT"); otlpEndpoint != "" {
		cfg.Observability.OTLPEndpoint = otlpEndpoint
	}
	if otlpTransport := os.Getenv("OTLP_TRANSPORT"); otlpTransport != "" {
		cfg.Observability.OTLPTransport = otlpTransport
	}
	if otlpLogsEnabled := os.Getenv("OTLP_LOGS_ENABLED"); otlpLogsEnabled != "" {
		cfg.Observability.OTLPLogsEnabled = (otlpLogsEnabled == "true")
	}
	if environment := os.Getenv("ENVIRONMENT"); environment != "" {
		cfg.Observability.Environment = environment
	}
	// Handle boolean and float environment variables
	if tempoInsecure := os.Getenv("TEMPO_INSECURE"); tempoInsecure != "" {
		cfg.Observability.TempoInsecure = (tempoInsecure == "true")
	}
	if sampleRate := os.Getenv("TEMPO_SAMPLE_RATE"); sampleRate != "" {
		if rate, err := strconv.ParseFloat(sampleRate, 64); err == nil {
			cfg.Observability.TempoSampleRate = rate
		}
	}

	// PWA overrides
	if pwaBaseURL := os.Getenv("PWA_BASE_URL"); pwaBaseURL != "" {
		cfg.PWA.BaseURL = pwaBaseURL
	}
	if pwaTimeout := os.Getenv("PWA_REQUEST_TIMEOUT"); pwaTimeout != "" {
		if timeout, err := strconv.Atoi(pwaTimeout); err == nil {
			cfg.PWA.RequestTimeout = timeout
		}
	}
}

// Getter methods for backward compatibility or global defaults.
// Use GuildConfigResolver for guild-specific settings.

// GetGuildID returns the default guild ID.
// Deprecated: Use interaction GuildID or context-based scoping.
func (c *Config) GetGuildID() string {
	return c.Discord.GuildID
}

// GetSignupChannelID returns the default signup channel ID.
func (c *Config) GetSignupChannelID() string {
	return c.Discord.SignupChannelID
}

// GetSignupMessageID returns the default signup message ID.
func (c *Config) GetSignupMessageID() string {
	return c.Discord.SignupMessageID
}

// GetSignupEmoji returns the default signup emoji.
func (c *Config) GetSignupEmoji() string {
	return c.Discord.SignupEmoji
}

// GetEventChannelID returns the default event channel ID.
func (c *Config) GetEventChannelID() string {
	return c.Discord.EventChannelID
}

// GetLeaderboardChannelID returns the default leaderboard channel ID.
func (c *Config) GetLeaderboardChannelID() string {
	return c.Discord.LeaderboardChannelID
}

// GetRegisteredRoleID returns the default registered role ID.
func (c *Config) GetRegisteredRoleID() string {
	return c.Discord.RegisteredRoleID
}

// GetAdminRoleID returns the default admin role ID.
func (c *Config) GetAdminRoleID() string {
	return c.Discord.AdminRoleID
}

func (c *Config) GetRoleMappings() map[string]string {
	return c.Discord.RoleMappings
}

// IsGuildConfigured is deprecated and should not be used as it relies on global state mutation.
// Use GuildConfigResolver instead.
func (c *Config) IsGuildConfigured(guildID string) bool {
	return false
}
