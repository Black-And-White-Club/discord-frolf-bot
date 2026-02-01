package utils

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// PublishUserProfile extracts profile data from a Discord interaction and publishes an event.
// This should be called asynchronously (go PublishUserProfile(...)) to avoid blocking.
func PublishUserProfile(
	ctx context.Context,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	user *discordgo.User,
	member *discordgo.Member,
	guildID string,
) {
	if user == nil {
		return
	}

	// Determine display name: prefer guild nickname, fall back to username
	displayName := user.Username
	if member != nil && member.Nick != "" {
		displayName = member.Nick
	}

	// Avatar hash (empty string if using default)
	avatarHash := ""
	if user.Avatar != "" {
		avatarHash = user.Avatar
	}

	payload := &userevents.UserProfileUpdatedPayloadV1{
		UserID:      sharedtypes.DiscordID(user.ID),
		GuildID:     sharedtypes.GuildID(guildID),
		Username:    user.Username,
		DisplayName: displayName,
		AvatarHash:  avatarHash,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("Failed to marshal user profile payload", "error", err)
		return
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("user_id", user.ID)
	msg.Metadata.Set("guild_id", guildID)
	msg.Metadata.Set("topic", userevents.UserProfileUpdatedV1)

	err = eventBus.Publish(userevents.UserProfileUpdatedV1, msg)
	if err != nil {
		logger.Warn("Failed to publish user profile event",
			"error", err,
			"user_id", user.ID,
			"guild_id", guildID,
		)
	} else {
		logger.Debug("Published user profile event",
			"user_id", user.ID,
			"display_name", displayName,
		)
	}
}
