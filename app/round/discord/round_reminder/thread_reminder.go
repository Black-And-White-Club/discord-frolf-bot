package roundreminder

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// SendRoundReminder processes the Discord round reminder event and creates a thread with mentions
func (rm *roundReminderManager) SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayload) (bool, error) {
	rm.logger.Info(ctx, "Handling round reminder",
		attr.String("round_id", strconv.FormatInt(int64(payload.RoundID), 10)),
		attr.String("reminder_type", payload.ReminderType),
		attr.String("channel_id", payload.DiscordChannelID))

	// Check for message ID first
	messageID := payload.EventMessageID
	if messageID == "" {
		rm.logger.Error(ctx, "No message ID provided in payload",
			attr.String("round_id", strconv.FormatInt(int64(payload.RoundID), 10)))
		return false, fmt.Errorf("no message ID provided in payload")
	}
	// Get the channel to verify it exists
	_, err := rm.session.GetChannel(payload.DiscordChannelID)
	if err != nil {
		rm.logger.Error(ctx, "Failed to get channel",
			attr.String("channel_id", payload.DiscordChannelID),
			attr.Error(err))
		return false, fmt.Errorf("failed to get channel: %w", err)
	}

	// Create thread name for 1-hour reminder
	threadName := fmt.Sprintf("⏰ 1 Hour Reminder: %s", payload.RoundTitle)

	// Create thread in the channel
	threadData := &discordgo.ThreadStart{
		Name: threadName,
		Type: discordgo.ChannelTypeGuildPublicThread,
	}

	thread, err := rm.session.MessageThreadStartComplex(payload.DiscordChannelID, string(messageID), threadData)
	if err != nil {
		// Check if the error is due to a thread already existing
		if strings.Contains(err.Error(), "Thread already exists") {
			rm.logger.Warn(ctx, "Thread already exists, attempting to get existing thread",
				attr.String("channel_id", payload.DiscordChannelID),
				attr.String("message_id", string(messageID)))

			// Get existing threads in the channel
			threads, err := rm.session.ThreadsActive(payload.DiscordGuildID)
			if err != nil {
				rm.logger.Error(ctx, "Failed to get active threads",
					attr.String("guild_id", payload.DiscordGuildID),
					attr.Error(err))
				return false, fmt.Errorf("failed to get active threads: %w", err)
			}

			// Ensure threads.Threads is not nil
			if threads == nil || threads.Threads == nil {
				rm.logger.Error(ctx, "ThreadsActive returned nil", attr.String("guild_id", payload.DiscordGuildID))
				return false, fmt.Errorf("failed to retrieve active threads")
			}

			// Find the thread for this message
			var existingThread *discordgo.Channel
			for _, t := range threads.Threads {
				if t.ParentID == payload.DiscordChannelID && t.Name == threadName {
					existingThread = t
					break
				}
			}

			if existingThread == nil {
				rm.logger.Error(ctx, "Could not find existing thread",
					attr.String("channel_id", payload.DiscordChannelID),
					attr.String("thread_name", threadName))
				return false, fmt.Errorf("could not find existing thread")
			}

			thread = existingThread
		} else {
			rm.logger.Error(ctx, "Failed to create thread",
				attr.String("channel_id", payload.DiscordChannelID),
				attr.String("message_id", string(messageID)),
				attr.Error(err))
			return false, fmt.Errorf("failed to create thread: %w", err)
		}
	}

	// Format the reminder message with user mentions
	var userMentions []string
	for _, userID := range payload.UserIDs {
		userMentions = append(userMentions, fmt.Sprintf("<@%s>", userID))
	}

	// Format the time string
	startTimeStr := "TBD"
	if payload.StartTime != nil {
		startTime := time.Time(*payload.StartTime)
		startTimeStr = startTime.Format("Mon Jan 2 at 3:04 PM")
	}

	// Format the location string
	locationStr := "TBD"
	if payload.Location != nil && *payload.Location != "" {
		locationStr = string(*payload.Location)
	}

	// Create the 1-hour reminder message
	reminderMessage := fmt.Sprintf("**1 HOUR REMINDER** 🏆\n\n%s\n\nRound \"%s\" is starting in 1 hour (%s) at %s!",
		strings.Join(userMentions, " "),
		payload.RoundTitle,
		startTimeStr,
		locationStr)

	// Send the message to the thread
	_, err = rm.session.ChannelMessageSend(thread.ID, reminderMessage)
	if err != nil {
		rm.logger.Error(ctx, "Failed to send message to thread",
			attr.String("thread_id", thread.ID),
			attr.Error(err))
		return false, fmt.Errorf("failed to send message to thread: %w", err)
	}

	rm.logger.Info(ctx, "Successfully sent round reminder",
		attr.String("round_id", strconv.FormatInt(int64(payload.RoundID), 10)),
		attr.String("thread_id", thread.ID),
		attr.Int("mentioned_users", len(payload.UserIDs)))

	return true, nil // Success
}
