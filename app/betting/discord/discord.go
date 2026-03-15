package bettingdiscord

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/betting/discord/bet"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"go.opentelemetry.io/otel/trace"
)

// BettingDiscord handles all Discord-related functionality for the betting module.
type BettingDiscord struct {
	logger     *slog.Logger
	cfg        *config.Config
	betManager bet.BetManager
}

// NewBettingDiscord creates a new BettingDiscord instance.
func NewBettingDiscord(
	ctx context.Context,
	session discord.Session,
	logger *slog.Logger,
	cfg *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (*BettingDiscord, error) {
	if session == nil {
		logger.ErrorContext(ctx, "session cannot be nil")
		return nil, fmt.Errorf("session cannot be nil")
	}

	betManager := bet.NewBetManager(session, logger, cfg, tracer, metrics)

	return &BettingDiscord{
		logger:     logger,
		cfg:        cfg,
		betManager: betManager,
	}, nil
}

// GetBetManager returns the bet command manager.
func (d *BettingDiscord) GetBetManager() bet.BetManager {
	return d.betManager
}
