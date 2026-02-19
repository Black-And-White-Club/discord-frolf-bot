package signup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// SignupManager defines the interface for signup operations.
type SignupManager interface {
	SendSignupModal(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error)
	HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error)
	MessageReactionAdd(s discord.Session, r *discordgo.MessageReactionAdd) (SignupOperationResult, error)
	HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) (SignupOperationResult, error)
	HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (SignupOperationResult, error)
	SendSignupResult(ctx context.Context, interactionToken string, success bool, failureReason ...string) (SignupOperationResult, error)
	// TrackChannelForReactions registers a channel to have its reactions processed
	TrackChannelForReactions(channelID string)
	// SyncMember fetches a guild member from Discord and publishes a profile update event.
	SyncMember(ctx context.Context, guildID, userID string) error
}

type signupManager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	config              *config.Config // Deprecated: use guildConfigResolver for per-guild config
	guildConfigResolver guildconfig.GuildConfigResolver
	interactionStore    storage.ISInterface[any]
	guildConfigCache    storage.ISInterface[storage.GuildConfig]
	tracer              trace.Tracer
	metrics             discordmetrics.DiscordMetrics
	operationWrapper    func(ctx context.Context, opName string, fn func(ctx context.Context) (SignupOperationResult, error)) (SignupOperationResult, error)
	trackedChannels     sync.Map // map[channelID]bool - channels we listen for reactions on (no backend call on miss)
}

// NewSignupManager creates a new SignupManager instance.
func NewSignupManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	config *config.Config, // Deprecated: use guildConfigResolver for per-guild config
	guildConfigResolver guildconfig.GuildConfigResolver,
	interactionStore storage.ISInterface[any],
	guildConfigCache storage.ISInterface[storage.GuildConfig],
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (SignupManager, error) {
	if logger != nil {
		logger.InfoContext(context.Background(), "Creating SignupManager")
	}

	return &signupManager{
		session:             session,
		publisher:           publisher,
		logger:              logger,
		helper:              helper,
		config:              config, // Deprecated
		guildConfigResolver: guildConfigResolver,
		interactionStore:    interactionStore,
		guildConfigCache:    guildConfigCache,
		tracer:              tracer,
		metrics:             metrics,
		operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) (SignupOperationResult, error)) (SignupOperationResult, error) {
			return wrapSignupOperation(ctx, opName, fn, logger, tracer, metrics)
		},
	}, nil
}

// createEvent creates a Watermill message to send to the backend.
func (sm *signupManager) createEvent(ctx context.Context, topic string, payload interface{}, i *discordgo.InteractionCreate) (*message.Message, error) {
	newEvent := message.NewMessage(watermill.NewUUID(), nil)

	if newEvent.Metadata == nil {
		newEvent.Metadata = make(map[string]string)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to marshal payload in createEvent", attr.Error(err))
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	newEvent.Payload = payloadBytes
	newEvent.Metadata.Set("handler_name", "Signup Event")
	newEvent.Metadata.Set("topic", topic)
	newEvent.Metadata.Set("domain", "discord")
	newEvent.Metadata.Set("interaction_id", i.Interaction.ID)
	newEvent.Metadata.Set("interaction_token", i.Interaction.Token)

	// Use GuildID from the actual interaction
	if i.Interaction.GuildID != "" {
		newEvent.Metadata.Set("guild_id", i.Interaction.GuildID)
	} else {
		sm.logger.WarnContext(ctx, "Interaction missing GuildID in createEvent")
	}

	return newEvent, nil
}

// wrapSignupOperation is the shared tracing/logging/metrics wrapper
func wrapSignupOperation(
	ctx context.Context,
	operationName string,
	fn func(ctx context.Context) (SignupOperationResult, error),
	logger *slog.Logger,
	tracer trace.Tracer,
	metrics discordmetrics.DiscordMetrics,
) (result SignupOperationResult, err error) {
	if fn == nil {
		return SignupOperationResult{}, errors.New("operation function is nil")
	}

	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("noop")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	start := time.Now()

	defer func() {
		duration := time.Since(start)
		if logger != nil {
			logger.InfoContext(ctx, fmt.Sprintf("Completed %s", operationName),
				attr.String("duration_sec", fmt.Sprintf("%.2f", duration.Seconds())),
			)
		}
		if metrics != nil {
			metrics.RecordAPIRequestDuration(ctx, operationName, duration)
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			if logger != nil {
				logger.ErrorContext(ctx, "Recovered from panic", attr.Error(err))
			}
			span.RecordError(err)
			if metrics != nil {
				metrics.RecordAPIError(ctx, operationName, "panic")
			}
		}
	}()

	result, err = fn(ctx)
	if err != nil {
		wrapped := fmt.Errorf("%s operation error: %w", operationName, err)
		if logger != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error in %s", operationName), attr.Error(wrapped))
		}
		span.RecordError(wrapped)
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "operation_error")
		}
		return SignupOperationResult{}, wrapped
	}

	if result.Error != nil {
		span.RecordError(result.Error)
		if metrics != nil {
			metrics.RecordAPIError(ctx, operationName, "result_error")
		}
	} else if metrics != nil {
		metrics.RecordAPIRequest(ctx, operationName)
	}

	return result, nil
}

// TrackChannelForReactions registers a channel to have its reactions processed.
// This should be called when a guild is set up or when the bot creates a new managed channel.
func (sm *signupManager) TrackChannelForReactions(channelID string) {
	if channelID == "" {
		return
	}
	sm.trackedChannels.Store(channelID, true)
}

