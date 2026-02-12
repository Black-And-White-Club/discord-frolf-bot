package history

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
)

// HistoryManager handles /history Discord commands.
type HistoryManager interface {
	HandleHistoryCommand(ctx context.Context, i *discordgo.InteractionCreate)
	HandleTagHistoryResponse(ctx context.Context, payload *leaderboardevents.TagHistoryResponsePayloadV1)
	HandleTagHistoryFailed(ctx context.Context, payload *leaderboardevents.TagHistoryFailedPayloadV1)
	HandleTagGraphResponse(ctx context.Context, payload *leaderboardevents.TagGraphResponsePayloadV1)
	HandleTagGraphFailed(ctx context.Context, payload *leaderboardevents.TagGraphFailedPayloadV1)
}

// historyManager implements HistoryManager.
type historyManager struct {
	session          discord.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	interactionStore storage.ISInterface[any]
	metrics          discordmetrics.DiscordMetrics
}

// NewHistoryManager creates a new HistoryManager.
func NewHistoryManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	interactionStore storage.ISInterface[any],
	metrics discordmetrics.DiscordMetrics,
) HistoryManager {
	return &historyManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		interactionStore: interactionStore,
		metrics:          metrics,
	}
}

// HistoryCommand returns the /history command definition.
func HistoryCommand() *discordgo.ApplicationCommand {
	limitMinValue := 1.0
	return &discordgo.ApplicationCommand{
		Name:        "history",
		Description: "View tag history and charts",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "member",
				Description: "View tag history for a specific member",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "user",
						Description: "The member to look up",
						Type:        discordgo.ApplicationCommandOptionUser,
						Required:    false,
					},
					{
						Name:        "limit",
						Description: "Number of history entries to show (default 50, max 100)",
						Type:        discordgo.ApplicationCommandOptionInteger,
						Required:    false,
						MinValue:    &limitMinValue,
						MaxValue:    100,
					},
				},
			},
			{
				Name:        "chart",
				Description: "View tag history chart for a member",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "user",
						Description: "The member whose chart to view",
						Type:        discordgo.ApplicationCommandOptionUser,
						Required:    false,
					},
				},
			},
		},
	}
}
