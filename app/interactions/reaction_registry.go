// interactions/reaction_registry.go
package interactions

import (
	"context"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

// ReactionHandlerAdd defines the signature for reaction addition handlers with context
type ReactionHandlerAdd func(ctx context.Context, s discord.Session, r *discordgo.MessageReactionAdd)

// ReactionHandlerRemove defines the signature for reaction removal handlers with context
type ReactionHandlerRemove func(ctx context.Context, s discord.Session, r *discordgo.MessageReactionRemove)

// ReactionRegistry manages reaction event handlers
type ReactionRegistry struct {
	messageReactionAddHandlers    []ReactionHandlerAdd
	messageReactionRemoveHandlers []ReactionHandlerRemove
	logger                        *slog.Logger
}

// NewReactionRegistry creates a new ReactionRegistry
func NewReactionRegistry(logger *slog.Logger) *ReactionRegistry {
	return &ReactionRegistry{
		messageReactionAddHandlers:    make([]ReactionHandlerAdd, 0),
		messageReactionRemoveHandlers: make([]ReactionHandlerRemove, 0),
		logger:                        logger,
	}
}

// RegisterMessageReactionAddHandler registers a handler for MessageReactionAdd events
func (r *ReactionRegistry) RegisterMessageReactionAddHandler(handler ReactionHandlerAdd) {
	r.messageReactionAddHandlers = append(r.messageReactionAddHandlers, handler)
}

// RegisterMessageReactionRemoveHandler registers a handler for MessageReactionRemove events
func (r *ReactionRegistry) RegisterMessageReactionRemoveHandler(handler ReactionHandlerRemove) {
	r.messageReactionRemoveHandlers = append(r.messageReactionRemoveHandlers, handler)
}

// RegisterWithSession registers all handlers with the Discord session
func (r *ReactionRegistry) RegisterWithSession(session discordgoAdder, wrapperSession discord.Session) {
	// Register MessageReactionAdd handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageReactionAdd) {
		if e == nil {
			return
		}

		var sessionUserID string
		if s != nil && s.State != nil && s.State.User != nil {
			sessionUserID = s.State.User.ID
		}

		// Ignore bot reactions to avoid loops
		if e.UserID != "" && sessionUserID != "" && e.UserID == sessionUserID {
			return
		}

		ctx := context.Background() // Base context for the reaction event chain

		if r.logger != nil {
			r.logger.Info("Processing ReactionAdd handlers",
				slog.String("emoji", e.Emoji.Name),
				slog.String("message_id", e.MessageID))
		}

		for _, handler := range r.messageReactionAddHandlers {
			handler(ctx, wrapperSession, e)
		}
	})

	// Register MessageReactionRemove handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageReactionRemove) {
		if e == nil {
			return
		}

		var sessionUserID string
		if s != nil && s.State != nil && s.State.User != nil {
			sessionUserID = s.State.User.ID
		}

		if e.UserID != "" && sessionUserID != "" && e.UserID == sessionUserID {
			return
		}

		ctx := context.Background()

		if r.logger != nil {
			r.logger.Info("Processing ReactionRemove handlers",
				slog.String("emoji", e.Emoji.Name),
				slog.String("message_id", e.MessageID))
		}

		for _, handler := range r.messageReactionRemoveHandlers {
			handler(ctx, wrapperSession, e)
		}
	})
}
