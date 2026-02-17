package scorecardupload

import (
	"context"
	"log/slog"
	"runtime/debug"
	"strings"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

const (
	fileUploadWorkerCount = 4
	fileUploadQueueSize   = 64
)

type fileUploadTask struct {
	session discord.Session
	message *discordgo.MessageCreate
}

type fileUploadDispatcher struct {
	manager ScorecardUploadManager
	logger  *slog.Logger
	queue   chan fileUploadTask
}

func newFileUploadDispatcher(manager ScorecardUploadManager, logger *slog.Logger, workers, queueSize int) *fileUploadDispatcher {
	if workers <= 0 {
		workers = 1
	}
	if queueSize <= 0 {
		queueSize = 1
	}

	d := &fileUploadDispatcher{
		manager: manager,
		logger:  logger,
		queue:   make(chan fileUploadTask, queueSize),
	}

	for range workers {
		go func() {
			for task := range d.queue {
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							if d.logger != nil {
								d.logger.Error("Recovered panic in scorecard upload worker",
									attr.Any("panic", recovered),
									attr.String("stack_trace", string(debug.Stack())),
								)
							}
						}
					}()

					d.manager.HandleFileUploadMessage(task.session, task.message)
				}()
			}
		}()
	}

	return d
}

func (d *fileUploadDispatcher) submit(task fileUploadTask) bool {
	select {
	case d.queue <- task:
		return true
	default:
		return false
	}
}

func RegisterHandlers(registry *interactions.Registry, messageRegistry *interactions.MessageRegistry, manager ScorecardUploadManager) {
	dispatcher := newFileUploadDispatcher(manager, slog.Default(), fileUploadWorkerCount, fileUploadQueueSize)

	// Scorecard upload button (from round embeds)
	registry.RegisterMutatingHandler("round_upload_scorecard|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		if i == nil || i.Interaction == nil {
			slog.WarnContext(ctx, "Ignoring scorecard upload button with nil interaction payload")
			return
		}

		userID := interactionUserIDFromCreate(i)
		if userID == "" {
			slog.WarnContext(ctx, "Ignoring scorecard upload button with missing user",
				attr.String("interaction_id", i.ID))
			return
		}

		slog.InfoContext(ctx, "Handling scorecard upload button press",
			attr.String("custom_id", i.MessageComponentData().CustomID),
			attr.String("interaction_id", i.ID),
			attr.String("user_id", userID),
		)
		manager.HandleScorecardUploadButton(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	// Scorecard upload modal submission
	registry.RegisterMutatingHandler("scorecard_upload_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		if i == nil || i.Interaction == nil {
			slog.WarnContext(ctx, "Ignoring scorecard upload modal with nil interaction payload")
			return
		}

		userID := interactionUserIDFromCreate(i)
		if userID == "" {
			slog.WarnContext(ctx, "Ignoring scorecard upload modal with missing user",
				attr.String("interaction_id", i.ID))
			return
		}

		slog.InfoContext(ctx, "Handling scorecard upload modal submission",
			attr.String("interaction_id", i.ID),
			attr.String("user_id", userID),
		)
		manager.HandleScorecardUploadModalSubmit(ctx, i)
	}, interactions.MutatingHandlerPolicy{RequiredPermission: interactions.PlayerRequired, RequiresSetup: true})

	// File upload message listener - adapter to provide context to legacy handler
	messageRegistry.RegisterMessageCreateHandler(func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate) {
		if m == nil || m.Message == nil {
			slog.WarnContext(ctx, "Ignoring scorecard upload message with nil payload")
			return
		}

		if m.Author == nil {
			slog.WarnContext(ctx, "Ignoring scorecard upload message with nil author",
				attr.String("channel_id", m.ChannelID),
				attr.String("message_id", m.ID))
			return
		}
		if !isLikelyScorecardUploadMessage(m) {
			return
		}

		if dispatcher.submit(fileUploadTask{session: s, message: m}) {
			return
		}

		channelID := ""
		userID := ""
		if m != nil {
			channelID = m.ChannelID
			if m.Author != nil {
				userID = m.Author.ID
			}
		}

		slog.WarnContext(ctx, "Scorecard upload queue is full; rejecting message",
			attr.String("channel_id", channelID),
			attr.String("user_id", userID))
		if channelID != "" && s != nil {
			_, _ = s.ChannelMessageSend(channelID, "Upload queue is busy. Please try again in a few seconds.")
		}
	})
}

func isLikelyScorecardUploadMessage(m *discordgo.MessageCreate) bool {
	if m == nil || m.Message == nil || m.Author == nil || m.Author.Bot {
		return false
	}
	for _, attachment := range m.Attachments {
		filename := strings.ToLower(attachment.Filename)
		if strings.HasSuffix(filename, ".csv") || strings.HasSuffix(filename, ".xlsx") {
			return true
		}
	}
	return false
}

func interactionUserIDFromCreate(i *discordgo.InteractionCreate) string {
	if i == nil {
		return ""
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}
