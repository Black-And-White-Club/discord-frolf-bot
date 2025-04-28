package updateround

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (urm *updateRoundManager) HandleEditRoundButton(ctx context.Context, i *discordgo.InteractionCreate) (UpdateRoundOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_edit_round")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	urm.logger.InfoContext(ctx, "Handling edit round button interaction",
		attr.String("interaction_id", i.ID),
		attr.String("custom_id", i.MessageComponentData().CustomID),
		attr.UserID(sharedtypes.DiscordID(i.Member.User.ID)))

	return urm.operationWrapper(ctx, "handle_edit_round_button", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			err := fmt.Errorf("invalid custom_id format: %s", customID)
			urm.logger.ErrorContext(ctx, err.Error())
			return UpdateRoundOperationResult{Error: err}, nil
		}

		// Parse UUID properly
		roundUUID, err := uuid.Parse(parts[1])
		if err != nil {
			err := fmt.Errorf("invalid UUID for round ID: %w", err)
			urm.logger.ErrorContext(ctx, err.Error())
			return UpdateRoundOperationResult{Error: err}, nil
		}
		roundID := sharedtypes.RoundID(roundUUID)

		urm.logger.InfoContext(ctx, "Opening modal for round edit", attr.RoundID("round_id", roundID))

		// Send the update round modal
		result, err := urm.SendUpdateRoundModal(ctx, i)
		if err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}
		if result.Error != nil {
			return result, nil
		}
		return UpdateRoundOperationResult{Success: "modal sent"}, nil
	})
}
