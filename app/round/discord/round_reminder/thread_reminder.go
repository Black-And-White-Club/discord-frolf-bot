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

// SendRoundReminder processes the Discord round reminder event and creates a thread with mentions
func (rm *roundReminderManager) SendRoundReminder(ctx context.Context, payload *roundevents.DiscordReminderPayload) (RoundReminderOperationResult, error) {
	return rm.operationWrapper(ctx, "SendRoundReminder", func(ctx context.Context) (RoundReminderOperationResult, error) {
		rm.logger.InfoContext(ctx, "Handling round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("reminder_type", payload.ReminderType),
			attr.String("channel_id", payload.DiscordChannelID))

		discordMessageID := payload.EventMessageID
		if discordMessageID == "" {
			err := fmt.Errorf("no message ID provided in payload")
			rm.logger.ErrorContext(ctx, err.Error(), attr.RoundID("round_id", payload.RoundID))
			return RoundReminderOperationResult{Error: err}, err
		}

		_, err := rm.session.GetChannel(payload.DiscordChannelID)
		if err != nil {
			err = fmt.Errorf("failed to get channel: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(),
				attr.String("channel_id", payload.DiscordChannelID),
				attr.Error(err))
			return RoundReminderOperationResult{Error: err}, err
		}

		threadName := fmt.Sprintf("⏰ 1 Hour Reminder: %s", payload.RoundTitle)
		threadData := &discordgo.ThreadStart{
			Name: threadName,
			Type: discordgo.ChannelTypeGuildPublicThread,
		}

		thread, err := rm.session.MessageThreadStartComplex(payload.DiscordChannelID, discordMessageID, threadData)
		if err != nil {
			if strings.Contains(err.Error(), "Thread already exists") {
				rm.logger.WarnContext(ctx, "Thread already exists, attempting to get existing thread",
					attr.String("channel_id", payload.DiscordChannelID),
					attr.String("message_id", discordMessageID))

				threads, err := rm.session.ThreadsActive(payload.DiscordGuildID)
				if err != nil {
					err = fmt.Errorf("failed to get active threads: %w", err)
					rm.logger.ErrorContext(ctx, err.Error(),
						attr.String("guild_id", payload.DiscordGuildID),
						attr.Error(err))
					return RoundReminderOperationResult{Error: err}, err
				}

				if threads == nil || threads.Threads == nil {
					err := fmt.Errorf("failed to retrieve active threads")
					rm.logger.ErrorContext(ctx, err.Error(), attr.String("guild_id", payload.DiscordGuildID))
					return RoundReminderOperationResult{Error: err}, err
				}

				for _, t := range threads.Threads {
					if t.ParentID == payload.DiscordChannelID && t.Name == threadName {
						thread = t
						break
					}
				}

				if thread == nil {
					err := fmt.Errorf("could not find existing thread")
					rm.logger.ErrorContext(ctx, err.Error(),
						attr.String("channel_id", payload.DiscordChannelID),
						attr.String("thread_name", threadName))
					return RoundReminderOperationResult{Error: err}, err
				}
			} else {
				err = fmt.Errorf("failed to create thread: %w", err)
				rm.logger.ErrorContext(ctx, err.Error(),
					attr.String("channel_id", payload.DiscordChannelID),
					attr.String("message_id", discordMessageID),
					attr.Error(err))
				return RoundReminderOperationResult{Error: err}, err
			}
		}

		// Mention users
		var userMentions []string
		for _, userID := range payload.UserIDs {
			userMentions = append(userMentions, fmt.Sprintf("<@%s>", userID))
		}

		startTimeStr := "TBD"
		if payload.StartTime != nil {
			startTimeStr = time.Time(*payload.StartTime).Format("Mon Jan 2 at 3:04 PM")
		}

		locationStr := "TBD"
		if payload.Location != nil && *payload.Location != "" {
			locationStr = string(*payload.Location)
		}

		reminderMessage := fmt.Sprintf("**1 HOUR REMINDER** 🏆\n\n%s\n\nRound \"%s\" is starting in 1 hour (%s) at %s!",
			strings.Join(userMentions, " "),
			payload.RoundTitle,
			startTimeStr,
			locationStr)

		_, err = rm.session.ChannelMessageSend(thread.ID, reminderMessage)
		if err != nil {
			err = fmt.Errorf("failed to send message to thread: %w", err)
			rm.logger.ErrorContext(ctx, err.Error(),
				attr.String("thread_id", thread.ID),
				attr.Error(err))
			return RoundReminderOperationResult{Error: err}, err
		}

		rm.logger.InfoContext(ctx, "Successfully sent round reminder",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("thread_id", thread.ID),
			attr.Int("mentioned_users", len(payload.UserIDs)))

		return RoundReminderOperationResult{Success: true}, nil
	})
}
