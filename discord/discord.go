package discord

import (
	"github.com/bwmarrin/discordgo"
)

// Discord defines the interface for interacting with Discord.
type Discord interface {
	UserChannelCreate(userID string) (*discordgo.Channel, error)
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	MessageReactionAdd(channelID, messageID, emojiID string) error
	GetChannel(channelID string) (*discordgo.Channel, error)
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) ([]*discordgo.Message, error)
	// Expose a way to access state-related functionality through a separate interface.
	State() State

	// Expose a way to access the underlying session
	GetSession() *discordgo.Session
	GetBotUser() (*discordgo.User, error)
}

// State defines an interface that provides access to the Discord state.
type State interface {
	UserChannelPermissions(userID string, channelID string) (apermissions int64, err error)
	Guild(guildID string) (*discordgo.Guild, error)
	Member(guildID, userID string) (*discordgo.Member, error)
	Channel(channelID string) (*discordgo.Channel, error)
	// Add more methods from discordgo.State as needed by your application.
}

// DiscordSession is an implementation of the Discord interface.
type DiscordSession struct {
	session *discordgo.Session
}

// Session wraps discordgo.Session to implement the Discord interface.
type Session struct {
	*discordgo.Session
}

// NewDiscordSession creates a new DiscordSession.
func NewDiscordSession(session *discordgo.Session) *DiscordSession {
	return &DiscordSession{session: session}
}

// UserChannelCreate creates a new DM channel with a user.
func (d *DiscordSession) UserChannelCreate(userID string) (*discordgo.Channel, error) {
	return d.session.UserChannelCreate(userID)
}

// ChannelMessageSend sends a message to a channel.
func (d *DiscordSession) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.ChannelMessageSend(channelID, content, options...)
}

// GuildMemberRoleAdd adds a role to a guild member.
func (d *DiscordSession) GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	return d.session.GuildMemberRoleAdd(guildID, userID, roleID, options...)
}

// State returns a State interface for accessing state-related information.
func (d *DiscordSession) State() State {
	return d.session.State
}

// GetSession returns the underlying discordgo session.
func (d *DiscordSession) GetSession() *discordgo.Session {
	return d.session
}

// MessageReactionAdd handles adding a reaction to a message.
func (d *DiscordSession) MessageReactionAdd(channelID, messageID, emojiID string) error {
	return d.session.MessageReactionAdd(channelID, messageID, emojiID)
}

// Compile-time check to ensure *discordgo.State implements the State interface.
var _ State = (*discordgo.State)(nil)

// MessageReactionAdd represents a reaction add event.
type MessageReactionAdd struct {
	*discordgo.MessageReactionAdd
}

// MessageCreate represents a message create event.
type MessageCreate struct {
	*discordgo.MessageCreate
}

// Add GetBotUser implementation to DiscordSession
func (d *DiscordSession) GetBotUser() (*discordgo.User, error) {
	return d.session.User("@me")
}

// GetChannel retrieves a channel by its ID.
func (d *DiscordSession) GetChannel(channelID string) (*discordgo.Channel, error) {
	return d.session.Channel(channelID)
}

// ChannelMessages fetches messages from a channel.
func (s *Session) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) ([]*discordgo.Message, error) {
	return s.Session.ChannelMessages(channelID, limit, beforeID, afterID, aroundID)
}

// ChannelMessages fetches messages from a channel.
func (s *DiscordSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) ([]*discordgo.Message, error) {
	return s.session.ChannelMessages(channelID, limit, beforeID, afterID, aroundID)
}
