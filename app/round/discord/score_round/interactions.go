package scoreround

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleScoreButton opens the score submission modal when a user clicks "Enter Score"
func (srm *scoreRoundManager) HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return ScoreRoundOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	srm.logger.InfoContext(ctx, "Handling score button interaction", attr.UserID(sharedtypes.DiscordID(userID)))

	return srm.operationWrapper(ctx, "handle_score_button", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		// Get user from either Member or direct User field
		var user *discordgo.User
		if i.Member != nil && i.Member.User != nil {
			user = i.Member.User
		} else if i.User != nil {
			user = i.User
		} else {
			return ScoreRoundOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
		}

		customIDParts := strings.Split(i.MessageComponentData().CustomID, "|")
		if len(customIDParts) < 2 {
			err := fmt.Errorf("invalid custom ID for score button: %s", i.MessageComponentData().CustomID)
			srm.logger.ErrorContext(ctx, "Invalid CustomID for score button", attr.Error(err))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		roundID := customIDParts[1]

		// Check if the round is finalized
		if strings.Contains(i.MessageComponentData().CustomID, "finalized") {
			srm.logger.InfoContext(ctx, "Attempted score submission on a finalized round",
				attr.String("round_id", roundID),
				attr.String("user_id", user.ID))

			err := srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ This round has been finalized. Score updates are no longer allowed.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Success: "Finalized round interaction handled"}, nil
		}

		srm.logger.InfoContext(ctx, "Opening score submission modal",
			attr.String("round_id", roundID),
			attr.String("user_id", user.ID),
			attr.String("correlation_id", i.ID))

		err := srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "Submit Your Score",
				CustomID: fmt.Sprintf("submit_score_modal|%s|%s", roundID, user.ID),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "score_input",
								Label:       "Enter your score (e.g., -3, 0, +5)",
								Style:       discordgo.TextInputShort,
								Required:    true,
								Placeholder: "Enter your disc golf score",
							},
						},
					},
				},
			},
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to open score modal",
				attr.Error(err),
				attr.String("round_id", roundID),
				attr.String("user_id", user.ID),
				attr.String("correlation_id", i.ID))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		srm.logger.InfoContext(ctx, "Successfully opened score modal",
			attr.String("round_id", roundID),
			attr.String("user_id", user.ID),
			attr.String("correlation_id", i.ID))

		return ScoreRoundOperationResult{Success: "Score modal opened successfully"}, nil
	})
}

