package roundrsvp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
)

// --- Constants (Consider moving to a shared package) ---
const (
	placeholderNoParticipants = "*No participants*"
	tagPrefix                 = "Tag:"
)

// UpdateRoundEventEmbed updates the round event embed with new participant information.
// PRESERVED: old signature was (ctx, channelID, messageID, acceptedParticipants, declinedParticipants, tentativeParticipants []Participant) — may be reused in PWA
func (rrm *roundRsvpManager) UpdateRoundEventEmbed(ctx context.Context, channelID string, messageID string, participants []roundtypes.Participant) (RoundRsvpOperationResult, error) {
	// Multi-tenant support: resolve channelID from guild config if not provided
	resolvedChannelID := channelID
	if resolvedChannelID == "" {
		guildID, _ := ctx.Value("guild_id").(string)
		if guildID != "" {
			cfg, err := rrm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
			if err == nil && cfg != nil && cfg.EventChannelID != "" {
				resolvedChannelID = cfg.EventChannelID
			}
		}
	}

	return rrm.operationWrapper(ctx, "UpdateRoundEventEmbed", func(ctx context.Context) (RoundRsvpOperationResult, error) {
		msg, err := rrm.session.ChannelMessage(resolvedChannelID, messageID)
		if err != nil {
			// Check if message was deleted (404) - don't retry, just log and succeed
			var restErr *discordgo.RESTError
			if errors.As(err, &restErr) && restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
				rrm.logger.WarnContext(ctx, "Message was deleted, skipping embed update",
					attr.String("channel_id", resolvedChannelID),
					attr.String("discord_message_id", messageID),
					attr.Int("discord_error_code", restErr.Message.Code))
				return RoundRsvpOperationResult{}, nil // Success - nothing to update
			}
			wrappedErr := fmt.Errorf("failed to fetch message: %w", err)
			rrm.logger.ErrorContext(ctx, "Failed to fetch message for embed update",
				attr.Error(wrappedErr),
				attr.String("channel_id", resolvedChannelID),
				attr.String("discord_message_id", messageID))
			return RoundRsvpOperationResult{Error: wrappedErr}, wrappedErr
		}

		if len(msg.Embeds) == 0 {
			err := fmt.Errorf("no embeds found in message")
			rrm.logger.ErrorContext(ctx, "No embeds found in message",
				attr.String("channel_id", resolvedChannelID),
				attr.String("discord_message_id", messageID))
			return RoundRsvpOperationResult{Error: err}, err
		}

		embed := msg.Embeds[0]

		if len(embed.Fields) < 3 { // Fields: 0=Time, 1=Location, 2=Participants
			err := fmt.Errorf("embed does not have expected fields (expected at least 3, got %d)", len(embed.Fields))
			rrm.logger.ErrorContext(ctx, "Embed doesn't have expected fields",
				attr.String("channel_id", resolvedChannelID),
				attr.String("discord_message_id", messageID),
				attr.Int("field_count", len(embed.Fields)))
			return RoundRsvpOperationResult{Error: err}, err
		}

		// Update the single Participants field at index 2
		embed.Fields[2].Value = rrm.formatParticipants(ctx, participants)

		// PRESERVED: old 3-field update — may be reused in PWA
		// embed.Fields[2].Value = rrm.formatParticipants(ctx, acceptedParticipants)
		// embed.Fields[3].Value = rrm.formatParticipants(ctx, declinedParticipants)
		// embed.Fields[4].Value = rrm.formatParticipants(ctx, tentativeParticipants)

		// Update the message in the channel
		updatedMsg, err := rrm.session.ChannelMessageEditEmbed(resolvedChannelID, messageID, embed)
		if err != nil {
			wrappedErr := fmt.Errorf("failed to update embed: %w", err)
			rrm.logger.ErrorContext(ctx, "Failed to update round event embed",
				attr.Error(wrappedErr),
				attr.String("channel_id", resolvedChannelID),
				attr.String("discord_message_id", messageID))
			return RoundRsvpOperationResult{Error: wrappedErr}, wrappedErr
		}

		rrm.logger.InfoContext(ctx, "Successfully updated round event embed",
			attr.String("channel_id", resolvedChannelID),
			attr.String("discord_message_id", messageID),
			attr.Int("participant_count", len(participants)))

		return RoundRsvpOperationResult{Success: updatedMsg}, nil
	})
}

// formatParticipants formats the participant list for the embed field value in the RSVP embed.
// It formats as "<@USER_ID> Tag: N" or just "<@USER_ID>" if no tag number.
// Includes logging for failed user info fetches.
func (rrm *roundRsvpManager) formatParticipants(ctx context.Context, participants []roundtypes.Participant) string {
	if len(participants) == 0 {
		return placeholderNoParticipants // Use consistent placeholder
	}

	var withTag []roundtypes.Participant
	var withoutTag []roundtypes.Participant

	for _, participant := range participants {
		if participant.TagNumber != nil && *participant.TagNumber > 0 {
			withTag = append(withTag, participant)
		} else {
			withoutTag = append(withoutTag, participant)
		}
	}

	sort.Slice(withTag, func(i, j int) bool {
		if withTag[i].TagNumber == nil || withTag[j].TagNumber == nil {
			return false
		}
		return *withTag[i].TagNumber < *withTag[j].TagNumber
	})

	sortedParticipants := append(withTag, withoutTag...)

	var lines []string
	for _, participant := range sortedParticipants {
		// Attempt to fetch user/member info primarily for logging purposes if it fails.
		// The fetched display name is NOT used in the final output line format.
		_, err := rrm.session.User(string(participant.UserID))
		if err != nil {
			// Log the error using the context. Format the potential display name attempt directly in the log attribute.
			rrm.logger.ErrorContext(ctx, "Failed to get user info for participant in formatParticipants",
				attr.Error(err),
				attr.String("user_id", string(participant.UserID)),
				// Format display name attempt for logging context if fetch fails
				attr.String("display_name_attempted", fmt.Sprintf("User ID: %s (Fetch Failed)", participant.UserID)),
				attr.String("status", string(participant.Response)),
				attr.Any("tag_number", participant.TagNumber), // Use attr.Any for pointers that might be nil
			)
			// Continue without fetching member if user fetch failed
		} else {
			// Fetch guild ID from context for member lookup
			guildID, _ := ctx.Value("guild_id").(string)
			if guildID != "" {
				_, memberErr := rrm.session.GuildMember(guildID, string(participant.UserID))
				if memberErr != nil {
					rrm.logger.WarnContext(ctx, "Failed to get guild member info for participant in formatParticipants",
						attr.Error(memberErr),
						attr.String("user_id", string(participant.UserID)),
						attr.String("guild_id", guildID),
						attr.String("status", string(participant.Response)),
						attr.Any("tag_number", participant.TagNumber),
					)
				}
			}
		}

		line := ""
		if participant.TagNumber != nil && *participant.TagNumber > 0 {
			// RSVP Format: <@USER_ID> Tag: N
			line = fmt.Sprintf("<@%s> %s %d", participant.UserID, tagPrefix, *participant.TagNumber) // Excluded icon and display name text
		} else {
			// RSVP Format: <@USER_ID> (for participants without a tag number)
			line = fmt.Sprintf("<@%s>", participant.UserID) // Excluded icon and display name text
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
