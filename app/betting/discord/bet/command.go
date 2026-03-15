package bet

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// BetManager handles the /bet Discord command.
type BetManager interface {
	HandleBetCommand(ctx context.Context, i *discordgo.InteractionCreate)
}

// betManager implements BetManager.
type betManager struct {
	session discord.Session
	logger  *slog.Logger
	cfg     *config.Config
	tracer  trace.Tracer
	metrics discordmetrics.DiscordMetrics
}

// NewBetManager creates a new BetManager.
func NewBetManager(
	session discord.Session,
	logger *slog.Logger,
	cfg *config.Config,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) BetManager {
	return &betManager{
		session: session,
		logger:  logger,
		cfg:     cfg,
		tracer:  tracer,
		metrics: metrics,
	}
}

// BetCommand returns the /bet command definition.
func BetCommand() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "bet",
		Description: "Access the seasonal betting module for this club",
	}
}
