package leaderboardhandlers

import (
	"fmt"

	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/events"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetTagByDiscordID handles a request *from Discord* to get a user's tag.
func (h *LeaderboardHandlers) HandleGetTagByDiscordID(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling GetTagByDiscordID request", attr.CorrelationIDFromMsg(msg))
	// 1. Unmarshal the *Discord* request payload.
	var discordPayload discordleaderboardevents.LeaderboardTagAvailabilityRequestPayload //  Payload type
	if err := h.Helper.UnmarshalPayload(msg, &discordPayload); err != nil {
		return nil, err // unmarshalPayload already logs
	}
	// 2. Extract relevant data from the Discord payload.
	userID := discordPayload.UserID
	// 4. Create the *backend* request payload.
	backendPayload := leaderboardevents.GetTagByDiscordIDRequestPayload{
		DiscordID: leaderboardtypes.DiscordID(userID),
	}
	// 5. Create the backend message, setting metadata.
	backendMsg, err := h.Helper.CreateResultMessage(msg, backendPayload, leaderboardevents.GetTagByDiscordIDRequest)
	if err != nil {
		h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to create backend message")
	}
	h.Logger.Info(ctx, "Successfully translated GetTagByDiscordID request", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{backendMsg}, nil
}

// HandleGetTagByDiscordIDResponse translates a backend tag response to a Discord response.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDResponse(msg *message.Message) ([]*message.Message, error) {
	ctx := msg.Context()
	h.Logger.Info(ctx, "Handling GetTagByDiscordIDResponse", attr.CorrelationIDFromMsg(msg))
	// 1. Unmarshal the *backend* response payload.
	var backendPayload leaderboardevents.GetTagByDiscordIDResponsePayload
	if err := h.Helper.UnmarshalPayload(msg, &backendPayload); err != nil {
		return nil, err // unmarshalPayload already logs
	}
	// 3. Create the *Discord* response payload.
	discordPayload := discordleaderboardevents.LeaderboardTagAvailabilityResponsePayload{
		CommonMetadata: events.CommonMetadata{},
		TagNumber:      backendPayload.TagNumber,
	}
	// 4. Create the Discord message.
	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagAvailabilityResponseTopic)
	if err != nil {
		// Log with context *here*.
		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
		return nil, fmt.Errorf("failed to create discord message")
	}
	h.Logger.Info(ctx, "Successfully translated GetTagByDiscordIDResponse", attr.CorrelationIDFromMsg(msg))
	return []*message.Message{discordMsg}, nil
}