func (srm *scoreRoundManager) HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_submission")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return ScoreRoundOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	srm.logger.InfoContext(ctx, "Handling score submission", attr.UserID(sharedtypes.DiscordID(userID)))

	return srm.operationWrapper(ctx, "handle_score_submission", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		data := i.ModalSubmitData()

		// Validate modal custom ID
		parts := strings.Split(data.CustomID, "|")
		srm.logger.DebugContext(ctx, "Splitting modal custom ID", "customID", data.CustomID, "parts", parts)
		if len(parts) < 3 {
			err := fmt.Errorf("invalid modal custom ID: %s", data.CustomID)
			srm.logger.ErrorContext(ctx, "Invalid modal custom ID", attr.Error(err))

			err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Something went wrong with your submission. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to respond to interaction", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid modal custom ID")}, nil
		}

		// Parse round ID and User ID from custom ID
		roundIDStr, userIDFromModal := parts[1], parts[2]
		srm.logger.DebugContext(ctx, "Parsing round ID and user ID from custom ID", "roundIDStr", roundIDStr, "userID", userIDFromModal)
		roundID, err := uuid.Parse(roundIDStr)
		if err != nil {
			err = fmt.Errorf("invalid round ID: %s", roundIDStr)
			srm.logger.ErrorContext(ctx, "Invalid round ID format", attr.Error(err), attr.String("user_id", userIDFromModal))

			err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Invalid round information. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to respond to interaction", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid round ID")}, nil
		}

		// Acknowledge the interaction as deferred ephemeral message update
		srm.logger.DebugContext(ctx, "Acknowledging score submission interaction")
		err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to acknowledge score submission", attr.Error(err))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		// Extract score input from modal components
		var scoreStr string
		if len(data.Components) > 0 {
			// Handle both pointer and value types for ActionsRow
			var actionsRow *discordgo.ActionsRow
			switch component := data.Components[0].(type) {
			case *discordgo.ActionsRow:
				actionsRow = component
			case discordgo.ActionsRow:
				actionsRow = &component
			}

			if actionsRow != nil && len(actionsRow.Components) > 0 {
				// Handle both pointer and value types for TextInput
				switch textComponent := actionsRow.Components[0].(type) {
				case *discordgo.TextInput:
					scoreStr = strings.TrimSpace(textComponent.Value)
				case discordgo.TextInput:
					scoreStr = strings.TrimSpace(textComponent.Value)
				}
			}
		}
		srm.logger.DebugContext(ctx, "Extracted score input", "scoreStr", scoreStr)

		if scoreStr == "" {
			err := fmt.Errorf("could not extract score input from modal")
			srm.logger.ErrorContext(ctx, "Could not extract score input", attr.Error(err))

			// Send followup error message
			_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Could not read your score. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to send followup message", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("could not extract score input")}, nil
		}

		// Convert score string to integer
		srm.logger.DebugContext(ctx, "Converting score string to integer", "scoreStr", scoreStr)
		score, err := strconv.Atoi(scoreStr)
		if err != nil {
			err = fmt.Errorf("invalid score input format: %s", scoreStr)
			srm.logger.ErrorContext(ctx, "Invalid score input format", attr.Error(err), attr.String("score_str", scoreStr))

			// Send followup error message
			_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Invalid score. Please enter a valid number (e.g., -3, 0, +5).",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to send followup message", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid score input")}, nil
		}

		// Store the score as sharedtypes.Score
		scoreValue := sharedtypes.Score(score)

		// Prepare the payload for the backend ScoreUpdateRequest
		payload := roundevents.ScoreUpdateRequestPayload{
			RoundID:     sharedtypes.RoundID(roundID),
			Participant: sharedtypes.DiscordID(userIDFromModal),
			Score:       &scoreValue, // Pass pointer to score value
		}

		msg := message.NewMessage(watermill.NewUUID(), nil)
		msg.Metadata.Set("topic", roundevents.RoundScoreUpdateRequest) // Set the topic in metadata

		// Add nil check for Message before accessing ID
		if i.Message != nil {
			msg.Metadata.Set("discord_message_id", i.Message.ID) // Set the original Discord message ID here!
		}

		srm.logger.DebugContext(ctx, "Creating result message for ScoreUpdateRequest with metadata")
		// Use the message created with metadata. CreateResultMessage likely copies metadata.
		resultMsg, err := srm.helper.CreateResultMessage(msg, payload, roundevents.RoundScoreUpdateRequest)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to create result message for ScoreUpdateRequest", attr.Error(err))

			// Send followup error message
			_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Something went wrong while submitting your score. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to send followup message", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("failed to create result message")}, nil
		}

		srm.logger.DebugContext(ctx, "Publishing score update request")
		// Publish the message
		err = srm.publisher.Publish(roundevents.RoundScoreUpdateRequest, resultMsg)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to publish score update request", attr.Error(err))

			// Send followup error message
			_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "Failed to submit your score. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				srm.logger.ErrorContext(ctx, "Failed to send followup message", attr.Error(err))
				return ScoreRoundOperationResult{Error: err}, nil
			}
			return ScoreRoundOperationResult{Error: fmt.Errorf("failed to publish message")}, nil
		}

		// Add trace event for score submission (Optional, metadata includes message ID now)
		tracePayload := map[string]interface{}{
			"round_id":       roundID,
			"participant_id": userIDFromModal,
			"score":          score,
			"channel_id":     i.ChannelID,
			"status":         "score_submitted",
			// The message ID is already in the metadata copied from 'msg'
		}

		srm.logger.DebugContext(ctx, "Creating trace event message")
		// Create trace message. Metadata from 'msg' (including correlation_id and discord_message_id) should be copied by CreateResultMessage.
		traceMsg, err := srm.helper.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
		if err == nil {
			srm.logger.DebugContext(ctx, "Publishing trace event")
			srm.publisher.Publish(roundevents.RoundTraceEvent, traceMsg)
		} else {
			srm.logger.ErrorContext(ctx, "Failed to create trace event message for score submission", attr.Error(err))
		}

		// Send confirmation to user via followup message
		srm.logger.DebugContext(ctx, "Sending confirmation to user")
		_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("Your score of %d has been submitted! You'll receive a confirmation once it's processed.", score),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to send followup confirmation message", attr.Error(err))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		// Add nil check for Message before accessing ID in logs
		var messageID string
		if i.Message != nil {
			messageID = i.Message.ID
		}

		srm.logger.InfoContext(ctx, "Score submission processed successfully, request published",
			attr.RoundID("round_id", sharedtypes.RoundID(roundID)),
			attr.String("user_id", userIDFromModal),
			attr.Int("score", score),
			attr.String("discord_message_id", messageID), // Log the original message ID for context
		)

		return ScoreRoundOperationResult{Success: "Score submission processed successfully"}, nil
	})
}