// SyncMember fetches a guild member from Discord and publishes a profile update event.
// This enables self-healing data when club memberships have empty display names.
func (sm *signupManager) SyncMember(ctx context.Context, guildID, userID string) error {
	ctx, span := sm.tracer.Start(ctx, "SignupManager.SyncMember", trace.WithAttributes(
		attribute.String("guild_id", guildID),
		attribute.String("user_id", userID),
	))
	defer span.End()

	member, err := sm.session.GuildMember(guildID, userID)
	if err != nil {
		sm.logger.WarnContext(ctx, "Failed to fetch guild member for profile sync",
			attr.Error(err),
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
		)
		span.RecordError(err)
		return fmt.Errorf("failed to fetch guild member: %w", err)
	}

	if member == nil || member.User == nil {
		sm.logger.WarnContext(ctx, "Guild member or user is nil",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
		)
		return fmt.Errorf("member not found in guild")
	}

	sm.publishUserProfile(ctx, member, guildID)
	sm.publishRoleSync(ctx, guildID, member.User.ID, member.Roles)

	sm.logger.InfoContext(ctx, "Profile sync completed",
		attr.String("guild_id", guildID),
		attr.String("user_id", userID),
		attr.String("display_name", guildNickname(member)),
	)

	return nil
}

// publishUserProfile publishes a profile update event from a guild member.
func (sm *signupManager) publishUserProfile(ctx context.Context, member *discordgo.Member, guildID string) {
	if member == nil || member.User == nil {
		return
	}

	user := member.User

	payload := &userevents.UserProfileUpdatedPayloadV1{
		UserID:      sharedtypes.DiscordID(user.ID),
		GuildID:     sharedtypes.GuildID(guildID),
		Username:    user.Username,
		DisplayName: guildNickname(member),
		AvatarHash:  user.Avatar,
	}

	msg, err := sm.helper.CreateNewMessage(payload, userevents.UserProfileUpdatedV1)
	if err != nil || msg == nil {
		sm.logger.WarnContext(ctx, "Failed to create user profile message", attr.Error(err))
		return
	}

	msg.Metadata.Set("user_id", user.ID)
	msg.Metadata.Set("guild_id", guildID)

	if err := sm.publisher.Publish(userevents.UserProfileUpdatedV1, msg); err != nil {
		sm.logger.WarnContext(ctx, "Failed to publish user profile event",
			attr.Error(err),
			attr.String("user_id", user.ID),
			attr.String("guild_id", guildID),
		)
	}
}

// guildNickname returns the member's server-specific nickname, or empty string if none is set.
func guildNickname(member *discordgo.Member) string {
	return member.Nick
}

// publishRoleSync derives a user's role from their Discord guild roles and publishes
// a UserRoleUpdateRequestedV1 event so frolf-bot can update the DB. Best-effort.
func (sm *signupManager) publishRoleSync(ctx context.Context, guildID, userID string, memberRoles []string) {
	guildCfg, err := sm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
	if err != nil || guildCfg == nil {
		sm.logger.WarnContext(ctx, "Failed to get guild config for role sync",
			attr.Error(err),
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
		)
		return
	}

	derived := deriveRole(memberRoles, guildCfg)
	if derived == "" {
		sm.logger.InfoContext(ctx, "No recognized role found for member, skipping role sync",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
		)
		return
	}

	payload := &userevents.UserRoleUpdateRequestedPayloadV1{
		GuildID: sharedtypes.GuildID(guildID),
		UserID:  sharedtypes.DiscordID(userID),
		Role:    derived,
		// RequesterID intentionally empty for system-initiated syncs
	}

	msg, err := sm.helper.CreateNewMessage(payload, userevents.UserRoleUpdateRequestedV1)
	if err != nil || msg == nil {
		sm.logger.WarnContext(ctx, "Failed to create role sync message", attr.Error(err))
		return
	}

	msg.Metadata.Set("user_id", userID)
	msg.Metadata.Set("guild_id", guildID)

	if err := sm.publisher.Publish(userevents.UserRoleUpdateRequestedV1, msg); err != nil {
		sm.logger.WarnContext(ctx, "Failed to publish role sync event",
			attr.Error(err),
			attr.String("user_id", userID),
			attr.String("guild_id", guildID),
		)
	} else {
		sm.logger.InfoContext(ctx, "Published role sync event",
			attr.String("user_id", userID),
			attr.String("guild_id", guildID),
			attr.String("role", string(derived)),
		)
	}
}

// deriveRole maps a member's Discord role IDs to a UserRoleEnum using guild config.
// Priority: Admin > Editor > Registered. Returns "" if none match (skip update).
func deriveRole(memberRoles []string, cfg *storage.GuildConfig) sharedtypes.UserRoleEnum {
	for _, r := range memberRoles {
		if cfg.AdminRoleID != "" && r == cfg.AdminRoleID {
			return sharedtypes.UserRoleAdmin
		}
	}
	for _, r := range memberRoles {
		if cfg.EditorRoleID != "" && r == cfg.EditorRoleID {
			return sharedtypes.UserRoleEditor
		}
	}
	for _, r := range memberRoles {
		if cfg.RegisteredRoleID != "" && r == cfg.RegisteredRoleID {
			return sharedtypes.UserRoleUser
		}
	}
	return ""
}

type SignupOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
