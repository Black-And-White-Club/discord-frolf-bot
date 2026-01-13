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

// normalizeParticipantInput parses a Discord user mention and returns the user ID.
//
// It accepts input strings in the following formats:
//   - "<@1234>"   => "1234"
//   - "<@!1234>"  => "1234"
//   - "1234"      => "1234"
//   - "  <@1234>  " => "1234"
//   - ""          => ""
//   - "some text" => "some text"
//
// If the input is a Discord mention (e.g., "<@1234>" or "<@!1234>"), it extracts and returns the user ID.
// Otherwise, it returns the trimmed input string.
// Edge cases:
//   - Input with extra whitespace is trimmed.
//   - Input that does not match the mention format is returned as-is (trimmed).
func normalizeParticipantInput(in string) string {
	// Normalize input by trimming whitespace and extracting user ID from Discord mention formats.
	// Go's string functions are UTF-8 safe, so Unicode and edge cases (e.g., extra spaces, unusual characters)
	// are handled correctly by TrimSpace and string prefix/suffix checks.
	in = strings.TrimSpace(in)
	if in == "" {
		return in
	}
	if strings.HasPrefix(in, "<@") && strings.HasSuffix(in, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(in, "<@"), ">")
		inner = strings.TrimPrefix(inner, "!")
		return inner
	}
	return in
}

// HandleScoreSubmission entry point chooses bulk vs single based on custom ID.
func (srm *scoreRoundManager) HandleScoreSubmission(ctx context.Context, i *discordgo.InteractionCreate) (ScoreRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_score_submission")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal")

	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	} else {
		return ScoreRoundOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	srm.logger.InfoContext(ctx, "Handling score submission", attr.UserID(sharedtypes.DiscordID(userID)))

	return srm.operationWrapper(ctx, "handle_score_submission", func(ctx context.Context) (ScoreRoundOperationResult, error) {
		data := i.ModalSubmitData()
		parts := strings.Split(data.CustomID, "|")
		if len(parts) < 3 {
			_ = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Something went wrong with your submission. Please try again.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid modal custom ID")}, nil
		}
		roundIDStr, userIDFromModal := parts[1], parts[2]
		if strings.HasPrefix(data.CustomID, submitBulkOverridePrefix) {
			return srm.handleBulkScoreSubmission(ctx, i, roundIDStr, userIDFromModal)
		}

		roundID, err := uuid.Parse(roundIDStr)
		if err != nil {
			_ = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "Invalid round information. Please try again.", Flags: discordgo.MessageFlagsEphemeral},
			})
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid round ID")}, nil
		}

		if err = srm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredChannelMessageWithSource, Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral}}); err != nil {
			return ScoreRoundOperationResult{Error: err}, nil
		}

		var scoreStr string
		participantID := userIDFromModal
		for _, comp := range data.Components {
			var row *discordgo.ActionsRow
			switch c := comp.(type) {
			case *discordgo.ActionsRow:
				row = c
			case discordgo.ActionsRow:
				row = &c
			}
			if row == nil {
				continue
			}
			for _, inner := range row.Components {
				switch ti := inner.(type) {
				case *discordgo.TextInput:
					switch ti.CustomID {
					case "score_input":
						scoreStr = strings.TrimSpace(ti.Value)
					case "participant_input":
						if p := strings.TrimSpace(ti.Value); p != "" {
							participantID = normalizeParticipantInput(p)
						}
					}
				case discordgo.TextInput:
					switch ti.CustomID {
					case "score_input":
						scoreStr = strings.TrimSpace(ti.Value)
					case "participant_input":
						if p := strings.TrimSpace(ti.Value); p != "" {
							participantID = normalizeParticipantInput(p)
						}
					}
				}
			}
		}

		if scoreStr == "" {
			_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Could not read your score. Please try again.", Flags: discordgo.MessageFlagsEphemeral})
			return ScoreRoundOperationResult{Error: fmt.Errorf("could not extract score input")}, nil
		}

		scoreVal, err := strconv.Atoi(scoreStr)
		if err != nil {
			_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Invalid score. Please enter a valid number (e.g., -3, 0, +5).", Flags: discordgo.MessageFlagsEphemeral})
			return ScoreRoundOperationResult{Error: fmt.Errorf("invalid score input")}, nil
		}
		if scoreVal < scoreMin || scoreVal > scoreMax {
			_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: fmt.Sprintf("Invalid score: %d. Scores must be between %d and +%d.", scoreVal, scoreMin, scoreMax), Flags: discordgo.MessageFlagsEphemeral})
			return ScoreRoundOperationResult{Error: fmt.Errorf("score out of range")}, nil
		}

		// Build and publish the canonical round domain score update request.
		// The Discord code acts as a transport adapter: validate/auth has already
		// happened, so we publish the domain event directly and preserve transport
		// IDs in metadata for idempotency and embed updates.
		scoreValue := sharedtypes.Score(scoreVal)
		scorePayload := &roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:   sharedtypes.GuildID(i.GuildID),
			RoundID:   sharedtypes.RoundID(roundID),
			UserID:    sharedtypes.DiscordID(participantID),
			Score:     &scoreValue,
			ChannelID: i.ChannelID,
			MessageID: "",
		}
		if i.Message != nil {
			scorePayload.MessageID = i.Message.ID
		}

		scoreMsg := message.NewMessage(watermill.NewUUID(), nil)
		if scoreMsg.Metadata == nil {
			scoreMsg.Metadata = message.Metadata{}
		}
		// Publish the canonical domain topic so round handlers receive it directly
		scoreMsg.Metadata.Set("topic", roundevents.RoundScoreUpdateRequestedV1)
		if i.GuildID != "" {
			scoreMsg.Metadata.Set("guild_id", i.GuildID)
		}
		if i.Message != nil {
			scoreMsg.Metadata.Set("discord_message_id", i.Message.ID)
			scoreMsg.Metadata.Set("message_id", i.Message.ID)
		}
		if i.ChannelID != "" {
			scoreMsg.Metadata.Set("channel_id", i.ChannelID)
		}

		resultScoreMsg, err := srm.helper.CreateResultMessage(scoreMsg, scorePayload, roundevents.RoundScoreUpdateRequestedV1)
		if err != nil {
			_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Something went wrong while submitting your score. Please try again later.", Flags: discordgo.MessageFlagsEphemeral})
			return ScoreRoundOperationResult{Error: fmt.Errorf("failed to create result message")}, nil
		}
		if err = srm.publisher.Publish(roundevents.RoundScoreUpdateRequestedV1, resultScoreMsg); err != nil {
			_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: "Failed to submit your score. Please try again later.", Flags: discordgo.MessageFlagsEphemeral})
			return ScoreRoundOperationResult{Error: fmt.Errorf("failed to publish message")}, nil
		}

		_, _ = srm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{Content: fmt.Sprintf("Your score of %d has been submitted!", scoreVal), Flags: discordgo.MessageFlagsEphemeral})
		return ScoreRoundOperationResult{Success: "Score submission processed successfully"}, nil
	})
}
