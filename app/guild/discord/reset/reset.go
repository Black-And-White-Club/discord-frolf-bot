package reset

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	discordgocommands "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace"
)

// ResetManager handles guild configuration reset operations.
type ResetManager interface {
	HandleResetCommand(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetConfirmButton(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetCancelButton(ctx context.Context, i *discordgo.InteractionCreate) error
	DeleteResources(ctx context.Context, guildID string, state guildtypes.ResourceState) (map[string]guildtypes.DeletionResult, error)
}

type resetManager struct {
	session          discordgocommands.Session
	publisher        eventbus.EventBus
	logger           *slog.Logger
	helper           utils.Helpers
	config           *config.Config
	interactionStore storage.ISInterface
	tracer           trace.Tracer
	metrics          discordmetrics.DiscordMetrics
	operationWrapper func(ctx context.Context, operationName string, fn func(context.Context) error) error
}

// NewResetManager creates a new reset manager.
func NewResetManager(
	session discordgocommands.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config,
	interactionStore storage.ISInterface,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (ResetManager, error) {
	if session == nil {
		return nil, fmt.Errorf("session cannot be nil")
	}
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if helper == nil {
		return nil, fmt.Errorf("helper cannot be nil")
	}

	if logger != nil {
		logger.InfoContext(context.Background(), "Creating Guild ResetManager")
	}

	return &resetManager{
		session:          session,
		publisher:        publisher,
		logger:           logger,
		helper:           helper,
		config:           config,
		interactionStore: interactionStore,
		tracer:           tracer,
		metrics:          metrics,
		operationWrapper: func(ctx context.Context, operationName string, fn func(context.Context) error) error {
			return wrapResetOperation(ctx, operationName, fn, logger, tracer, metrics)
		},
	}, nil
}

// wrapResetOperation wraps reset operations with observability and error handling
func wrapResetOperation(
	ctx context.Context,
	operationName string,
	fn func(context.Context) error,
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) error {
	// Start tracing span
	ctx, span := tracer.Start(ctx, fmt.Sprintf("guild.reset.%s", operationName))
	defer span.End()

	start := time.Now()

	err := fn(ctx)

	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		if logger != nil {
			logger.ErrorContext(ctx, "Guild reset operation failed",
				"operation", operationName,
				"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()),
				"error", err)
		}
		return err
	}

	if logger != nil {
		logger.InfoContext(ctx, "Guild reset operation completed",
			"operation", operationName,
			"duration_sec", fmt.Sprintf("%.2f", duration.Seconds()))
	}

	return nil
}

// Local result keys (kept in sync with handlers' keys where possible)
const (
	resultSignupMessage      = "signup_message"
	resultSignupChannel      = "signup_channel"
	resultEventChannel       = "event_channel"
	resultLeaderboardChannel = "leaderboard_channel"
	resultUserRole           = "user_role"
	resultEditorRole         = "editor_role"
	resultAdminRole          = "admin_role"
)

// DeleteResources performs best-effort, idempotent deletions of Discord
// resources captured in the provided ResourceState. It returns a map of per-
// resource DeletionResult and does not perform event publishing.
func (rm *resetManager) DeleteResources(ctx context.Context, guildID string, state guildtypes.ResourceState) (map[string]guildtypes.DeletionResult, error) {
	results := make(map[string]guildtypes.DeletionResult)
	now := time.Now()
	recordSuccess := func(key string) {
		results[key] = guildtypes.DeletionResult{Status: "success", DeletedAt: &now}
	}
	recordFailure := func(key string, err error) {
		results[key] = guildtypes.DeletionResult{Status: "failed", Error: err.Error()}
	}

	if rm.session == nil {
		err := fmt.Errorf("session is nil")
		rm.logger.ErrorContext(ctx, "DeleteResources failed", attr.Error(err))
		return results, err
	}

	if state.IsEmpty() {
		return results, nil
	}

	// Signup message
	if state.SignupMessageID != "" && state.SignupChannelID != "" {
		if err := rm.session.ChannelMessageDelete(state.SignupChannelID, state.SignupMessageID); err != nil {
			// Treat Unknown Message as already deleted (success). Other REST errors
			// such as Missing Permissions should be logged but not retried.
			var restErr *discordgo.RESTError
			if errors.As(err, &restErr) {
				switch restErr.Message.Code {
				case discordgo.ErrCodeUnknownMessage, discordgo.ErrCodeUnknownChannel:
					rm.logger.InfoContext(ctx, "Signup message or channel already deleted",
						attr.String("guild_id", guildID),
						attr.String("channel_id", state.SignupChannelID),
						attr.String("discord_message_id", state.SignupMessageID),
						"code", restErr.Message.Code)
					recordSuccess(resultSignupMessage)
				case discordgo.ErrCodeMissingPermissions:
					rm.logger.WarnContext(ctx, "Missing permissions to delete signup message",
						attr.String("guild_id", guildID),
						attr.String("channel_id", state.SignupChannelID),
						attr.String("discord_message_id", state.SignupMessageID))
					recordFailure(resultSignupMessage, fmt.Errorf("missing permissions"))
				default:
					// Unknown error type — return to trigger retry
					return results, fmt.Errorf("failed to delete signup message %s/%s: %w", state.SignupChannelID, state.SignupMessageID, err)
				}
			} else {
				rm.logger.ErrorContext(ctx, "Failed to delete signup message",
					attr.String("guild_id", guildID),
					attr.String("channel_id", state.SignupChannelID),
					attr.String("discord_message_id", state.SignupMessageID),
					attr.Error(err))
				recordFailure(resultSignupMessage, err)
			}
		} else {
			recordSuccess(resultSignupMessage)
		}
	}

	// Channels
	channelDeletes := map[string]string{
		resultSignupChannel:      state.SignupChannelID,
		resultEventChannel:       state.EventChannelID,
		resultLeaderboardChannel: state.LeaderboardChannelID,
	}
	for key, channelID := range channelDeletes {
		if channelID == "" {
			continue
		}
		if err := rm.session.ChannelDelete(channelID); err != nil {
			var restErr *discordgo.RESTError
			if errors.As(err, &restErr) {
				// 10003 = Unknown Channel, 10004 = Unknown Guild
				if restErr.Message.Code == discordgo.ErrCodeUnknownChannel || restErr.Message.Code == discordgo.ErrCodeUnknownGuild {
					rm.logger.InfoContext(ctx, "Channel already deleted or guild missing",
						attr.String("guild_id", guildID),
						attr.String("channel_id", channelID),
						"code", restErr.Message.Code)
					recordSuccess(key)
					continue
				}

				// 50013 = Missing Permissions — log and continue without retry
				if restErr.Message.Code == discordgo.ErrCodeMissingPermissions {
					rm.logger.WarnContext(ctx, "Missing permissions to delete channel",
						attr.String("guild_id", guildID),
						attr.String("channel_id", channelID))
					recordFailure(key, fmt.Errorf("missing permissions"))
					continue
				}
			}

			// Any other error should be returned to trigger a retry
			rm.logger.ErrorContext(ctx, "Failed to delete channel",
				attr.String("guild_id", guildID),
				attr.String("channel_id", channelID),
				attr.Error(err))
			return results, fmt.Errorf("failed to delete channel %s: %w", channelID, err)
		} else {
			recordSuccess(key)
		}
	}

	// Roles
	roleDeletes := map[string]string{
		resultUserRole:   state.UserRoleID,
		resultEditorRole: state.EditorRoleID,
		resultAdminRole:  state.AdminRoleID,
	}
	for key, roleID := range roleDeletes {
		if roleID == "" {
			continue
		}
		if err := rm.session.GuildRoleDelete(guildID, roleID); err != nil {
			var restErr *discordgo.RESTError
			if errors.As(err, &restErr) {
				// 10011 = Unknown Role, 10004 = Unknown Guild
				if restErr.Message.Code == discordgo.ErrCodeUnknownRole || restErr.Message.Code == discordgo.ErrCodeUnknownGuild {
					rm.logger.InfoContext(ctx, "Role already deleted or guild missing",
						attr.String("guild_id", guildID),
						attr.String("role_id", roleID),
						"code", restErr.Message.Code)
					recordSuccess(key)
					continue
				}

				// 50013 = Missing Permissions — log and continue without retry
				if restErr.Message.Code == discordgo.ErrCodeMissingPermissions {
					rm.logger.WarnContext(ctx, "Missing permissions to delete role",
						attr.String("guild_id", guildID),
						attr.String("role_id", roleID))
					recordFailure(key, fmt.Errorf("missing permissions"))
					continue
				}
			}

			// Any other error should be returned to trigger a retry
			rm.logger.ErrorContext(ctx, "Failed to delete role",
				attr.String("guild_id", guildID),
				attr.String("role_id", roleID),
				attr.Error(err))
			return results, fmt.Errorf("failed to delete role %s: %w", roleID, err)
		} else {
			recordSuccess(key)
		}
	}

	return results, nil
}
