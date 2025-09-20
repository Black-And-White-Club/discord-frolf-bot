package guildconfig

import (
	"fmt"
	"strings"
)

// ConfigLoadingError indicates that a guild config is currently being loaded
type ConfigLoadingError struct {
	GuildID string
}

func (e *ConfigLoadingError) Error() string {
	return fmt.Sprintf("guild config is being loaded for guild %s", e.GuildID)
}

// IsConfigLoading checks if an error indicates config loading
func IsConfigLoading(err error) bool {
	_, ok := err.(*ConfigLoadingError)
	return ok
}

// ConfigNotFoundError indicates that a guild config doesn't exist (permanent failure)
type ConfigNotFoundError struct {
	GuildID string
	Reason  string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("guild config not found for guild %s: %s", e.GuildID, e.Reason)
}

// IsConfigNotFound checks if an error indicates a permanent config absence
func IsConfigNotFound(err error) bool {
	_, ok := err.(*ConfigNotFoundError)
	return ok
}

// ConfigTemporaryError indicates a temporary failure that might succeed on retry
type ConfigTemporaryError struct {
	GuildID string
	Reason  string
	Cause   error
}

func (e *ConfigTemporaryError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("temporary error getting guild config for %s: %s (caused by: %v)", e.GuildID, e.Reason, e.Cause)
	}
	return fmt.Sprintf("temporary error getting guild config for %s: %s", e.GuildID, e.Reason)
}

func (e *ConfigTemporaryError) Unwrap() error {
	return e.Cause
}

// IsConfigTemporaryError checks if an error indicates a temporary failure
func IsConfigTemporaryError(err error) bool {
	_, ok := err.(*ConfigTemporaryError)
	return ok
}

// Helper functions for creating error instances

// NewConfigNotFoundError creates a ConfigNotFoundError for permanent failures
func NewConfigNotFoundError(guildID, reason string) *ConfigNotFoundError {
	return &ConfigNotFoundError{
		GuildID: guildID,
		Reason:  reason,
	}
}

// NewConfigTemporaryError creates a ConfigTemporaryError for temporary failures
func NewConfigTemporaryError(guildID, reason string, cause error) *ConfigTemporaryError {
	return &ConfigTemporaryError{
		GuildID: guildID,
		Reason:  reason,
		Cause:   cause,
	}
}

// NewConfigLoadingError creates a ConfigLoadingError for in-progress requests
func NewConfigLoadingError(guildID string) *ConfigLoadingError {
	return &ConfigLoadingError{
		GuildID: guildID,
	}
}

// ClassifyBackendError helps backend services classify errors appropriately
// This function can be used by backend event handlers to determine if an error is permanent or temporary
func ClassifyBackendError(err error, guildID string) (isPermanent bool, reason string) {
	if err == nil {
		return false, ""
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Permanent errors - configuration or logical issues
	permanentPatterns := []string{
		"guild not found",
		"guild does not exist",
		"guild not configured",
		"not configured",
		"invalid guild",
		"unauthorized",
		"forbidden",
		"permission denied",
		"guild setup incomplete",
		"setup required",
	}

	for _, pattern := range permanentPatterns {
		if strings.Contains(lowerMsg, pattern) {
			return true, fmt.Sprintf("permanent failure: %s", errMsg)
		}
	}

	// Temporary errors - infrastructure or network issues
	temporaryPatterns := []string{
		"timeout",
		"connection",
		"network",
		"unavailable",
		"busy",
		"rate limit",
		"database",
		"redis",
		"context deadline",
		"context canceled",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range temporaryPatterns {
		if strings.Contains(lowerMsg, pattern) {
			return false, fmt.Sprintf("temporary failure: %s", errMsg)
		}
	}

	// Default to temporary for unknown errors to be safe
	// This prevents permanent caching of potentially recoverable issues
	return false, fmt.Sprintf("unknown error (treated as temporary): %s", errMsg)
}
