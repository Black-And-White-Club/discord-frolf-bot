package leaderboardhandlers

// import (
// 	"fmt"

// 	discordleaderboardevents "github.com/Black-And-White-Club/discord-frolf-bot/events/leaderboard"
// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// // -- Tag Swap --
// // HandleTagSwapRequest translates a Discord tag swap request to a backend request.
// func (h *LeaderboardHandlers) HandleTagSwapRequest(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagSwapRequest", attr.CorrelationIDFromMsg(msg))
// 	var discordPayload discordleaderboardevents.LeaderboardTagSwapRequestPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &discordPayload); err != nil {
// 		return nil, err
// 	}
// 	// Extract and validate.
// 	user1ID := discordPayload.User1ID         // User initiating the swap
// 	user2ID := discordPayload.User2ID         // Target user
// 	requestorID := discordPayload.RequestorID // Could be same as user1ID, or an admin
// 	channelID := discordPayload.ChannelID
// 	messageID := discordPayload.MessageID
// 	if user1ID == "" || user2ID == "" || requestorID == "" || channelID == "" {
// 		h.Logger.Error(ctx, "Invalid TagSwapRequest payload", attr.CorrelationIDFromMsg(msg),
// 			attr.String("user1_id", user1ID), attr.String("user2_id", user2ID), attr.String("requestor_id", requestorID))
// 		return nil, fmt.Errorf("invalid TagSwapRequest payload: missing required fields")
// 	}
// 	// Create the backend request.
// 	backendPayload := leaderboardevents.TagSwapRequestedPayload{
// 		RequestorID: requestorID, // The user who initiated the swap (or admin).
// 		TargetID:    user2ID,     // The user they want to swap with.
// 	}
// 	backendMsg, err := h.Helper.CreateResultMessage(msg, backendPayload, leaderboardevents.TagSwapRequested)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create backend message")
// 	}
// 	backendMsg.Metadata.Set("user_id", requestorID)
// 	backendMsg.Metadata.Set("channel_id", channelID)
// 	backendMsg.Metadata.Set("message_id", messageID)
// 	h.Logger.Info(ctx, "Successfully translated TagSwapRequest", attr.CorrelationIDFromMsg(msg))
// 	return []*message.Message{backendMsg}, nil
// }

// // HandleTagSwappedResponse translates a backend TagSwapProcessed event to a Discord response.
// func (h *LeaderboardHandlers) HandleTagSwappedResponse(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagSwappedResponse", attr.CorrelationIDFromMsg(msg))
// 	var backendPayload leaderboardevents.TagSwapProcessedPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &backendPayload); err != nil {
// 		return nil, err
// 	}
// 	userID := msg.Metadata.Get("user_id") // User who *initiated* the swap.
// 	channelID := msg.Metadata.Get("channel_id")
// 	messageID := msg.Metadata.Get("message_id")
// 	if userID == "" || channelID == "" {
// 		h.Logger.Error(ctx, "Missing required metadata for TagSwappedResponse", attr.CorrelationIDFromMsg(msg))
// 		return nil, fmt.Errorf("missing required metadata (user_id or channel_id)")
// 	}
// 	// Create the Discord response.
// 	discordPayload := discordleaderboardevents.LeaderboardTagSwappedPayload{
// 		User1ID:   backendPayload.RequestorID, // Use consistent field names.
// 		User2ID:   backendPayload.TargetID,
// 		ChannelID: channelID,
// 		MessageID: messageID,
// 	}
// 	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagSwappedTopic)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create discord message")
// 	}
// 	h.Logger.Info(ctx, "Successfully translated TagSwappedResponse", attr.CorrelationIDFromMsg(msg), attr.String("user_id", userID))
// 	return []*message.Message{discordMsg}, nil
// }

// // HandleTagSwapFailedResponse translates a backend TagSwapFailed to a Discord response.
// func (h *LeaderboardHandlers) HandleTagSwapFailedResponse(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagSwapFailedResponse", attr.CorrelationIDFromMsg(msg))
// 	var backendPayload leaderboardevents.TagSwapFailedPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &backendPayload); err != nil {
// 		return nil, err
// 	}
// 	// Get metadata from the *original request*.
// 	userID := msg.Metadata.Get("user_id") // User who *initiated* the swap.
// 	channelID := msg.Metadata.Get("channel_id")
// 	messageID := msg.Metadata.Get("message_id")
// 	if userID == "" || channelID == "" {
// 		h.Logger.Error(ctx, "Missing required metadata for TagSwapFailedResponse", attr.CorrelationIDFromMsg(msg))
// 		return nil, fmt.Errorf("missing required metadata (user_id or channel_id)")
// 	}
// 	discordPayload := discordleaderboardevents.LeaderboardTagSwapFailedPayload{
// 		User1ID:   backendPayload.RequestorID,
// 		User2ID:   backendPayload.TargetID,
// 		Reason:    backendPayload.Reason,
// 		ChannelID: channelID,
// 		MessageID: messageID,
// 	}
// 	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagSwapFailedTopic)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create discord message: %w", err)
// 	}
// 	h.Logger.Info(ctx, "Successfully translated TagSwapFailedResponse", attr.CorrelationIDFromMsg(msg), attr.String("user_id", userID))
// 	return []*message.Message{discordMsg}, nil
// }
