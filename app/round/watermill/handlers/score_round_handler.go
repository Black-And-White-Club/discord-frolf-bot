package roundhandlers

import (
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleParticipantScoreUpdated handles a successful score update event
func (h *RoundHandlers) HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling participant score update", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.ParticipantScoreUpdatedPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	scorePtr := &payload.Score

	// Send confirmation message via Discord
	err := h.RoundDiscord.GetScoreRoundManager().SendScoreUpdateConfirmation(payload.ChannelID, payload.Participant, scorePtr)
	if err != nil {
		h.Logger.Error(ctx, "Failed to send score update confirmation", attr.Error(err))
	}

	return nil, nil
}

// HandleScoreUpdateError processes a failed score update
func (h *RoundHandlers) HandleScoreUpdateError(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling score update error", attr.CorrelationIDFromMsg(msg))

	var payload roundevents.RoundScoreUpdateErrorPayload
	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
		return nil, err
	}

	// Ensure there is an error message before sending it
	if payload.Error == "" {
		h.Logger.Error(ctx, "Received empty error message in HandleScoreUpdateError")
		return nil, nil
	}

	// Notify the user in Discord
	err := h.RoundDiscord.GetScoreRoundManager().SendScoreUpdateError(payload.ScoreUpdateRequest.Participant, payload.Error)
	if err != nil {
		h.Logger.Error(ctx, "Failed to send score update error notification", attr.Error(err))
	}

	return nil, nil
}
