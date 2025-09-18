package tagupdates

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// Constants for embed parsing (copied from other managers for consistency)
const (
	placeholderNoParticipants = "*No participants*"
	tagPrefix                 = "Tag:"
)

// Regex for parsing participant lines - simplified for tag-only updates
var participantLineRegex = regexp.MustCompile(`<@!?([a-zA-Z0-9]+)>` + // Capture User ID
	`(?:\s+` + tagPrefix + `\s*(\d+))?`) // Optionally capture Tag Number

// UpdateTagsInEmbed updates tag numbers for specific participants in round embeds.
func (tum *tagUpdateManager) UpdateTagsInEmbed(ctx context.Context, channelID, messageID string, tagUpdates map[sharedtypes.DiscordID]*sharedtypes.TagNumber) (TagUpdateOperationResult, error) {
	return tum.operationWrapper(ctx, "UpdateTagsInEmbed", func(ctx context.Context) (TagUpdateOperationResult, error) {
		if tum.session == nil {
			return TagUpdateOperationResult{Error: fmt.Errorf("session is nil")}, nil
		}

		// Multi-tenant: resolve channel ID from guild config if not provided
		resolvedChannelID := channelID
		if resolvedChannelID == "" {
			guildID, _ := ctx.Value("guild_id").(string)
			if guildID != "" {
				cfg, err := tum.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
				if err == nil && cfg != nil && cfg.EventChannelID != "" {
					resolvedChannelID = cfg.EventChannelID
				}
			}
		}

		// 1. Fetch the existing message and embed
		message, err := tum.session.ChannelMessage(resolvedChannelID, messageID)
		if err != nil {
			if tum.logger != nil {
				tum.logger.ErrorContext(ctx, "Failed to fetch message for tag update",
					attr.Error(err),
					attr.String("channel_id", resolvedChannelID),
					attr.String("message_id", messageID),
				)
			}
			return TagUpdateOperationResult{Error: err}, fmt.Errorf("failed to fetch message for tag update: %w", err)
		}

		if len(message.Embeds) == 0 {
			if tum.logger != nil {
				tum.logger.WarnContext(ctx, "No embeds found in message for tag update",
					attr.String("channel_id", resolvedChannelID),
					attr.String("message_id", messageID),
				)
			}
			return TagUpdateOperationResult{Success: "No embeds found to update"}, nil
		}

		// Assuming the round embed is the first embed
		embed := message.Embeds[0]
		usersFoundAndUpdated := make(map[sharedtypes.DiscordID]bool)

		// 2. Iterate through relevant embed fields (Accepted, Tentative, etc.)
		participantFields := []*discordgo.MessageEmbedField{}

		for _, field := range embed.Fields {
			fieldNameLower := strings.ToLower(field.Name)
			// Check for participant status fields
			if strings.Contains(fieldNameLower, "accepted") ||
				strings.Contains(fieldNameLower, "tentative") ||
				strings.Contains(fieldNameLower, "declined") ||
				strings.Contains(fieldNameLower, "participants") ||
				strings.Contains(fieldNameLower, "✅") ||
				strings.Contains(fieldNameLower, "❓") ||
				strings.Contains(fieldNameLower, "❌") {
				participantFields = append(participantFields, field)
			}
		}

		if len(participantFields) == 0 {
			tum.logger.WarnContext(ctx, "No participant fields found in embed for tag update",
				attr.String("channel_id", resolvedChannelID),
				attr.String("message_id", messageID),
			)
			return TagUpdateOperationResult{Success: "No participant fields found"}, nil
		}

		// 3. Parse lines in participant fields ONCE and update as needed
		updatedFieldValues := map[string]string{}

		for _, field := range participantFields {
			if tum.logger != nil {
				tum.logger.DebugContext(ctx, "Processing field for tag update",
					attr.String("field_name", field.Name),
					attr.String("field_value", field.Value),
				)
			}

			originalLines := strings.Split(field.Value, "\n")
			newLines := []string{}

			if strings.TrimSpace(field.Value) == "" || field.Value == placeholderNoParticipants {
				// If the field is empty or placeholder, no participants to update
				updatedFieldValues[field.Name] = field.Value
				if tum.logger != nil {
					tum.logger.DebugContext(ctx, "Field is empty or placeholder, skipping",
						attr.String("field_name", field.Name),
					)
				}
				continue
			}

			// Process each line once and check if the user needs a tag update
			for lineIdx, line := range originalLines {
				if tum.logger != nil {
					tum.logger.DebugContext(ctx, "Processing line in field",
						attr.String("field_name", field.Name),
						attr.Int("line_index", lineIdx),
						attr.String("line", line),
					)
				}

				parsedUserID, parsedTagNumber, ok := tum.parseParticipantLine(ctx, line)

				if !ok {
					// If a line can't be parsed, keep the original line
					if tum.logger != nil {
						tum.logger.WarnContext(ctx, "Failed to parse participant line in UpdateTagsInEmbed",
							attr.String("line", line),
							attr.String("field_name", field.Name),
							attr.String("message_id", messageID),
						)
					}
					newLines = append(newLines, line)
					continue
				}

				if tum.logger != nil {
					tum.logger.DebugContext(ctx, "Successfully parsed line",
						attr.String("parsed_user_id", string(parsedUserID)),
						attr.Bool("has_existing_tag", parsedTagNumber != nil),
					)
				}

				// Check if this user needs a tag update (SINGLE CHECK)
				if newTagNumber, shouldUpdate := tagUpdates[parsedUserID]; shouldUpdate {
					// Mark this user as found and updated
					usersFoundAndUpdated[parsedUserID] = true

					if tum.logger != nil {
						tum.logger.DebugContext(ctx, "Found target user, updating tag",
							attr.String("user_id", string(parsedUserID)),
							attr.Int("old_tag", func() int {
								if parsedTagNumber != nil {
									return int(*parsedTagNumber)
								}
								return 0
							}()),
							attr.Int("new_tag", int(*newTagNumber)),
						)
					}

					// Use the NEW tag number
					updatedLine := tum.formatParticipantLine(parsedUserID, newTagNumber)
					newLines = append(newLines, updatedLine)

					if tum.logger != nil {
						tum.logger.DebugContext(ctx, "Updated participant tag in embed field",
							attr.String("user_id", string(parsedUserID)),
							attr.String("field_name", field.Name),
							attr.String("original_line", line),
							attr.String("updated_line", updatedLine),
							attr.Int("new_tag", int(*newTagNumber)),
						)
					}

				} else {
					// User doesn't need updating - keep their current info
					originalLine := tum.formatParticipantLine(parsedUserID, parsedTagNumber)
					newLines = append(newLines, originalLine)
				}
			}

			// Update the field value
			if len(newLines) == 0 {
				updatedFieldValues[field.Name] = placeholderNoParticipants
			} else {
				updatedFieldValues[field.Name] = strings.Join(newLines, "\n")
			}
		}

		// Check if any of the target users were found and updated
		if len(usersFoundAndUpdated) == 0 {
			if tum.logger != nil {
				tum.logger.WarnContext(ctx, "None of the target users were found in embed fields for tag update",
					attr.String("channel_id", resolvedChannelID),
					attr.String("message_id", messageID),
				)
			}
			return TagUpdateOperationResult{Success: "No target users found in embed fields"}, nil
		}

		// 4. Update the embed with the new field values
		for i, field := range embed.Fields {
			if newValue, ok := updatedFieldValues[field.Name]; ok {
				embed.Fields[i].Value = newValue
			}
		}

		// 5. Edit the Discord message with the modified embed
		edit := &discordgo.MessageEdit{
			Channel: resolvedChannelID,
			ID:      messageID,
		}
		edit.SetEmbeds([]*discordgo.MessageEmbed{embed})

		updatedMsg, err := tum.session.ChannelMessageEditComplex(edit)
		if err != nil {
			if tum.logger != nil {
				tum.logger.ErrorContext(ctx, "Failed to update message with new tags",
					attr.Error(err),
					attr.String("channel_id", resolvedChannelID),
					attr.String("message_id", messageID),
				)
			}
			return TagUpdateOperationResult{Error: err}, fmt.Errorf("failed to edit message for tag update: %w", err)
		}

		if tum.logger != nil {
			tum.logger.InfoContext(ctx, "Successfully updated user tags in embed",
				attr.String("channel_id", resolvedChannelID),
				attr.String("message_id", messageID),
				attr.Int("users_updated", len(usersFoundAndUpdated)),
			)
		}

		return TagUpdateOperationResult{Success: updatedMsg}, nil
	})
}

