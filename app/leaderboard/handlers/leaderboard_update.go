package handlers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleBatchTagAssigned handles batch tag assignment completions and requests a full leaderboard snapshot.
func (h *LeaderboardHandlers) HandleBatchTagAssigned(ctx context.Context,
	payload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Handling batch tag assigned event")

	batchPayload := payload

	guildID := string(batchPayload.GuildID)
	assignmentCount := batchPayload.AssignmentCount
	if assignmentCount == 0 && len(batchPayload.Assignments) > 0 {
		assignmentCount = len(batchPayload.Assignments)
	}

	if assignmentCount == 0 {
		h.logger.WarnContext(ctx, "Received empty batch assignment data",
			attr.String("guild_id", guildID))
		return []handlerwrapper.Result{}, nil
	}

	// Retrieve correlation ID from context to update the original interaction
	var correlationID string
	if val := ctx.Value("correlation_id"); val != nil {
		if corr, ok := val.(string); ok {
			correlationID = corr
		}
	}

	// Update the interaction if this was initiated by a user command
	if correlationID != "" {
		successMessage := fmt.Sprintf("✅ Successfully assigned %d tags!", assignmentCount)
		// We use fmt to maximize compatibility since we need to import fmt
		if h.service != nil {
			claimTagManager := h.service.GetClaimTagManager()
			if claimTagManager != nil {
				res, err := claimTagManager.UpdateInteractionResponse(ctx, correlationID, successMessage)
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to update Discord interaction for batch success",
						attr.String("correlation_id", correlationID),
						attr.Error(err))
				} else {
					h.logger.InfoContext(ctx, "Successfully updated Discord interaction for batch success",
						attr.String("correlation_id", correlationID),
						attr.String("result", fmt.Sprintf("%v", res.Success)))
				}
			}
		}
	}

	// Batch assignment payloads are incremental changes (often a single assignment),
	// so request a full snapshot and render Discord from the canonical leaderboard response.
	return []handlerwrapper.Result{{
		Topic: leaderboardevents.GetLeaderboardRequestedV1,
		Payload: &leaderboardevents.GetLeaderboardRequestedPayloadV1{
			GuildID: batchPayload.GuildID,
		},
	}}, nil
}
