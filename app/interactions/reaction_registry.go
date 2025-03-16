// interactions/reaction_registry.go
package interactions

import (
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

// ReactionRegistry manages reaction event handlers
type ReactionRegistry struct {
	messageReactionAddHandlers    []func(s discord.Session, r *discordgo.MessageReactionAdd)
	messageReactionRemoveHandlers []func(s discord.Session, r *discordgo.MessageReactionRemove)
}

// NewReactionRegistry creates a new ReactionRegistry
func NewReactionRegistry() *ReactionRegistry {
	return &ReactionRegistry{
		messageReactionAddHandlers:    make([]func(s discord.Session, r *discordgo.MessageReactionAdd), 0),
		messageReactionRemoveHandlers: make([]func(s discord.Session, r *discordgo.MessageReactionRemove), 0),
	}
}

// RegisterMessageReactionAddHandler registers a handler for MessageReactionAdd events
func (r *ReactionRegistry) RegisterMessageReactionAddHandler(handler func(s discord.Session, r *discordgo.MessageReactionAdd)) {
	r.messageReactionAddHandlers = append(r.messageReactionAddHandlers, handler)
}

// RegisterMessageReactionRemoveHandler registers a handler for MessageReactionRemove events
func (r *ReactionRegistry) RegisterMessageReactionRemoveHandler(handler func(s discord.Session, r *discordgo.MessageReactionRemove)) {
	r.messageReactionRemoveHandlers = append(r.messageReactionRemoveHandlers, handler)
}

// RegisterWithSession registers all handlers with the Discord session
func (r *ReactionRegistry) RegisterWithSession(session *discordgo.Session, wrapperSession discord.Session) {
	// Register MessageReactionAdd handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageReactionAdd) {
		for _, handler := range r.messageReactionAddHandlers {
			handler(wrapperSession, e)
		}
	})

	// Register MessageReactionRemove handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageReactionRemove) {
		for _, handler := range r.messageReactionRemoveHandlers {
			handler(wrapperSession, e)
		}
	})
}
