package roundrsvp

import (
	"context"
	"fmt"
	"strings"

	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/google/uuid"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	wmmessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

// HandleRoundResponse handles the interaction with the Discord API.
func (rrm *roundRsvpManager) HandleRoundResponse(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_round_response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	rrm.logger.InfoContext(ctx, "Handling round RSVP",
		attr.UserID(sharedtypes.DiscordID(userID)))

	// Add nil check for Message before accessing ID
	var messageID string
	if i.Message != nil {
		messageID = i.Message.ID
	}

	rrm.logger.InfoContext(ctx, "Processing RSVP interaction",
		attr.String("interaction_id", i.ID),
		attr.String("discord_message_id", messageID),
	)

	return rrm.operationWrapper(ctx, "handle_round_response", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		// Get user from either Member or direct User field
		var user *discordgo.User
		if i.Member != nil && i.Member.User != nil {
			user = i.Member.User
		} else if i.User != nil {
			user = i.User
		} else {
			return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
		}

		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			return RoundRsvpOperationResult{Error: fmt.Errorf("invalid custom ID: %s", customID)}, nil
		}
		eventIDStr := parts[1]

		var response roundtypes.Response
		switch {
		case strings.HasPrefix(customID, "round_accept"):
			response = roundtypes.ResponseAccept
		case strings.HasPrefix(customID, "round_decline"):
			response = roundtypes.ResponseDecline
		case strings.HasPrefix(customID, "round_tentative"):
			response = roundtypes.ResponseTentative
		default:
			return RoundRsvpOperationResult{Error: fmt.Errorf("unknown response type: %s", customID)}, nil
		}

		rrm.logger.InfoContext(ctx, "RSVP DEBUG",
			attr.String("custom_id", customID),
			attr.String("response", string(response)),
			attr.String("user_id", user.ID),
		)
		if err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		}); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		roundUUID, err := uuid.Parse(eventIDStr)
		if err != nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("failed to parse round UUID: %w", err)}, nil
		}

		// var tagNumberPtr *sharedtypes.TagNumber = nil

		// Multi-tenant: resolve channel ID from guild config if possible
		resolvedChannelID := i.ChannelID
		if rrm.guildConfigResolver != nil && i.GuildID != "" {
			cfg, err := rrm.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
			if err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
			}
		}

		payload := discordroundevents.DiscordRoundParticipantJoinRequestPayload{
			RoundID:    sharedtypes.RoundID(roundUUID),
			UserID:     sharedtypes.DiscordID(user.ID),
			ChannelID:  resolvedChannelID,
			JoinedLate: nil, // Normal join, not late
			GuildID:    i.GuildID,
		}

		msg := &wmmessage.Message{
			Metadata: wmmessage.Metadata{
				"discord_message_id": messageID, // Use the safely extracted messageID
				"topic":              discordroundevents.RoundParticipantJoinReqTopic,
				"response":           string(response),
			},
		}

		resultMsg, err := rrm.helper.CreateResultMessage(msg, payload, discordroundevents.RoundParticipantJoinReqTopic)
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		if err := rrm.publisher.Publish(discordroundevents.RoundParticipantJoinReqTopic, resultMsg); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		_, err = rrm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: fmt.Sprintf("You have chosen: %s", string(response)),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		return RoundRsvpOperationResult{Success: "RSVP successfully processed"}, nil
	})
}

// InteractionJoinRoundLate handles the interaction when a user clicks "Join Round LATE"
func (rrm *roundRsvpManager) InteractionJoinRoundLate(ctx context.Context, i *discordgo.InteractionCreate) (RoundRsvpOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "join_round_late")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "button")

	// Add nil checks before accessing user ID
	var userID string
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else if i.User != nil {
		userID = i.User.ID
		ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	} else {
		return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
	}

	rrm.logger.InfoContext(ctx, "Handling late round join", attr.UserID(sharedtypes.DiscordID(userID)))

	return rrm.operationWrapper(ctx, "join_round_late", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		// Get user from either Member or direct User field
		var user *discordgo.User
		if i.Member != nil && i.Member.User != nil {
			user = i.Member.User
		} else if i.User != nil {
			user = i.User
		} else {
			return RoundRsvpOperationResult{Error: fmt.Errorf("unable to determine user from interaction")}, nil
		}

		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		if len(parts) < 2 {
			return RoundRsvpOperationResult{Error: fmt.Errorf("invalid custom ID format: %s", customID)}, nil
		}

		roundUUIDStr := strings.TrimPrefix(parts[1], "round-")
		roundUUID, err := uuid.Parse(roundUUIDStr)
		if err != nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("failed to parse round UUID: %w", err)}, nil
		}

		// Check if user is already participating by fetching the current embed
		// Multi-tenant: resolve channel ID from guild config if possible
		resolvedChannelID := i.ChannelID
		if rrm.guildConfigResolver != nil && i.GuildID != "" {
			cfg, err := rrm.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
			if err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
			}
		}

		if rrm.session == nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}
		if i.Message == nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("interaction message is nil")}, nil
		}
		message, err := rrm.session.ChannelMessage(resolvedChannelID, i.Message.ID)
		if err != nil {
			return RoundRsvpOperationResult{Error: fmt.Errorf("failed to fetch current message: %w", err)}, nil
		}

		if len(message.Embeds) > 0 {
			embed := message.Embeds[0]

			// Check if user is already in any of the embed fields
			userAlreadyJoined := false
			for _, field := range embed.Fields {
				if strings.Contains(field.Value, fmt.Sprintf("<@%s>", user.ID)) {
					userAlreadyJoined = true
					break
				}
			}

			if userAlreadyJoined {
				// User is already in the round, send ephemeral message
				err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You have already joined this round!",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				if err != nil {
					return RoundRsvpOperationResult{Error: err}, nil
				}

				rrm.logger.InfoContext(ctx, "User attempted to join round they're already in",
					attr.UserID(sharedtypes.DiscordID(user.ID)),
					attr.String("round_id", roundUUID.String()))

				return RoundRsvpOperationResult{Success: "User already joined"}, nil
			}
		}

		// User is not already in the round, proceed with join logic
		if err := rrm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		}); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		tagNumber := sharedtypes.TagNumber(0)
		joinedLate := true

		payload := roundevents.ParticipantJoinRequestPayload{
			RoundID:    sharedtypes.RoundID(roundUUID),
			UserID:     sharedtypes.DiscordID(user.ID),
			Response:   roundtypes.ResponseAccept,
			TagNumber:  &tagNumber,
			JoinedLate: &joinedLate,
			GuildID:    sharedtypes.GuildID(i.GuildID),
		}

		// Add nil check for Message before accessing ID
		var messageID string
		if i.Message != nil {
			messageID = i.Message.ID
		}

		msg := &wmmessage.Message{
			Metadata: wmmessage.Metadata{
				"discord_message_id": messageID, // Use the safely extracted messageID
				"topic":              discordroundevents.RoundParticipantJoinReqTopic,
			},
		}

		resultMsg, err := rrm.helper.CreateResultMessage(msg, payload, discordroundevents.RoundParticipantJoinReqTopic)
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		if err := rrm.publisher.Publish(discordroundevents.RoundParticipantJoinReqTopic, resultMsg); err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		_, err = rrm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "You have joined the round!",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		if err != nil {
			return RoundRsvpOperationResult{Error: err}, nil
		}

		rrm.logger.InfoContext(ctx, "Successfully processed late join request",
			attr.UserID(sharedtypes.DiscordID(user.ID)),
			attr.String("round_id", roundUUID.String()))

		return RoundRsvpOperationResult{Success: "Late join successfully processed"}, nil
	})
}
