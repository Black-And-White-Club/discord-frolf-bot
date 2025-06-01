package finalizeround

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// TransformRoundToFinalizedScorecard transforms the round event embed into a finalized scorecard format
// showing participants with their final scores and modifying the UI to indicate the round is finalized
func (frm *finalizeRoundManager) TransformRoundToFinalizedScorecard(payload roundevents.RoundFinalizedEmbedUpdatePayload) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	var embed *discordgo.MessageEmbed
	var components []discordgo.MessageComponent
	var err error

	// Add nil check for the payload itself
	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		return nil, nil, fmt.Errorf("invalid payload: round ID is empty")
	}

	_, err = frm.operationWrapper(context.Background(), "TransformRoundToFinalizedScorecard", func(ctx context.Context) (FinalizeRoundOperationResult, error) {
		// Log incoming payload for debugging
		frm.logger.InfoContext(ctx, "Processing round finalization",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("title", string(payload.Title)))

		// Create participant fields with appropriate nil checks
		participantFields := make([]*discordgo.MessageEmbedField, 0, len(payload.Participants))

		for i, participant := range payload.Participants {
			// Check if UserID is valid
			if participant.UserID == "" {
				frm.logger.WarnContext(ctx, "Skipping participant with empty UserID",
					attr.Int("index", i))
				continue
			}

			var username string
			user, err := frm.session.User(string(participant.UserID))
			if err != nil {
				frm.logger.WarnContext(ctx, "Failed to get participant info, skipping participant",
					attr.Error(err), attr.String("user_id", string(participant.UserID)))
				// Skip this participant instead of adding fallback name
				continue
			} else {
				username = user.Username
				// Only try to get member info if we successfully got the user
				if member, err := frm.session.GuildMember(frm.config.Discord.GuildID, string(participant.UserID)); err == nil && member != nil && member.Nick != "" {
					username = member.Nick
				}
			}

			scoreDisplay := "Score: --"
			if participant.Score != nil {
				scoreDisplay = fmt.Sprintf("Score: %+d", *participant.Score)
			}

			participantFields = append(participantFields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("🏌️ %s", username),
				Value:  scoreDisplay,
				Inline: true,
			})
		}

		// Handle nil location
		locationStr := ""
		if payload.Location != nil {
			locationStr = string(*payload.Location)
		}

		// Start with base fields - StartTime and Location
		embedFields := []*discordgo.MessageEmbedField{}

		// Add StartTime field if available
		if payload.StartTime != nil {
			embedFields = append(embedFields, &discordgo.MessageEmbedField{
				Name:  "📅 Started",
				Value: fmt.Sprintf("<t:%d:f>", time.Time(*payload.StartTime).Unix()),
			})
		}

		// Add Location field
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:  "📍 Location",
			Value: locationStr,
		})

		// Add participant fields
		embedFields = append(embedFields, participantFields...)

		// Construct the embed with defensive programming
		title := "Round Finalized"
		if payload.Title != "" {
			title = fmt.Sprintf("**%s** - Round Finalized", payload.Title)
		}

		embed = &discordgo.MessageEmbed{
			Title:       title,
			Description: fmt.Sprintf("Round at %s has been finalized. Admin/Editor access required for score updates.", locationStr),
			Color:       0x0000FF, // Blue for finalized round
			Fields:      embedFields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Round has been finalized. Only admins/editors can update scores.",
			},
			Timestamp: time.Now().Format(time.RFC3339), // Current time when finalized
		}

		// Generate a safe custom ID for the button - use the format expected by tests
		buttonID := fmt.Sprintf("round_enter_score_finalized|round-%d", payload.RoundID)

		// Keep the same button but with modified text to indicate admin requirement
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Admin/Editor Score Update",
						Style:    discordgo.DangerButton,
						CustomID: buttonID,
						Emoji:    &discordgo.ComponentEmoji{Name: "🔒"},
					},
				},
			},
		}

		// Log successful transformation
		frm.logger.InfoContext(ctx, "Successfully transformed round to finalized scorecard",
			attr.RoundID("round_id", payload.RoundID))

		return FinalizeRoundOperationResult{Success: true}, nil
	})
	// Add additional error handling outside the operation wrapper
	if err != nil {
		frm.logger.ErrorContext(context.Background(), "Failed to transform round to finalized scorecard",
			attr.Error(err), attr.RoundID("round_id", payload.RoundID))
		return nil, nil, fmt.Errorf("failed to transform round to finalized scorecard: %w", err)
	}

	// Add a safety check to ensure we're not returning a nil embed
	if embed == nil {
		frm.logger.ErrorContext(context.Background(), "Embed is nil after transformation",
			attr.RoundID("round_id", payload.RoundID))
		return nil, nil, fmt.Errorf("transformed embed is nil for round %s", payload.RoundID)
	}

	return embed, components, nil
}
