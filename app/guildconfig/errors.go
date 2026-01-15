package guildconfig

import (
	"errors"
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

// IsConfigLoading checks if an error indicates config loading using errors.As
func IsConfigLoading(err error) bool {
	var target *ConfigLoadingError
	return errors.As(err, &target)
}

// ConfigNotFoundError indicates that a guild config doesn't exist (permanent failure)
type ConfigNotFoundError struct {
	GuildID string
	Reason  string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("guild config not found for guild %s: %s", e.GuildID, e.Reason)
}

// IsConfigNotFound checks if an error indicates a permanent config absence using errors.As
func IsConfigNotFound(err error) bool {
	var target *ConfigNotFoundError
	return errors.As(err, &target)
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

// IsConfigTemporaryError checks if an error indicates a temporary failure using errors.As
func IsConfigTemporaryError(err error) bool {
	var target *ConfigTemporaryError
	return errors.As(err, &target)
}

/*
  Helper functions for creating error instances
*/

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
func ClassifyBackendError(err error, guildID string) (isPermanent bool, reason string) {
	if err == nil {
		return false, ""
	}

	// If it's already one of our typed errors, respect its existing classification
	if IsConfigNotFound(err) {
		return true, err.Error()
	}

	errMsg := err.Error()
	lowerMsg := strings.ToLower(errMsg)

	// Permanent patterns
	permanentPatterns := []string{
		"guild not found", "guild does not exist", "not configured", "not found",
		"invalid guild", "unauthorized", "forbidden", "setup required",
	}

	for _, pattern := range permanentPatterns {
		if strings.Contains(lowerMsg, pattern) {
			return true, fmt.Sprintf("permanent failure: %s", errMsg)
		}
	}

	// Temporary patterns
	temporaryPatterns := []string{
		"timeout", "connection", "network", "unavailable", "busy",
		"database", "redis", "context deadline", "context canceled",
	}

	for _, pattern := range temporaryPatterns {
		if strings.Contains(lowerMsg, pattern) {
			return false, fmt.Sprintf("temporary failure: %s", errMsg)
		}
	}

	return false, fmt.Sprintf("unknown error (treated as temporary): %s", errMsg)
}
