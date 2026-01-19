package roundhandlers

import (
	"context"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for all round event handlers using pure transformation pattern
type Handlers interface {
	// Creation flow
	HandleRoundCreateRequested(ctx context.Context, payload *discordroundevents.CreateRoundModalPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundCreated(ctx context.Context, payload *roundevents.RoundCreatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundCreationFailed(ctx context.Context, payload *roundevents.RoundCreationFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundValidationFailed(ctx context.Context, payload *roundevents.RoundValidationFailedPayloadV1) ([]handlerwrapper.Result, error)

	// Update flow
	HandleRoundUpdateRequested(ctx context.Context, payload *discordroundevents.RoundUpdateModalSubmittedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundUpdated(ctx context.Context, payload *roundevents.RoundEntityUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundUpdateFailed(ctx context.Context, payload *roundevents.RoundUpdateErrorPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundUpdateValidationFailed(ctx context.Context, payload *roundevents.RoundUpdateValidatedPayloadV1) ([]handlerwrapper.Result, error)

	// Participation
	HandleRoundParticipantJoinRequest(ctx context.Context, payload *discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundParticipantRemoved(ctx context.Context, payload *roundevents.ParticipantRemovedPayloadV1) ([]handlerwrapper.Result, error)

	// Scoring
	HandleDiscordRoundScoreUpdate(ctx context.Context, payload *discordroundevents.RoundScoreUpdateRequestDiscordPayloadV1) ([]handlerwrapper.Result, error)
	HandleParticipantScoreUpdated(ctx context.Context, payload *roundevents.ParticipantScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoresBulkUpdated(ctx context.Context, payload *roundevents.RoundScoresBulkUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScoreUpdateError(ctx context.Context, payload *roundevents.RoundScoreUpdateErrorPayloadV1) ([]handlerwrapper.Result, error)

	// Score override bridging (CorrectScore service)
	HandleScoreOverrideSuccess(ctx context.Context, payload *sharedevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error)

	// Scorecard import flow
	HandleScorecardUploaded(ctx context.Context, payload *roundevents.ScorecardUploadedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScorecardParseFailed(ctx context.Context, payload *roundevents.ScorecardParseFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleImportFailed(ctx context.Context, payload *roundevents.ImportFailedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScorecardURLRequested(ctx context.Context, payload *roundevents.ScorecardURLRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// Deletion flow
	HandleRoundDeleteRequested(ctx context.Context, payload *discordroundevents.RoundDeleteRequestDiscordPayloadV1) ([]handlerwrapper.Result, error)

	// Lifecycle
	HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedDiscordPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundStarted(ctx context.Context, payload *roundevents.DiscordRoundStartPayloadV1) ([]handlerwrapper.Result, error)

	// Tag handling
	HandleRoundParticipantJoined(ctx context.Context, payload *roundevents.ParticipantJoinedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundParticipantsUpdated(ctx context.Context, payload *roundevents.RoundParticipantsUpdatedPayloadV1) ([]handlerwrapper.Result, error)
	HandleScheduledRoundsSynced(ctx context.Context, payload *roundevents.ScheduledRoundsSyncedPayloadV1) ([]handlerwrapper.Result, error)

	// Reminders
	HandleRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) ([]handlerwrapper.Result, error)
}
