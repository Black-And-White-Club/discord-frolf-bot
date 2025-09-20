package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// HandleRoundDeleted handles the RoundDeleted event using the standardized wrapper
func (h *RoundHandlers) HandleRoundDeleted(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleRoundDeleted",
		&roundevents.RoundDeletedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			p := payload.(*roundevents.RoundDeletedPayload)

			h.Logger.InfoContext(ctx, "Received RoundDeleted event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", p.RoundID.String()),
			)

			if uuid.UUID(p.RoundID) == uuid.Nil {
				h.Logger.ErrorContext(ctx, "Missing RoundID in payload for RoundDeleted event", attr.CorrelationIDFromMsg(msg))
				return nil, fmt.Errorf("missing RoundID in round deleted payload")
			}

			discordMessageID, ok := msg.Metadata["discord_message_id"]
			if !ok || discordMessageID == "" {
				logMsg := "discord_message_id key not found in metadata"
				if ok && discordMessageID == "" {
					logMsg = "discord_message_id found in metadata but is empty"
				}
				h.Logger.ErrorContext(ctx, logMsg,
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("metadata_value", discordMessageID),
				)
				return nil, fmt.Errorf("discord_message_id not found or is empty in message metadata for round %s", p.RoundID.String())
			}

			h.Logger.InfoContext(ctx, "Attempting to delete Discord message for round",
				attr.RoundID("round_id", p.RoundID),
				attr.String("discord_message_id", discordMessageID),
			)

			result, err := h.RoundDiscord.GetDeleteRoundManager().DeleteRoundEventEmbed(ctx, discordMessageID, h.Config.GetEventChannelID())
			if err != nil {
				h.Logger.ErrorContext(ctx, "Error calling DeleteRoundEventEmbed service",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("error calling delete round embed service: %w", err)
			}

			success, ok := result.Success.(bool)
			if !ok {
				h.Logger.ErrorContext(ctx, "Unexpected type for result.Success from DeleteRoundEventEmbed",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Any("result_success_value", result.Success),
				)
				return nil, fmt.Errorf("unexpected type for result.Success in DeleteRoundEventEmbed result for round %s", p.RoundID.String())
			}

			if !success {
				h.Logger.WarnContext(ctx, "Round embed message deletion attempt was not successful via Discord API",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Any("deletion_error_result", result.Error),
				)
			} else {
				h.Logger.InfoContext(ctx, "Successfully confirmed Discord message deletion",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID),
				)
			}

			tracePayload := map[string]interface{}{
				"guild_id":                  p.GuildID,
				"round_id":                  p.RoundID,
				"event_type":                "round_deleted",
				"status":                    "embed_deletion_attempted",
				"discord_message_id":        discordMessageID,
				"embed_deletion_successful": success,
				"embed_deletion_error":      result.Error,
			}

			traceMsg, err := h.Helpers.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create trace event for RoundDeleted event",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", p.RoundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create trace event: %w", err)
			}

			return []*message.Message{traceMsg}, nil
		},
	)(msg)
}
