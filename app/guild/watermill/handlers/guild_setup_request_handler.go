package handlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGuildSetupRequest processes initial guild setup requests from Discord
func (h *GuildHandlers) HandleGuildSetupRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleGuildSetupRequest",
		&guildevents.GuildSetupEvent{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			setupEvent := payload.(*guildevents.GuildSetupEvent)

			h.Logger.InfoContext(ctx, "Processing guild setup request",
				attr.String("guild_id", setupEvent.GuildID),
				attr.String("guild_name", setupEvent.GuildName),
				attr.String("admin_user_id", setupEvent.AdminUserID))

			// Transform Discord event to backend request format
			// Backward compatibility: map RegisteredRoleID to UserRoleID if the latter is empty
			userRoleID := setupEvent.UserRoleID
			if userRoleID == "" && setupEvent.RegisteredRoleID != "" {
				userRoleID = setupEvent.RegisteredRoleID
			}
			backendPayload := sharedevents.GuildConfigRequestedPayload{
				GuildID:              sharedtypes.GuildID(setupEvent.GuildID),
				SignupChannelID:      setupEvent.SignupChannelID,
				SignupMessageID:      setupEvent.SignupMessageID,
				EventChannelID:       setupEvent.EventChannelID,
				LeaderboardChannelID: setupEvent.LeaderboardChannelID,
				UserRoleID:           userRoleID,
				EditorRoleID:         setupEvent.EditorRoleID,
				AdminRoleID:          setupEvent.AdminRoleID,
				SignupEmoji:          setupEvent.SignupEmoji,
				AutoSetupCompleted:   true,
				SetupCompletedAt:     &setupEvent.SetupCompletedAt,
			}

			// Create message to backend with the creation request topic
			backendMsg, err := h.Helpers.CreateResultMessage(msg, backendPayload, sharedevents.GuildConfigCreationRequested)
			if err != nil {
				h.Logger.ErrorContext(ctx, "Failed to create backend message",
					attr.String("guild_id", setupEvent.GuildID),
					attr.Error(err))
				return nil, fmt.Errorf("failed to create backend message: %w", err)
			}

			// Ensure guild_id is propagated for multi-tenant isolation
			backendMsg.Metadata.Set("guild_id", setupEvent.GuildID)

			h.Logger.InfoContext(ctx, "Forwarding guild setup request to backend",
				attr.String("guild_id", setupEvent.GuildID),
				attr.String("target_topic", sharedevents.GuildConfigCreationRequested))

			return []*message.Message{backendMsg}, nil
		},
	)(msg)
}
