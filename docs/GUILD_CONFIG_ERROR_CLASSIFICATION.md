# Guild Config Error Classification Guide

This guide explains how to properly classify and handle errors in the guild configuration system for robust error handling and user experience.

## Error Types

### 1. ConfigLoadingError
- **When to use**: When a config request is in progress
- **User experience**: "Loading... please try again in a moment"
- **Retry behavior**: User should retry after a brief wait

### 2. ConfigNotFoundError (Permanent)
- **When to use**: Guild doesn't exist, not configured, or setup incomplete
- **User experience**: "Server needs setup - ask admin to run /frolf-setup"
- **Retry behavior**: No automatic retry - requires manual intervention

### 3. ConfigTemporaryError (Temporary)
- **When to use**: Network issues, database timeouts, service unavailable
- **User experience**: "Temporary issue - please try again"
- **Retry behavior**: Automatic retry after circuit breaker recovery

## Backend Error Classification

### Automatic Classification

The registry provides `ClassifyBackendError()` to automatically classify errors:

```go
// In backend event handlers
func HandleConfigRequest(guildID string) {
    config, err := fetchGuildConfig(guildID)
    if err != nil {
        // Automatically classify the error
        isPermanent, reason := interactions.ClassifyBackendError(err, guildID)
        
        // Send failure event with classification
        sendConfigRetrievalFailed(guildID, reason, isPermanent)
        return
    }
    
    // Send success event
    sendConfigRetrieved(guildID, config)
}
```

### Manual Classification Examples

#### Permanent Errors (isPermanent = true)
```go
// Guild doesn't exist in Discord
if err == discord.ErrGuildNotFound {
    registry.HandleGuildConfigRetrievalFailed(guildID, "guild not found in Discord", true)
}

// Guild not configured in our system
if err == ErrGuildNotConfigured {
    registry.HandleGuildConfigRetrievalFailed(guildID, "guild setup incomplete", true)
}

// Permission denied
if err == ErrUnauthorized {
    registry.HandleGuildConfigRetrievalFailed(guildID, "insufficient permissions", true)
}
```

#### Temporary Errors (isPermanent = false)
```go
// Database connection timeout
if err == sql.ErrConnDone {
    registry.HandleGuildConfigRetrievalFailed(guildID, "database connection timeout", false)
}

// Redis unavailable
if strings.Contains(err.Error(), "redis") {
    registry.HandleGuildConfigRetrievalFailed(guildID, "cache temporarily unavailable", false)
}

// Context deadline exceeded
if err == context.DeadlineExceeded {
    registry.HandleGuildConfigRetrievalFailed(guildID, "request timeout", false)
}

// Discord API rate limit
if err == discord.ErrRateLimit {
    registry.HandleGuildConfigRetrievalFailed(guildID, "Discord API rate limited", false)
}
```

## Helper Methods

### Creating Specific Error Types
```go
// For permanent failures
err := interactions.NewConfigNotFoundError(guildID, "guild not configured")

// For temporary failures
err := interactions.NewConfigTemporaryError(guildID, "database timeout", originalErr)

// For loading state
err := interactions.NewConfigLoadingError(guildID)
```

### Unified Error Handling
```go
// Let the registry automatically classify and handle the error
registry.HandleBackendError(guildID, err)
```

### Error Type Detection
```go
// Check error types in your code
if interactions.IsConfigLoading(err) {
    // Handle loading state
} else if interactions.IsConfigNotFound(err) {
    // Handle permanent failure
} else if interactions.IsConfigTemporaryError(err) {
    // Handle temporary failure
}

// Get error type string for logging
errorType := interactions.GetErrorType(err)
log.Info("Error occurred", "type", errorType, "guild", guildID)
```

## Circuit Breaker Integration

The circuit breaker automatically:
- Opens after 5 consecutive temporary failures
- Stays open for 1 minute
- Only counts temporary failures (permanent failures don't affect circuit state)
- Returns `ConfigTemporaryError` when open

## Best Practices

1. **Default to Temporary**: When in doubt, classify as temporary to avoid permanent caching
2. **Be Specific**: Provide clear reason strings for better debugging
3. **Use Patterns**: The auto-classifier recognizes common error patterns
4. **Monitor Metrics**: Track error types for system health monitoring
5. **User Feedback**: Different error types provide appropriate user guidance

## Error Patterns Recognized

### Permanent Patterns
- "guild not found", "guild does not exist"
- "guild not configured", "not configured" 
- "setup required", "setup incomplete"
- "unauthorized", "forbidden", "permission denied"

### Temporary Patterns
- "timeout", "connection", "network"
- "unavailable", "busy", "rate limit"
- "database", "redis", "context deadline"
- "service unavailable", "internal server error"
- "bad gateway", "gateway timeout"

## Monitoring and Observability

The registry provides metrics for monitoring error patterns:
- Error type distribution
- Circuit breaker state changes
- Cache hit/miss ratios with error context
- Response time patterns for different error types

This classification enables better alerting, where temporary errors might trigger capacity alerts while permanent errors might trigger configuration alerts.
