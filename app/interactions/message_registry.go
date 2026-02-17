// interactions/message_registry.go
package interactions

import (
	"context"
	"log/slog"
	"runtime/debug"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

type discordgoAdder interface {
	AddHandler(handler interface{}) func()
}

// MessageHandlerCreate defines the signature for message creation handlers with context
type MessageHandlerCreate func(ctx context.Context, s discord.Session, m *discordgo.MessageCreate)

// MessageRegistry manages message event handlers
type MessageRegistry struct {
	messageCreateHandlers []MessageHandlerCreate
	logger                *slog.Logger
}

// NewMessageRegistry creates a new MessageRegistry
func NewMessageRegistry(logger *slog.Logger) *MessageRegistry {
	return &MessageRegistry{
		messageCreateHandlers: make([]MessageHandlerCreate, 0),
		logger:                logger,
	}
}

// RegisterMessageCreateHandler registers a handler for MessageCreate events
func (r *MessageRegistry) RegisterMessageCreateHandler(handler MessageHandlerCreate) {
	r.messageCreateHandlers = append(r.messageCreateHandlers, handler)
}

// RegisterWithSession registers all handlers with the Discord session
func (r *MessageRegistry) RegisterWithSession(session discordgoAdder, wrapperSession discord.Session) {
	// Register MessageCreate handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageCreate) {
		if e == nil || e.Message == nil {
			if r.logger != nil {
				r.logger.Warn("Ignoring MessageCreate event with nil payload")
			}
			return
		}

		if e.Author == nil {
			if r.logger != nil {
				r.logger.Warn("Ignoring MessageCreate event with nil author",
					slog.String("channel_id", e.ChannelID),
					slog.String("message_id", e.ID))
			}
			return
		}

		// 1. Safety Check: Ignore messages sent by the bot itself to prevent recursion
		var authorID string
		if e.Author != nil {
			authorID = e.Author.ID
		}

		var sessionUserID string
		if s != nil && s.State != nil && s.State.User != nil {
			sessionUserID = s.State.User.ID
		}

		if authorID != "" && sessionUserID != "" && authorID == sessionUserID {
			return
		}

		// 2. Context Initialization: Create a base context for this message event chain
		ctx := context.Background()

		if r.logger != nil {
			r.logger.Debug("Processing MessageCreate handlers",
				slog.String("author_id", authorID),
				slog.String("channel_id", e.ChannelID),
				slog.String("message_id", e.ID))
		}

		// 3. Execution: Distribute the message to all registered handlers
		for idx, handler := range r.messageCreateHandlers {
			r.runMessageCreateHandler(ctx, wrapperSession, e, idx, handler)
		}
	})
}

func (r *MessageRegistry) runMessageCreateHandler(ctx context.Context, wrapperSession discord.Session, e *discordgo.MessageCreate, index int, handler MessageHandlerCreate) {
	if handler == nil {
		if r.logger != nil {
			r.logger.Warn("Skipping nil MessageCreate handler", slog.Int("handler_index", index))
		}
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil && r.logger != nil {
			r.logger.Error("Recovered panic from MessageCreate handler",
				slog.Int("handler_index", index),
				slog.String("channel_id", e.ChannelID),
				slog.String("message_id", e.ID),
				slog.Any("panic", recovered),
				slog.String("stack_trace", string(debug.Stack())))
		}
	}()

	handler(ctx, wrapperSession, e)
}
