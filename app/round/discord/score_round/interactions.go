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
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleScoreButton opens the score submission modal when a user clicks "Enter Score"
func (srm *scoreRoundManager) HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	srm.logger.InfoContext(ctx, "Handling score button interaction", attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	return srm.operationWrapper(ctx, "handle_score_button", func(ctx context.Context) (ScoreRoundOperationResult, error) {
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
				attr.String("user_id", i.Member.User.ID))

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
			attr.String("user_id", i.Member.User.ID),
			attr.String("correlation_id", i.ID))

		err := srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "Submit Your Score",
				CustomID: fmt.Sprintf("submit_score_modal|%s|%s", roundID, i.Member.User.ID),
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
				attr.String("user_id", i.Member.User.ID),
				attr.String("correlation_id", i.ID))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		srm.logger.InfoContext(ctx, "Successfully opened score modal",
			attr.String("round_id", roundID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("correlation_id", i.ID))

		return ScoreRoundOperationResult{Success: "Score modal opened successfully"}, nil
	})
}

func (srm *scoreRoundManager) HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_submission")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	srm.logger.InfoContext(ctx, "Handling score submission", attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	return srm.operationWrapper(ctx, "handle_score_submission", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		data := i.ModalSubmitData()
		correlationID := i.ID

		// Validate modal custom ID
		parts := strings.Split(data.CustomID, "|")
		srm.logger.DebugContext(ctx, "Splitting modal custom ID", "customID", data.CustomID, "parts", parts)
		if len(parts) < 3 {
			err := fmt.Errorf("invalid modal custom ID: %s", data.CustomID)
			srm.logger.ErrorContext(ctx, "Invalid modal custom ID", attr.Error(err), attr.String("correlation_id", correlationID))

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

		// Parse round ID
		roundIDStr, userID := parts[1], parts[2]
		srm.logger.DebugContext(ctx, "Parsing round ID", "roundIDStr", roundIDStr)
		roundID, err := uuid.Parse(roundIDStr)
		if err != nil {
			err = fmt.Errorf("invalid round ID: %s", roundIDStr)
			srm.logger.ErrorContext(ctx, "Invalid round ID", attr.Error(err), attr.String("user_id", userID), attr.String("correlation_id", correlationID))

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

		// Acknowledge the interaction
		srm.logger.DebugContext(ctx, "Acknowledging interaction")
		err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to acknowledge score submission", attr.Error(err), attr.String("correlation_id", correlationID))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		// Extract score input
		var scoreStr string
		if len(data.Components) > 0 {
			if actionsRow, ok := data.Components[0].(*discordgo.ActionsRow); ok && len(actionsRow.Components) > 0 {
				if textInput, ok := actionsRow.Components[0].(*discordgo.TextInput); ok {
					scoreStr = strings.TrimSpace(textInput.Value)
				}
			} else if actionsRow, ok := data.Components[0].(discordgo.ActionsRow); ok && len(actionsRow.Components) > 0 {
				if textInput, ok := actionsRow.Components[0].(discordgo.TextInput); ok {
					scoreStr = strings.TrimSpace(textInput.Value)
				} else if textInput, ok := actionsRow.Components[0].(*discordgo.TextInput); ok {
					scoreStr = strings.TrimSpace(textInput.Value)
				}
			}
		}
		srm.logger.DebugContext(ctx, "Extracted score input", "scoreStr", scoreStr)

		if scoreStr == "" {
			err := fmt.Errorf("could not extract score input")
			srm.logger.ErrorContext(ctx, "Could not extract score input", attr.Error(err), attr.String("correlation_id", correlationID))

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
			err = fmt.Errorf("invalid score input: %s", scoreStr)
			srm.logger.ErrorContext(ctx, "Invalid score input", attr.Error(err), attr.String("correlation_id", correlationID))

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

		// Publish the score update request to backend
		payload := roundevents.ScoreUpdateRequestPayload{
			RoundID:     sharedtypes.RoundID(roundID),
			Participant: sharedtypes.DiscordID(userID),
			Score:       &scoreValue,
		}
		msg := message.NewMessage(correlationID, nil)
		msg.Metadata = message.Metadata{
			"correlation_id": correlationID,
			"topic":          roundevents.RoundScoreUpdateRequest,
		}

		srm.logger.DebugContext(ctx, "Creating result message")
		resultMsg, err := srm.helper.CreateResultMessage(msg, payload, roundevents.RoundScoreUpdateRequest)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to create result message", attr.Error(err), attr.String("correlation_id", correlationID))

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
		err = srm.publisher.Publish(roundevents.RoundScoreUpdateRequest, resultMsg)
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to publish score update request", attr.Error(err), attr.String("correlation_id", correlationID))

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

		// Add trace event for score submission
		tracePayload := map[string]interface{}{
			"round_id":       roundID,
			"participant_id": userID,
			"score":          score,
			"channel_id":     i.ChannelID,
			"status":         "score_submitted",
		}

		srm.logger.DebugContext(ctx, "Creating trace event message")
		traceMsg, err := srm.helper.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
		if err == nil {
			srm.logger.DebugContext(ctx, "Publishing trace event")
			srm.publisher.Publish(roundevents.RoundTraceEvent, traceMsg)
		}

		// Send confirmation to user
		srm.logger.DebugContext(ctx, "Sending confirmation to user")
		_, err = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("Your score of %d has been submitted! You'll receive a confirmation once it's processed.", score),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			srm.logger.ErrorContext(ctx, "Failed to send followup message", attr.Error(err))
			return ScoreRoundOperationResult{Error: err}, nil
		}

		srm.logger.InfoContext(ctx, "Score submission processed successfully",
			attr.RoundID("round_id", sharedtypes.RoundID(roundID)),
			attr.String("user_id", userID),
			attr.Int("score", score),
			attr.String("correlation_id", correlationID))

		return ScoreRoundOperationResult{Success: "Score submission processed successfully"}, nil
	})
}
