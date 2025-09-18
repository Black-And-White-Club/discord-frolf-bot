package guildconfig

import (
	"fmt"
	"strings"
	"time"
)

// ResolverConfig holds configuration for the guild config resolver
type ResolverConfig struct {
	// RequestTimeout is the maximum time allowed to publish a retrieval request.
	RequestTimeout time.Duration `yaml:"request_timeout" env:"GUILDCONFIG_REQUEST_TIMEOUT" default:"10s"`
	// ResponseTimeout is the maximum time to wait for a backend response before returning a loading error.
	ResponseTimeout time.Duration `yaml:"response_timeout" env:"GUILDCONFIG_RESPONSE_TIMEOUT" default:"30s"`
}

// Validate checks the configuration for logical consistency and reasonable values
func (c *ResolverConfig) Validate() error {
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive, got %v", c.RequestTimeout)
	}
	if c.ResponseTimeout <= 0 {
		return fmt.Errorf("response_timeout must be positive, got %v", c.ResponseTimeout)
	}
	if c.ResponseTimeout <= c.RequestTimeout {
		return fmt.Errorf("response_timeout (%v) must be longer than request_timeout (%v)", c.ResponseTimeout, c.RequestTimeout)
	}
	if c.RequestTimeout > 60*time.Second {
		return fmt.Errorf("request_timeout (%v) seems excessive, consider reducing for better UX", c.RequestTimeout)
	}
	if c.ResponseTimeout > 5*time.Minute {
		return fmt.Errorf("response_timeout (%v) seems excessive, consider reducing", c.ResponseTimeout)
	}
	return nil
}

// NewResolverConfigForEnvironment creates environment-specific configuration
func NewResolverConfigForEnvironment(env string) *ResolverConfig {
	switch strings.ToLower(env) {
	case "development", "dev":
		return &ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 15 * time.Second}
	case "staging", "stage":
		return &ResolverConfig{RequestTimeout: 8 * time.Second, ResponseTimeout: 25 * time.Second}
	case "production", "prod":
		return DefaultResolverConfig()
	default:
		return DefaultResolverConfig()
	}
}

// DefaultResolverConfig returns sensible defaults
func DefaultResolverConfig() *ResolverConfig {
	return &ResolverConfig{RequestTimeout: 10 * time.Second, ResponseTimeout: 30 * time.Second}
}
