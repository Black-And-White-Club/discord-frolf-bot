package scorehandlers

import (
	"fmt"

	discordscoreevents "github.com/Black-And-White-Club/discord-frolf-bot/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleScoreUpdateRequest translates a Discord score update request to a backend request.
func (h *ScoreHandlers) HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling ScoreUpdateRequest", attr.CorrelationIDFromMsg(msg))
	var discordPayload discordscoreevents.ScoreUpdateRequestPayload
	if err := h.unmarshalPayload(msg, &discordPayload); err != nil {
		return nil, err
	}
	userID := discordPayload.UserID //From common metadata assumption
	channelID := discordPayload.ChannelID
	messageID := discordPayload.MessageID
	// Validate.
	if discordPayload.RoundID == "" || discordPayload.Participant == "" || discordPayload.Score == nil {
		h.Logger.Error(ctx, "Invalid ScoreUpdateRequest payload", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("invalid payload: missing round_id, participant, or score")
	}
	// Translate to backend payload.
	backendPayload := scoreevents.ScoreUpdateRequestPayload{
		CommonMetadata: events.CommonMetadata{
			Domain:    "score",
			EventName: "score.update.request",
		},
		RoundID:     discordPayload.RoundID,
		Participant: discordPayload.Participant,
		Score:       discordPayload.Score, // Already a pointer.
		TagNumber:   discordPayload.TagNumber,
	}
	backendMsg, err := h.createResultMessage(msg, backendPayload, scoreevents.ScoreUpdateRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, err
	}
	backendMsg.Metadata.Set("user_id", userID) // Use for response routing and sending message to channel/user
	backendMsg.Metadata.Set("channel_id", channelID)
	backendMsg.Metadata.Set("message_id", messageID)
	h.Logger.Info(ctx, "Successfully translated ScoreUpdateRequest", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

// HandleScoreUpdateResponse translates a backend score update response to a Discord response.
func (h *ScoreHandlers) HandleScoreUpdateResponse(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling ScoreUpdateResponse", attr.CorrelationIDFromMsg(msg))
	var backendPayload scoreevents.ScoreUpdateResponsePayload //  Backend response payload.
	if err := h.unmarshalPayload(msg, &backendPayload); err != nil {
		return nil, err
	}
	userID := msg.Metadata.Get("user_id")
	channelID := msg.Metadata.Get("channel_id")
	messageID := msg.Metadata.Get("message_id")
	if userID == "" || channelID == "" {
		h.Logger.Error(ctx, "Missing required metadata for ScoreUpdateResponse", attr.CorrelationIDFromMsg(msg))
		return nil, fmt.Errorf("missing required metadata (user_id or channel_id)")
	}
	// Create the Discord response payload.
	discordPayload := discordscoreevents.ScoreUpdateResponsePayload{
		CommonMetadata: events.CommonMetadata{
			Domain:    "score",
			EventName: "score.update.response",
		},
		Success:     backendPayload.Success,
		RoundID:     backendPayload.RoundID,
		Participant: backendPayload.Participant,
		Error:       backendPayload.Error,
		UserID:      userID,
		ChannelID:   channelID,
		MessageID:   messageID,
	}
	discordMsg, err := h.createResultMessage(msg, discordPayload, discordscoreevents.ScoreUpdateResponseTopic)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to create discord message: %w", err)
	}
	h.Logger.Info(ctx, "Successfully translated ScoreUpdateResponse", attr.CorrelationIDFromMsg(msg), attr.String("user_id", userID))
	return []*message.Message{discordMsg}, nil
}
