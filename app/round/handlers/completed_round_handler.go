package handlers

import (
	"context"

	embedpagination "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/embed_pagination"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRoundCompleted clears pagination state after lifecycle completion.
func (h *RoundHandlers) HandleRoundCompleted(ctx context.Context, payload *roundevents.RoundCompletedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return []handlerwrapper.Result{}, nil
	}

	messageID := payload.RoundData.EventMessageID
	if messageID == "" {
		if metadataMessageID, ok := ctx.Value("discord_message_id").(string); ok {
			messageID = metadataMessageID
		}
	}

	if messageID == "" {
		h.logger.DebugContext(ctx, "RoundCompleted received without event message ID; skipping pagination cleanup",
			attr.RoundID("round_id", payload.RoundID),
		)
		return []handlerwrapper.Result{}, nil
	}

	embedpagination.Delete(messageID)

	h.logger.InfoContext(ctx, "Cleared pagination snapshot for completed round",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("discord_message_id", messageID),
	)

	return []handlerwrapper.Result{}, nil
}