// parseParticipantLine parses a participant line to extract UserID and TagNumber (no score data for scheduled rounds)
func (tum *tagUpdateManager) parseParticipantLine(ctx context.Context, line string) (sharedtypes.DiscordID, *sharedtypes.TagNumber, bool) {
	match := participantLineRegex.FindStringSubmatch(line)
	if len(match) < 2 || match[1] == "" {
		// Must find a User ID mention
		return "", nil, false
	}
	userID := sharedtypes.DiscordID(match[1])

	var tagNumber *sharedtypes.TagNumber
	if len(match) > 2 && match[2] != "" {
		// Found and captured a tag number
		parsedTag, err := strconv.Atoi(match[2])
		if err == nil {
			typedTag := sharedtypes.TagNumber(parsedTag)
			tagNumber = &typedTag
		} else {
			tum.logger.WarnContext(ctx, "Could not parse tag number from line in UpdateTagsInEmbed",
				attr.String("tag_str", match[2]),
				attr.String("line", line),
				attr.Error(err),
			)
		}
	}

	return userID, tagNumber, true
}

// formatParticipantLine formats a participant line for scheduled rounds (no score data)
func (tum *tagUpdateManager) formatParticipantLine(userID sharedtypes.DiscordID, tagNumber *sharedtypes.TagNumber) string {
	// Format tag display part
	tagDisplayPart := ""
	if tagNumber != nil && *tagNumber > 0 {
		tagDisplayPart = fmt.Sprintf(" %s %d", tagPrefix, *tagNumber)
	}

	// The line format for scheduled rounds: <@USER_ID> Tag: N (no score data)
	return fmt.Sprintf("<@%s>%s", userID, tagDisplayPart)
}
