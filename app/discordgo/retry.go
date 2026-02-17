package discord

import (
	"crypto/rand"
	"errors"
	"log/slog"
	"math/big"
	"net"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

const (
	maxDiscordAPIRetryAttempts = 5
	discordAPIBaseRetryDelay   = 200 * time.Millisecond
	discordAPIMaxRetryDelay    = 3 * time.Second
)

// RetryDiscordAPI retries transient Discord API failures with exponential backoff and jitter.
func RetryDiscordAPI(logger *slog.Logger, operation string, fn func() error) error {
	delay := discordAPIBaseRetryDelay
	var lastErr error

	for attempt := 1; attempt <= maxDiscordAPIRetryAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt == maxDiscordAPIRetryAttempts || !isRetryableDiscordError(err) {
			return err
		}

		wait := delay + randomJitter(delay/2)
		if logger != nil {
			logger.Warn("Retrying transient Discord API failure",
				attr.String("operation", operation),
				attr.Int("attempt", attempt),
				attr.Duration("retry_in", wait),
				attr.Error(err),
			)
		}

		time.Sleep(wait)
		delay *= 2
		if delay > discordAPIMaxRetryDelay {
			delay = discordAPIMaxRetryDelay
		}
	}

	return lastErr
}

func isRetryableDiscordError(err error) bool {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Response != nil {
			status := restErr.Response.StatusCode
			if status == 429 || status >= 500 {
				return true
			}
		}
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

func randomJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}

	n, err := rand.Int(rand.Reader, big.NewInt(max.Nanoseconds()+1))
	if err != nil {
		return 0
	}
	return time.Duration(n.Int64())
}
