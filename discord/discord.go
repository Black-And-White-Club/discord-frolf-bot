package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Session defines the interface for interacting with Discord.
type Session interface {
	UserChannelCreate(recipientID string, options ...discordgo.RequestOption) (st *discordgo.Channel, err error)
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	MessageReactionAdd(channelID, messageID, emojiID string) error
	GetChannel(channelID string) (*discordgo.Channel, error)
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) ([]*discordgo.Message, error)
	CreateThread(channelID, threadName string) (*discordgo.Channel, error)
	AddUserToThread(threadID, userID string) error
	GetBotUser() (*discordgo.User, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildScheduledEventCreate(guildID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	GuildScheduledEventEdit(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	ThreadStartComplex(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (ch *discordgo.Channel, err error)
	AddHandler(handler interface{}) func()
	Open() error
	Close() error
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
}

// State defines an interface that provides access to the Discord state.
type State interface {
	UserChannelPermissions(userID string, channelID string) (apermissions int64, err error)
	Guild(guildID string) (*discordgo.Guild, error)
	Member(guildID, userID string) (*discordgo.Member, error)
	Channel(channelID string) (*discordgo.Channel, error)
}

// DiscordSession is an implementation of the Session interface.
type DiscordSession struct {
	session *discordgo.Session
	logger  *slog.Logger
}

// DiscordState is an implementation of the State interface.  Again, consider
// removing this if you can.
type DiscordState struct {
	state *discordgo.State
}

// NewDiscordSession creates a new DiscordSession.
func NewDiscordSession(session *discordgo.Session, logger *slog.Logger) *DiscordSession {
	return &DiscordSession{session: session, logger: logger}
}

// NewDiscordState creates a new DiscordState.
func NewDiscordState(state *discordgo.State) *DiscordState {
	return &DiscordState{state: state}
}

// AddHandler wraps the discordgo AddHandler method.
func (d *DiscordSession) AddHandler(handler interface{}) func() {
	return d.session.AddHandler(handler)
}

// Open wraps the discordgo Open method.
func (d *DiscordSession) Open() error {
	d.logger.Info("Opening discord websocket connection")
	return d.session.Open()
}

// Close wraps the discordgo Close method.
func (d *DiscordSession) Close() error {
	d.logger.Info("Closing discord websocket connection")
	return d.session.Close()
}

// UserChannelCreate creates a new DM channel with a user.
func (d *DiscordSession) UserChannelCreate(recipientID string, options ...discordgo.RequestOption) (st *discordgo.Channel, err error) {
	return d.session.UserChannelCreate(recipientID, options...)
}

// ChannelMessageSend sends a message to a channel.
func (d *DiscordSession) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.ChannelMessageSend(channelID, content, options...)
}

// GuildMemberRoleAdd adds a role to a guild member.
func (d *DiscordSession) GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	return d.session.GuildMemberRoleAdd(guildID, userID, roleID, options...)
}

// MessageReactionAdd handles adding a reaction to a message.
func (d *DiscordSession) MessageReactionAdd(channelID, messageID, emojiID string) error {
	return d.session.MessageReactionAdd(channelID, messageID, emojiID)
}

// GetChannel retrieves a channel by its ID.
func (d *DiscordSession) GetChannel(channelID string) (*discordgo.Channel, error) {
	return d.session.Channel(channelID)
}

// ChannelMessages fetches messages from a channel.
func (d *DiscordSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string) ([]*discordgo.Message, error) {
	return d.session.ChannelMessages(channelID, limit, beforeID, afterID, aroundID)
}

// CreateThread creates a new thread in a channel.
func (d *DiscordSession) CreateThread(channelID, threadName string) (*discordgo.Channel, error) {
	// Simplified, as your original version was incomplete.  Adjust as needed.
	return d.session.ThreadStartComplex(channelID, &discordgo.ThreadStart{
		Name: threadName,
		Type: discordgo.ChannelTypeGuildPublicThread, // Or another appropriate type
	})
}

// AddUserToThread adds a user to a thread.
func (d *DiscordSession) AddUserToThread(threadID, userID string) error {
	return d.session.ThreadMemberAdd(threadID, userID)
}

// GetBotUser retrieves the bot user.
func (d *DiscordSession) GetBotUser() (*discordgo.User, error) {
	return d.session.User("@me")
}

// ChannelMessageSendComplex sends a complex message to a channel.
func (d *DiscordSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.ChannelMessageSendComplex(channelID, data, options...)
}

// GuildScheduledEventCreate creates a scheduled event for a guild.
func (d *DiscordSession) GuildScheduledEventCreate(guildID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	return d.session.GuildScheduledEventCreate(guildID, params, options...)
}

// GuildScheduledEventEdit edits a scheduled event.
func (d *DiscordSession) GuildScheduledEventEdit(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	return d.session.GuildScheduledEventEdit(guildID, eventID, params, options...)
}

// ThreadStartComplex starts a new thread in a channel.
func (d *DiscordSession) ThreadStartComplex(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return d.session.ThreadStartComplex(channelID, data, options...)
}

// UserChannelPermissions checks the permissions of a user in a channel.
func (d *DiscordState) UserChannelPermissions(userID string, channelID string) (int64, error) {
	return d.state.UserChannelPermissions(userID, channelID)
}

// Guild retrieves a guild by its ID.
func (d *DiscordState) Guild(guildID string) (*discordgo.Guild, error) {
	return d.state.Guild(guildID)
}

// Member retrieves a member from a guild.
func (d *DiscordState) Member(guildID, userID string) (*discordgo.Member, error) {
	return d.state.Member(guildID, userID)
}

// Channel retrieves a channel by its ID.
func (d *DiscordState) Channel(channelID string) (*discordgo.Channel, error) {
	return d.state.Channel(channelID)
}

// InteractionRespond responds to an interaction.
func (d *DiscordSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	return d.session.InteractionRespond(interaction, resp, options...)
}

// InteractionResponseEdit edits an interaction response.
func (d *DiscordSession) InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.InteractionResponseEdit(interaction, newresp, options...)
}

// Add the new methods to the Discord Session
func (d *DiscordSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	return d.session.ApplicationCommandCreate(appID, guildID, cmd, options...)
}

func (d *DiscordSession) ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	return d.session.ApplicationCommands(appID, guildID, options...)
}

func (d *DiscordSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	return d.session.ApplicationCommandDelete(appID, guildID, cmdID, options...)
}
