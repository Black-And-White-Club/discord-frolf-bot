package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
)

// Session defines the interface for interacting with Discord.
type Session interface {
	UserChannelCreate(recipientID string, options ...discordgo.RequestOption) (st *discordgo.Channel, err error)
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
	GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	GuildMemberRoleRemove(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	MessageReactionAdd(channelID, messageID, emojiID string) error
	GetChannel(channelID string, options ...discordgo.RequestOption) (st *discordgo.Channel, err error)
	ChannelMessageEditComplex(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (st *discordgo.Message, err error)
	WebhookExecute(webhookID string, token string, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (st *discordgo.Message, err error)
	ChannelMessages(channelID string, limit int, beforeID string, afterID string, aroundID string, options ...discordgo.RequestOption) (st []*discordgo.Message, err error)
	ThreadsActive(channelID string, options ...discordgo.RequestOption) (threads *discordgo.ThreadsList, err error)
	MessageThreadStartComplex(channelID string, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (ch *discordgo.Channel, err error)
	ThreadMemberAdd(threadID string, memberID string, options ...discordgo.RequestOption) error
	GetBotUser() (*discordgo.User, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildScheduledEventCreate(guildID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	GuildScheduledEventEdit(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	ThreadStartComplex(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (ch *discordgo.Channel, err error)
	AddHandler(handler interface{}) func()
	ChannelMessageDelete(channelID string, messageID string, options ...discordgo.RequestOption) (err error)
	User(userID string, options ...discordgo.RequestOption) (st *discordgo.User, err error)
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageEdit(interaction *discordgo.Interaction, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	Open() error
	Close() error
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ApplicationCommandPermissionsEdit(appID string, guildID string, cmdID string, permissions *discordgo.ApplicationCommandPermissionsList, options ...discordgo.RequestOption) (err error)
	ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	WebhookMessageEdit(webhookID string, token string, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (st *discordgo.Message, err error)
	ChannelMessageEditEmbed(channelID string, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessage(channelID string, messageID string, options ...discordgo.RequestOption) (st *discordgo.Message, err error)
	Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error)
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	ChannelEdit(channelID string, data *discordgo.ChannelEdit, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	GuildRoleCreate(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error)
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

func (d *DiscordSession) GetUnderlyingSession() *discordgo.Session {
	return d.session
}

// DiscordState is an implementation of the State interface.  Again, consider
// removing this if you can.
type DiscordState struct {
	state *discordgo.State
}

func (d *DiscordSession) ApplicationCommandPermissionsEdit(appID string, guildID string, cmdID string, permissions *discordgo.ApplicationCommandPermissionsList, options ...discordgo.RequestOption) (err error) {
	return d.session.ApplicationCommandPermissionsEdit(appID, guildID, cmdID, permissions, options...)
}

func (d *DiscordSession) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.FollowupMessageCreate(interaction, wait, data, options...)
}

func (d *DiscordSession) WebhookExecute(webhookID string, token string, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (st *discordgo.Message, err error) {
	return d.session.WebhookExecute("", token, wait, data, options...)
}

// NewDiscordSession creates a new DiscordSession.
func NewDiscordSession(session *discordgo.Session, logger *slog.Logger) *DiscordSession {
	return &DiscordSession{session: session, logger: logger}
}

// NewDiscordState creates a new DiscordState.
func NewDiscordState(state *discordgo.State) *DiscordState {
	return &DiscordState{state: state}
}

func (d *DiscordSession) ChannelMessageEditComplex(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (st *discordgo.Message, err error) {
	return d.session.ChannelMessageEditComplex(m, options...)
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

// MessageReactionAdd handles adding a reaction to a message.
func (d *DiscordSession) MessageReactionAdd(channelID, messageID, emojiID string) error {
	return d.session.MessageReactionAdd(channelID, messageID, emojiID)
}

// GetChannel retrieves a channel by its ID.
func (d *DiscordSession) GetChannel(channelID string, options ...discordgo.RequestOption) (st *discordgo.Channel, err error) {
	return d.session.Channel(channelID, options...)
}

// ChannelMessages fetches messages from a channel.
func (d *DiscordSession) ChannelMessages(channelID string, limit int, beforeID string, afterID string, aroundID string, options ...discordgo.RequestOption) (st []*discordgo.Message, err error) {
	return d.session.ChannelMessages(channelID, limit, beforeID, afterID, aroundID)
}

// CreateThread creates a new thread in a channel.
func (d *DiscordSession) MessageThreadStartComplex(channelID string, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (ch *discordgo.Channel, err error) {
	return d.session.MessageThreadStartComplex(channelID, messageID, data)
}

// AddUserToThread adds a user to a thread.
func (d *DiscordSession) ThreadMemberAdd(threadID string, memberID string, options ...discordgo.RequestOption) error {
	return d.session.ThreadMemberAdd(threadID, memberID)
}

// GetBotUser retrieves the bot user.
func (d *DiscordSession) GetBotUser() (*discordgo.User, error) {
	return d.session.User("@me")
}

func (d *DiscordSession) ChannelMessageDelete(channelID string, messageID string, options ...discordgo.RequestOption) (err error) {
	return d.session.ChannelMessageDelete(channelID, messageID, options...)
}

func (d *DiscordSession) ChannelMessageEditEmbed(channelID string, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.ChannelMessageEditEmbed(channelID, messageID, embed, options...)
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

func (d *DiscordSession) ThreadsActive(channelID string, options ...discordgo.RequestOption) (threads *discordgo.ThreadsList, err error) {
	return d.session.ThreadsActive(channelID, options...)
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

func (d *DiscordSession) User(userID string, options ...discordgo.RequestOption) (st *discordgo.User, err error) {
	return d.session.User(userID, options...)
}

// GuildMemberRoleAdd adds a role to a guild member.
func (d *DiscordSession) GuildMemberRoleAdd(guildID string, userID string, roleID string, options ...discordgo.RequestOption) (err error) {
	return d.session.GuildMemberRoleAdd(guildID, userID, roleID, options...)
}

func (d *DiscordSession) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	return d.session.GuildMember(guildID, userID, options...)
}

func (d *DiscordSession) GuildMemberRoleRemove(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	return d.session.GuildMemberRoleRemove(guildID, userID, roleID, options...)
}

func (d *DiscordSession) FollowupMessageEdit(interaction *discordgo.Interaction, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.session.FollowupMessageEdit(interaction, messageID, data, options...)
}

func (d *DiscordSession) WebhookMessageEdit(webhookID string, token string, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (st *discordgo.Message, err error) {
	return d.session.WebhookMessageEdit(webhookID, token, messageID, data, options...)
}

func (d *DiscordSession) ChannelMessage(channelID string, messageID string, options ...discordgo.RequestOption) (st *discordgo.Message, err error) {
	return d.session.ChannelMessage(channelID, messageID, options...)
}

func (d *DiscordSession) Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
	return d.session.Guild(guildID, options...)
}

func (d *DiscordSession) GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	return d.session.GuildChannels(guildID, options...)
}

func (d *DiscordSession) GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return d.session.GuildChannelCreate(guildID, name, ctype, options...)
}

func (d *DiscordSession) ChannelEdit(channelID string, data *discordgo.ChannelEdit, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return d.session.ChannelEdit(channelID, data, options...)
}

func (d *DiscordSession) GuildRoleCreate(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
	return d.session.GuildRoleCreate(guildID, params, options...)
}
