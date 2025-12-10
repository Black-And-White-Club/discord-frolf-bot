package interactions

import (
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

// MessageRegistry manages message event handlers
type MessageRegistry struct {
	messageCreateHandlers []func(s discord.Session, m *discordgo.MessageCreate)
}

// NewMessageRegistry creates a new MessageRegistry
func NewMessageRegistry() *MessageRegistry {
	return &MessageRegistry{
		messageCreateHandlers: make([]func(s discord.Session, m *discordgo.MessageCreate), 0),
	}
}

// RegisterMessageCreateHandler registers a handler for MessageCreate events
func (r *MessageRegistry) RegisterMessageCreateHandler(handler func(s discord.Session, m *discordgo.MessageCreate)) {
	r.messageCreateHandlers = append(r.messageCreateHandlers, handler)
}

// RegisterWithSession registers all handlers with the Discord session
func (r *MessageRegistry) RegisterWithSession(session *discordgo.Session, wrapperSession discord.Session) {
	// Register MessageCreate handler
	session.AddHandler(func(s *discordgo.Session, e *discordgo.MessageCreate) {
		for _, handler := range r.messageCreateHandlers {
			handler(wrapperSession, e)
		}
	})
}
