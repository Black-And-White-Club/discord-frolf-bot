package leaderboardhandlers

// import (
// 	"fmt"

// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain/types"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// // -- Tag Assignment --
// // HandleTagAssignRequest translates a Discord tag assignment request to a backend request.
// func (h *LeaderboardHandlers) HandleTagAssignRequest(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagAssignRequest", attr.CorrelationIDFromMsg(msg))
// 	var discordPayload discordleaderboardevents.LeaderboardTagAssignRequestPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &discordPayload); err != nil {
// 		return nil, err
// 	}
// 	// Extract and validate.
// 	targetUserID := discordPayload.TargetUserID
// 	requestorID := discordPayload.RequestorID // ID of the user *making* the request
// 	tagNumber := discordPayload.TagNumber
// 	channelID := discordPayload.ChannelID
// 	messageID := discordPayload.MessageID
// 	if targetUserID == "" || requestorID == "" || tagNumber <= 0 || channelID == "" {
// 		h.Logger.Error(ctx, "Invalid TagAssignRequest payload", attr.CorrelationIDFromMsg(msg),
// 			attr.String("target_user_id", targetUserID), attr.String("requestor_id", requestorID), attr.Int("tag_number", tagNumber))
// 		return nil, fmt.Errorf("invalid TagAssignRequest payload: missing required fields")
// 	}
// 	// Create the backend request.
// 	backendPayload := leaderboardevents.TagAssignmentRequestedPayload{
// 		DiscordID:  leaderboardtypes.DiscordID(targetUserID), // User *receiving* the tag
// 		TagNumber:  tagNumber,
// 		UpdateID:   messageID, // Use messageID as a unique identifier.
// 		Source:     "manual",  // Indicate this is a manual assignment.
// 		UpdateType: "new_tag", // Indicate this is a new tag.
// 	}
// 	backendMsg, err := h.Helper.CreateResultMessage(msg, backendPayload, leaderboardevents.LeaderboardTagAssignmentRequested)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create backend message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create backend message")
// 	}
// 	backendMsg.Metadata.Set("user_id", targetUserID)     // *Target* user ID
// 	backendMsg.Metadata.Set("requestor_id", requestorID) // ID of user making the request
// 	backendMsg.Metadata.Set("channel_id", channelID)
// 	backendMsg.Metadata.Set("message_id", messageID)
// 	h.Logger.Info(ctx, "Successfully translated TagAssignRequest", attr.CorrelationIDFromMsg(msg))
// 	return []*message.Message{backendMsg}, nil
// }

// // HandleTagAssignedResponse translates a backend TagAssigned event to a Discord response.
// func (h *LeaderboardHandlers) HandleTagAssignedResponse(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagAssignedResponse", attr.CorrelationIDFromMsg(msg))
// 	var backendPayload leaderboardevents.TagAssignedPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &backendPayload); err != nil {
// 		return nil, err
// 	}
// 	userID := msg.Metadata.Get("user_id")           // Target User
// 	requestorID := msg.Metadata.Get("requestor_id") // Requestor
// 	channelID := msg.Metadata.Get("channel_id")
// 	messageID := msg.Metadata.Get("message_id")
// 	// Create the Discord response.
// 	discordPayload := discordleaderboardevents.LeaderboardTagAssignedPayload{
// 		TargetUserID: string(backendPayload.DiscordID), // User who received the tag
// 		TagNumber:    backendPayload.TagNumber,
// 		ChannelID:    channelID, // Send back to the original channel
// 		MessageID:    messageID, // For updating the original message
// 	}
// 	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagAssignedTopic)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create discord message")
// 	}
// 	h.Logger.Info(ctx, "Successfully translated TagAssignedResponse", attr.CorrelationIDFromMsg(msg), attr.String("user_id", userID), attr.String("requestor_id", requestorID))
// 	return []*message.Message{discordMsg}, nil
// }

// // HandleTagAssignFailedResponse translates a backend TagAssignmentFailed event to a Discord response.
// func (h *LeaderboardHandlers) HandleTagAssignFailedResponse(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling TagAssignFailedResponse", attr.CorrelationIDFromMsg(msg))
// 	var backendPayload leaderboardevents.TagAssignmentFailedPayload
// 	if err := h.Helper.UnmarshalPayload(msg, &backendPayload); err != nil {
// 		return nil, err
// 	}
// 	// Get the metadata for the original requestor
// 	userID := msg.Metadata.Get("user_id")           // Target User
// 	requestorID := msg.Metadata.Get("requestor_id") // Requestor
// 	channelID := msg.Metadata.Get("channel_id")
// 	messageID := msg.Metadata.Get("message_id")
// 	// Create the Discord response
// 	discordPayload := discordleaderboardevents.LeaderboardTagAssignFailedPayload{
// 		TargetUserID: string(backendPayload.DiscordID),
// 		TagNumber:    backendPayload.TagNumber,
// 		Reason:       backendPayload.Reason,
// 		ChannelID:    channelID, // Send back to original channel
// 		MessageID:    messageID, // For updating the original message.
// 	}
// 	discordMsg, err := h.Helper.CreateResultMessage(msg, discordPayload, discordleaderboardevents.LeaderboardTagAssignFailedTopic)
// 	if err != nil {
// 		h.Logger.Error(ctx, "Failed to create discord message", attr.CorrelationIDFromMsg(msg), attr.Error(err))
// 		return nil, fmt.Errorf("failed to create discord message: %w", err)
// 	}
// 	h.Logger.Info(ctx, "Successfully translated TagAssignFailedResponse", attr.CorrelationIDFromMsg(msg), attr.String("user_id", userID), attr.String("requestor_id", requestorID))
// 	return []*message.Message{discordMsg}, nil
// }
