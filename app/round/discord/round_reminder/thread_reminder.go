package roundreminder

import (
	"context"
	"fmt"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func (rm *roundReminderManager) SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayload) (RoundReminderOperationResult, error) {
	return rm.operationWrapper(ctx, "SendRoundReminder", func(ctx context.Context) (RoundReminderOperationResult, error) {
		rm.logger.InfoContext(ctx, "Processing round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("reminder_type", payload.ReminderType),
			attr.String("channel_id", payload.DiscordChannelID))

		// Debug payload data with proper type handling
		rm.logger.InfoContext(ctx, "Reminder payload details",
			attr.String("start_time_is_nil", fmt.Sprintf("%t", payload.StartTime == nil)),
			attr.String("location_is_nil", fmt.Sprintf("%t", payload.Location == nil)),
			attr.Int("user_count", len(payload.UserIDs)))

		if payload.StartTime != nil {
			rm.logger.InfoContext(ctx, "StartTime value",
				attr.String("start_time", payload.StartTime.AsTime().Format(time.RFC3339)))
		}

		if payload.Location != nil {
			rm.logger.InfoContext(ctx, "Location value",
				attr.String("location", string(*payload.Location)))
		}

		// Early validation - fail fast before any Discord operations
		if payload.EventMessageID == "" {
			err := fmt.Errorf("no message ID provided in payload")
			rm.logger.ErrorContext(ctx, err.Error(), attr.RoundID("round_id", payload.RoundID))
			return RoundReminderOperationResult{Error: err}, err
		}

		// Validate channel exists - fail fast
		_, err := rm.session.GetChannel(payload.DiscordChannelID)
		if err != nil {
			err = fmt.Errorf("failed to get channel: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(),
				attr.String("channel_id", payload.DiscordChannelID),
				attr.Error(err))
			return RoundReminderOperationResult{Error: err}, err
		}

		// Build message content early - fail fast if data is invalid
		var userMentions []string
		for _, userID := range payload.UserIDs {
			userMentions = append(userMentions, fmt.Sprintf("<@%s>", string(userID)))
		}

		startTimeStr := "TBD"
		if payload.StartTime != nil {
			unixTimestamp := payload.StartTime.AsTime().Unix()
			startTimeStr = fmt.Sprintf("<t:%d:f>", unixTimestamp)
		}

		locationStr := "TBD"
		if payload.Location != nil && string(*payload.Location) != "" {
			locationStr = string(*payload.Location)
		}

		mentionsText := ""
		if len(userMentions) > 0 {
			mentionsText = strings.Join(userMentions, " ") + "\n\n"
		}

		reminderMessage := fmt.Sprintf("**1 HOUR REMINDER** 🏆\n\n%sRound \"%s\" is starting in 1 hour (%s) at %s!",
			mentionsText,
			string(payload.RoundTitle),
			startTimeStr,
			locationStr)

		threadName := fmt.Sprintf("⏰ 1 Hour Reminder: %s", payload.RoundTitle)

		// Try to find existing thread first (with simplified logic)
		var thread *discordgo.Channel

		// Method 1: Check if the original message already has a thread
		if message, msgErr := rm.session.ChannelMessage(payload.DiscordChannelID, payload.EventMessageID); msgErr == nil && message.Thread != nil {
			thread = message.Thread
			rm.logger.InfoContext(ctx, "Found existing thread via message", attr.String("thread_id", thread.ID))
		}

		// Method 2: Search active threads if no thread found via message
		if thread == nil {
			if threads, threadErr := rm.session.ThreadsActive(payload.DiscordGuildID); threadErr == nil && threads != nil && threads.Threads != nil {
				for _, t := range threads.Threads {
					if t.ParentID == payload.DiscordChannelID && t.Name == threadName {
						thread = t
						rm.logger.InfoContext(ctx, "Found existing thread via search", attr.String("thread_id", t.ID))
						break
					}
				}
			}
		}

		// Create thread only if absolutely necessary
		if thread == nil {
			rm.logger.InfoContext(ctx, "Creating new thread for reminder")

			newThread, createErr := rm.session.MessageThreadStartComplex(payload.DiscordChannelID, payload.EventMessageID, &discordgo.ThreadStart{
				Name: threadName,
				Type: discordgo.ChannelTypeGuildPublicThread,
			})

			if createErr != nil {
				// Handle thread already exists error gracefully
				if strings.Contains(createErr.Error(), "Thread already exists") || strings.Contains(createErr.Error(), "160004") {
					rm.logger.InfoContext(ctx, "Thread creation failed due to existing thread, attempting final search")

					// One more attempt to find the thread
					if message, msgErr := rm.session.ChannelMessage(payload.DiscordChannelID, payload.EventMessageID); msgErr == nil && message.Thread != nil {
						thread = message.Thread
						rm.logger.InfoContext(ctx, "Found thread after creation failure", attr.String("thread_id", thread.ID))
					}

					if thread == nil {
						// If we still can't find it, this might be a race condition
						// Log as warning but don't fail - the thread might exist but be inaccessible
						rm.logger.WarnContext(ctx, "Thread exists but cannot be found, will attempt to send to main channel")

						// Fallback: send to main channel instead of failing
						_, sendErr := rm.session.ChannelMessageSend(payload.DiscordChannelID, fmt.Sprintf("🧵 Thread creation issue - posting reminder here:\n\n%s", reminderMessage))
						if sendErr != nil {
							err = fmt.Errorf("failed to send reminder to main channel as fallback: %w", sendErr)
							rm.logger.ErrorContext(ctx, err.Error())
							return RoundReminderOperationResult{Error: err}, err
						}

						rm.logger.InfoContext(ctx, "Successfully sent reminder to main channel as fallback")
						return RoundReminderOperationResult{Success: true}, nil
					}
				} else {
					err = fmt.Errorf("failed to create thread: %w", createErr)
					rm.logger.ErrorContext(ctx, err.Error())
					return RoundReminderOperationResult{Error: err}, err
				}
			} else {
				thread = newThread
				rm.logger.InfoContext(ctx, "Successfully created new thread", attr.String("thread_id", thread.ID))
			}
		}

		// CRITICAL: Send the reminder message - this is the main operation
		// Everything after this point should be non-critical
		_, err = rm.session.ChannelMessageSend(thread.ID, reminderMessage)
		if err != nil {
			err = fmt.Errorf("failed to send reminder message to thread: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(), attr.String("thread_id", thread.ID))
			return RoundReminderOperationResult{Error: err}, err
		}

		// SUCCESS: The reminder has been sent successfully
		rm.logger.InfoContext(ctx, "Successfully sent round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("thread_id", thread.ID),
			attr.Int("mentioned_users", len(payload.UserIDs)))

		// Return success immediately - no additional operations that could fail
		return RoundReminderOperationResult{Success: true}, nil
	})
}
