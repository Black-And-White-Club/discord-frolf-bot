package handlers

import (
	"context"
	"errors"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

var errInteractionStoreUnavailable = errors.New("interaction store unavailable")

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	correlationID, _ := ctx.Value("correlation_id").(string)
	return correlationID
}

// getInteractionForGuildResponse prefers the per-request correlation key and
// falls back to the legacy guild key for one release.
func (h *GuildHandlers) getInteractionForGuildResponse(ctx context.Context, guildID string) (*discordgo.Interaction, string, error) {
	if h.interactionStore == nil {
		return nil, "", errInteractionStoreUnavailable
	}

	if correlationID := correlationIDFromContext(ctx); correlationID != "" {
		interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, correlationID)
		if err == nil {
			return interaction, correlationID, nil
		}
		if h.logger != nil {
			h.logger.WarnContext(ctx, "Failed to resolve interaction by correlation ID; falling back to guild key",
				attr.String("guild_id", guildID),
				attr.String("correlation_id", correlationID),
				attr.Error(err))
		}
	}

	interaction, err := discordutils.GetInteraction(ctx, h.interactionStore, guildID)
	if err != nil {
		return nil, "", err
	}

	return interaction, guildID, nil
}
