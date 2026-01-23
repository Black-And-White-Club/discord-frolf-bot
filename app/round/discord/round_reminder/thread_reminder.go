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

// SendRoundReminder sends a 1-hour round reminder to the appropriate Discord thread or channel.
func (rm *roundReminderManager) SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) (RoundReminderOperationResult, error) {
	return rm.operationWrapper(ctx, "SendRoundReminder", func(ctx context.Context) (RoundReminderOperationResult, error) {
		rm.logPayloadDetails(ctx, payload)

		// Early payload validation
		if err := rm.validatePayload(ctx, payload); err != nil {
			return RoundReminderOperationResult{Error: err}, err
		}

		// Resolve the channel ID to send the reminder to
		resolvedChannelID := rm.resolveChannelID(ctx, payload)

		// Validate channel existence
		if _, err := rm.session.GetChannel(resolvedChannelID); err != nil {
			err = fmt.Errorf("failed to get channel: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(), attr.String("channel_id", resolvedChannelID), attr.Error(err))
			return RoundReminderOperationResult{Error: err}, err
		}

		// Build message content
		reminderMessage := rm.buildReminderMessage(payload)
		threadName := fmt.Sprintf("⏰ 1 Hour Reminder: %s", payload.RoundTitle)

		// Find or create the thread
		thread, err := rm.findOrCreateThread(ctx, resolvedChannelID, payload.EventMessageID, threadName)
		if err != nil {
			// If thread creation fails, fallback to sending to main channel
			rm.logger.WarnContext(ctx, "Thread unavailable, sending to main channel as fallback")
			if sendErr := rm.sendMessageToChannel(ctx, resolvedChannelID, reminderMessage); sendErr != nil {
				err = fmt.Errorf("failed to send reminder to main channel as fallback: %w", sendErr)
				rm.logger.ErrorContext(ctx, err.Error())
				return RoundReminderOperationResult{Error: err}, err
			}
			return RoundReminderOperationResult{Success: true}, nil
		}

		// Send the reminder to the thread
		if err := rm.sendMessageToChannel(ctx, thread.ID, reminderMessage); err != nil {
			err = fmt.Errorf("failed to send reminder message to thread: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(), attr.String("thread_id", thread.ID))
			return RoundReminderOperationResult{Error: err}, err
		}

		rm.logger.InfoContext(ctx, "Successfully sent round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("thread_id", thread.ID),
			attr.Int("mentioned_users", len(payload.UserIDs)))

		return RoundReminderOperationResult{Success: true}, nil
	})
}

// -------------------- Helper Functions --------------------

// logPayloadDetails logs relevant payload information
func (rm *roundReminderManager) logPayloadDetails(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) {
	startTimeStr := "nil"
	if payload.StartTime != nil {
		startTimeStr = payload.StartTime.AsTime().Format(time.RFC3339)
	}

	locationStr := "nil"
	if payload.Location != "" {
		locationStr = string(payload.Location)
	}

	rm.logger.InfoContext(ctx, "Processing round reminder",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("reminder_type", payload.ReminderType),
		attr.String("channel_id", payload.DiscordChannelID),
		attr.String("start_time", startTimeStr),
		attr.String("location", locationStr),
		attr.Int("user_count", len(payload.UserIDs)),
	)
}

// validatePayload performs basic validation on the payload
func (rm *roundReminderManager) validatePayload(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) error {
	if payload.EventMessageID == "" {
		err := fmt.Errorf("no message ID provided in payload")
		rm.logger.ErrorContext(ctx, err.Error(), attr.RoundID("round_id", payload.RoundID))
		return err
	}
	if payload.RoundTitle == "" {
		err := fmt.Errorf("no round title provided in payload")
		rm.logger.WarnContext(ctx, err.Error())
	}
	return nil
}

// resolveChannelID determines which channel ID to use for sending the reminder
func (rm *roundReminderManager) resolveChannelID(ctx context.Context, payload *roundevents.DiscordReminderPayloadV1) string {
	if payload.DiscordChannelID != "" {
		return payload.DiscordChannelID
	}

	if rm.guildConfigResolver != nil && payload.DiscordGuildID != "" {
		guildConfig, err := rm.guildConfigResolver.GetGuildConfigWithContext(ctx, payload.DiscordGuildID)
		if err == nil && guildConfig != nil && guildConfig.EventChannelID != "" {
			rm.logger.InfoContext(ctx, "Resolved channel ID from guild config", attr.String("channel_id", guildConfig.EventChannelID))
			return guildConfig.EventChannelID
		}
		rm.logger.WarnContext(ctx, "Failed to resolve channel ID from guild config", attr.Error(err))
	}

	return ""
}

// buildReminderMessage constructs the reminder message with mentions and formatted start time/location
func (rm *roundReminderManager) buildReminderMessage(payload *roundevents.DiscordReminderPayloadV1) string {
	var sb strings.Builder

	if len(payload.UserIDs) > 0 {
		for _, userID := range payload.UserIDs {
			sb.WriteString(fmt.Sprintf("<@%s> ", string(userID)))
		}
		sb.WriteString("\n\n")
	}

	startTimeStr := "TBD"
	if payload.StartTime != nil {
		startTimeStr = fmt.Sprintf("<t:%d:f>", payload.StartTime.AsTime().Unix())
	}

	locationStr := "TBD"
	if payload.Location != "" {
		locationStr = string(payload.Location)
	}

	sb.WriteString(fmt.Sprintf("**1 HOUR REMINDER** 🏆\n\nRound \"%s\" is starting in 1 hour (%s) at %s!",
		payload.RoundTitle, startTimeStr, locationStr))

	return sb.String()
}

// findOrCreateThread finds an existing thread or creates a new one
func (rm *roundReminderManager) findOrCreateThread(ctx context.Context, channelID, messageID, threadName string) (*discordgo.Channel, error) {
	// Check if the message already has a thread
	if message, err := rm.session.ChannelMessage(channelID, messageID); err == nil && message.Thread != nil {
		rm.logger.InfoContext(ctx, "Found existing thread via message", attr.String("thread_id", message.Thread.ID))
		return message.Thread, nil
	}

	// Search active threads in the guild
	if threadsResp, err := rm.session.ThreadsActive(channelID); err == nil && threadsResp != nil {
		for _, t := range threadsResp.Threads {
			if t.ParentID == channelID && t.Name == threadName {
				rm.logger.InfoContext(ctx, "Found existing thread via search", attr.String("thread_id", t.ID))
				return t, nil
			}
		}
	}

	// Attempt to create a new thread
	newThread, err := rm.session.MessageThreadStartComplex(channelID, messageID, &discordgo.ThreadStart{
		Name: threadName,
		Type: discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		if strings.Contains(err.Error(), "Thread already exists") || strings.Contains(err.Error(), "160004") {
			// Attempt to fetch again after race condition
			if message, msgErr := rm.session.ChannelMessage(channelID, messageID); msgErr == nil && message.Thread != nil {
				rm.logger.InfoContext(ctx, "Found thread after creation race", attr.String("thread_id", message.Thread.ID))
				return message.Thread, nil
			}
			return nil, fmt.Errorf("thread exists but cannot be retrieved: %w", err)
		}
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	rm.logger.InfoContext(ctx, "Successfully created new thread", attr.String("thread_id", newThread.ID))
	return newThread, nil
}

// sendMessageToChannel sends a message to the given Discord channel ID
func (rm *roundReminderManager) sendMessageToChannel(ctx context.Context, channelID, message string) error {
	_, err := rm.session.ChannelMessageSend(channelID, message)
	return err
}
