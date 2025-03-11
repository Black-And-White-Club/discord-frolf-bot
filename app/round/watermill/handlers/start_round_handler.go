package roundhandlers

import (
	"encoding/json"
	"fmt"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundStarted(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling round started event", attr.CorrelationIDFromMsg(msg))
	var payload roundevents.RoundStartedPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		h.Logger.Error(ctx, "Failed to unmarshal RoundStartedPayload", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	h.Logger.Info(ctx, "Preparing to notify Discord of round start", attr.Int64("round_id", payload.RoundID), attr.CorrelationIDFromMsg(msg))
	// Construct the Discord-specific payload
	discordPayload := discordroundevents.DiscordRoundStartPayload{
		RoundID:   payload.RoundID,
		Title:     payload.Title,
		Location:  payload.Location,
		StartTime: payload.StartTime,
		ChannelID: payload.ChannelID,
	}
	discordMsg, err := h.createResultMessage(msg, discordPayload, discordroundevents.RoundStartedTopic)
	if err != nil {
		return nil, err
	}
	h.Logger.Info(ctx, "Successfully processed round started event", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
