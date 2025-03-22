package scoreround

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// HandleScoreButton opens the score submission modal when a user clicks "Enter Score"
func (srm *scoreRoundManager) HandleScoreButton(ctx context.Context, i *discordgo.InteractionCreate) {
	customIDParts := strings.Split(i.MessageComponentData().CustomID, "|")
	if len(customIDParts) < 2 {
		srm.logger.Error(ctx, "Invalid CustomID for score button", attr.String("custom_id", i.MessageComponentData().CustomID))
		return
	}

	roundID := customIDParts[1]

	// Check if the round is finalized
	if strings.Contains(i.MessageComponentData().CustomID, "finalized") {
		srm.logger.Info(ctx, "Attempted score submission on a finalized round", attr.String("round_id", roundID), attr.String("user_id", i.Member.User.ID))

		srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ This round has been finalized. Score updates are no longer allowed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	srm.logger.Info(ctx, "Opening score submission modal",
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
		srm.logger.Error(ctx, "Failed to open score modal",
			attr.Error(err),
			attr.String("round_id", roundID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("correlation_id", i.ID))
	} else {
		srm.logger.Info(ctx, "Successfully opened score modal",
			attr.String("round_id", roundID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("correlation_id", i.ID))
	}

}

// HandleScoreSubmission processes the score entered in the modal
func (srm *scoreRoundManager) HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	correlationID := i.ID

	parts := strings.Split(data.CustomID, "|")
	if len(parts) < 3 {
		srm.logger.Error(ctx, "Invalid modal custom ID",
			attr.String("custom_id", data.CustomID),
			attr.String("correlation_id", correlationID))

		srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Something went wrong with your submission. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	roundIDStr, userID := parts[1], parts[2]

	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		srm.logger.Error(ctx, "Invalid round ID",
			attr.String("round_id_str", roundIDStr),
			attr.String("user_id", userID),
			attr.Error(err),
			attr.String("correlation_id", correlationID))

		srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid round information. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Acknowledge the interaction first to avoid timeout
	err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		srm.logger.Error(ctx, "Failed to acknowledge score submission",
			attr.Error(err),
			attr.String("correlation_id", correlationID))
		return
	}

	// Fix the type assertion here - handle both pointer and value types
	var scoreStr string
	if len(data.Components) > 0 {
		// Try to handle both pointer and value types
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

	if scoreStr == "" {
		srm.logger.Error(ctx, "Could not extract score input",
			attr.String("correlation_id", correlationID))

		srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Could not read your score. Please try again.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	score, err := strconv.Atoi(scoreStr)
	if err != nil {
		srm.logger.Error(ctx, "Invalid score input",
			attr.String("score_input", scoreStr),
			attr.Error(err),
			attr.String("correlation_id", correlationID))

		srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Invalid score. Please enter a valid number (e.g., -3, 0, +5).",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	srm.logger.Info(ctx, "Processing score submission",
		attr.Int64("round_id", roundID),
		attr.String("user_id", userID),
		attr.Int("score", score),
		attr.String("correlation_id", correlationID))

	// Publish score update request to backend
	payload := roundevents.ScoreUpdateRequestPayload{
		RoundID:     roundtypes.ID(roundID),
		Participant: roundtypes.UserID(userID),
		Score:       &score,
	}
	msg := message.NewMessage(correlationID, nil)
	msg.Metadata = message.Metadata{
		"correlation_id": correlationID,
		"topic":          roundevents.RoundScoreUpdateRequest,
	}

	resultMsg, err := srm.helper.CreateResultMessage(msg, payload, roundevents.RoundScoreUpdateRequest)
	if err != nil {
		srm.logger.Error(ctx, "Failed to create result message",
			attr.Error(err),
			attr.String("correlation_id", correlationID))

		srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Something went wrong while submitting your score. Please try again later.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	err = srm.publisher.Publish(roundevents.RoundScoreUpdateRequest, resultMsg)
	if err != nil {
		srm.logger.Error(ctx, "Failed to publish score update request",
			attr.Error(err),
			attr.String("correlation_id", correlationID))

		srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Failed to submit your score. Please try again later.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	// Add trace event for score submission
	tracePayload := map[string]interface{}{
		"round_id":       roundID,
		"participant_id": userID,
		"score":          score,
		"channel_id":     i.ChannelID,
		"status":         "score_submitted",
	}

	traceMsg, err := srm.helper.CreateResultMessage(msg, tracePayload, roundevents.RoundTraceEvent)
	if err == nil {
		srm.publisher.Publish(roundevents.RoundTraceEvent, traceMsg)
	}

	// Send confirmation to user
	srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Your score of %d has been submitted! You'll receive a confirmation once it's processed.", score),
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	srm.logger.Info(ctx, "Score submission processed successfully",
		attr.Int64("round_id", roundID),
		attr.String("user_id", userID),
		attr.Int("score", score),
		attr.String("correlation_id", correlationID))
}
