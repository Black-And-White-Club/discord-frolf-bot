package discord

import (
	"github.com/bwmarrin/discordgo"
)

// FakeSession provides a programmable stub for the Session interface.
// It follows the Fake/Stub pattern for testing, where each interface method
// has a corresponding Func field that can be set per-test.
type FakeSession struct {
	trace []string

	// --- Interaction Methods ---
	InteractionRespondFunc      func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponseEditFunc func(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreateFunc   func(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageEditFunc     func(interaction *discordgo.Interaction, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)

	// --- Message Methods ---
	ChannelMessageSendFunc        func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendComplexFunc func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditComplexFunc func(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditEmbedFunc   func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageDeleteFunc      func(channelID, messageID string, options ...discordgo.RequestOption) error
	ChannelMessageFunc            func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessagesFunc           func(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error)

	// --- User/Member Methods ---
	UserFunc                  func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error)
	GetBotUserFunc            func() (*discordgo.User, error)
	UserChannelCreateFunc     func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	GuildMemberFunc           func(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
	GuildMemberRoleAddFunc    func(guildID, userID, roleID string, options ...discordgo.RequestOption) error
	GuildMemberRoleRemoveFunc func(guildID, userID, roleID string, options ...discordgo.RequestOption) error

	// --- Channel Methods ---
	GetChannelFunc    func(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	ChannelDeleteFunc func(channelID string, options ...discordgo.RequestOption) error
	ChannelEditFunc   func(channelID string, data *discordgo.ChannelEdit, options ...discordgo.RequestOption) (*discordgo.Channel, error)

	// --- Guild Methods ---
	GuildFunc              func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error)
	GuildChannelsFunc      func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildChannelCreateFunc func(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	GuildRoleCreateFunc    func(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error)
	GuildRoleDeleteFunc    func(guildID, roleID string, options ...discordgo.RequestOption) error

	// --- Thread Methods ---
	ThreadsActiveFunc             func(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error)
	MessageThreadStartComplexFunc func(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	ThreadStartComplexFunc        func(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	ThreadMemberAddFunc           func(threadID, memberID string, options ...discordgo.RequestOption) error

	// --- Reaction Methods ---
	MessageReactionAddFunc func(channelID, messageID, emojiID string) error

	// --- Webhook Methods ---
	WebhookExecuteFunc     func(webhookID, token string, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	WebhookMessageEditFunc func(webhookID, token, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)

	// --- Scheduled Event Methods ---
	GuildScheduledEventsFunc      func(guildID string, userCount bool, options ...discordgo.RequestOption) ([]*discordgo.GuildScheduledEvent, error)
	GuildScheduledEventCreateFunc func(guildID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	GuildScheduledEventEditFunc   func(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error)
	GuildScheduledEventDeleteFunc func(guildID, eventID string, options ...discordgo.RequestOption) error

	// --- Application Command Methods ---
	ApplicationCommandCreateFunc          func(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommandsFunc               func(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
	ApplicationCommandDeleteFunc          func(appID, guildID, cmdID string, options ...discordgo.RequestOption) error
	ApplicationCommandPermissionsEditFunc func(appID, guildID, cmdID string, permissions *discordgo.ApplicationCommandPermissionsList, options ...discordgo.RequestOption) error

	// --- Handler/Lifecycle Methods ---
	AddHandlerFunc func(handler interface{}) func()
	OpenFunc       func() error
	CloseFunc      func() error
}

// NewFakeSession initializes a new FakeSession with an empty trace.
func NewFakeSession() *FakeSession {
	return &FakeSession{
		trace: []string{},
	}
}

func (f *FakeSession) record(step string) {
	f.trace = append(f.trace, step)
}

// Trace returns the sequence of method calls made to the fake.
func (f *FakeSession) Trace() []string {
	out := make([]string, len(f.trace))
	copy(out, f.trace)
	return out
}

// --- Interaction Methods Implementation ---

func (f *FakeSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	f.record("InteractionRespond")
	if f.InteractionRespondFunc != nil {
		return f.InteractionRespondFunc(interaction, resp, options...)
	}
	return nil
}

func (f *FakeSession) InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("InteractionResponseEdit")
	if f.InteractionResponseEditFunc != nil {
		return f.InteractionResponseEditFunc(interaction, newresp, options...)
	}
	return &discordgo.Message{ID: "fake-msg-123"}, nil
}

func (f *FakeSession) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("FollowupMessageCreate")
	if f.FollowupMessageCreateFunc != nil {
		return f.FollowupMessageCreateFunc(interaction, wait, data, options...)
	}
	return &discordgo.Message{ID: "fake-followup-123"}, nil
}

func (f *FakeSession) FollowupMessageEdit(interaction *discordgo.Interaction, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("FollowupMessageEdit")
	if f.FollowupMessageEditFunc != nil {
		return f.FollowupMessageEditFunc(interaction, messageID, data, options...)
	}
	return &discordgo.Message{ID: messageID}, nil
}

// --- Message Methods Implementation ---

func (f *FakeSession) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("ChannelMessageSend")
	if f.ChannelMessageSendFunc != nil {
		return f.ChannelMessageSendFunc(channelID, content, options...)
	}
	return &discordgo.Message{ID: "fake-msg-123", ChannelID: channelID, Content: content}, nil
}

func (f *FakeSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("ChannelMessageSendComplex")
	if f.ChannelMessageSendComplexFunc != nil {
		return f.ChannelMessageSendComplexFunc(channelID, data, options...)
	}
	return &discordgo.Message{ID: "fake-msg-123", ChannelID: channelID}, nil
}

func (f *FakeSession) ChannelMessageEditComplex(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("ChannelMessageEditComplex")
	if f.ChannelMessageEditComplexFunc != nil {
		return f.ChannelMessageEditComplexFunc(m, options...)
	}
	return &discordgo.Message{ID: m.ID, ChannelID: m.Channel}, nil
}

func (f *FakeSession) ChannelMessageEditEmbed(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("ChannelMessageEditEmbed")
	if f.ChannelMessageEditEmbedFunc != nil {
		return f.ChannelMessageEditEmbedFunc(channelID, messageID, embed, options...)
	}
	return &discordgo.Message{ID: messageID, ChannelID: channelID}, nil
}

func (f *FakeSession) ChannelMessageDelete(channelID, messageID string, options ...discordgo.RequestOption) error {
	f.record("ChannelMessageDelete")
	if f.ChannelMessageDeleteFunc != nil {
		return f.ChannelMessageDeleteFunc(channelID, messageID, options...)
	}
	return nil
}

func (f *FakeSession) ChannelMessage(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("ChannelMessage")
	if f.ChannelMessageFunc != nil {
		return f.ChannelMessageFunc(channelID, messageID, options...)
	}
	return &discordgo.Message{ID: messageID, ChannelID: channelID}, nil
}

func (f *FakeSession) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error) {
	f.record("ChannelMessages")
	if f.ChannelMessagesFunc != nil {
		return f.ChannelMessagesFunc(channelID, limit, beforeID, afterID, aroundID, options...)
	}
	return []*discordgo.Message{}, nil
}

// --- User/Member Methods Implementation ---

func (f *FakeSession) User(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
	f.record("User")
	if f.UserFunc != nil {
		return f.UserFunc(userID, options...)
	}
	return &discordgo.User{ID: userID, Username: "fake-user"}, nil
}

func (f *FakeSession) GetBotUser() (*discordgo.User, error) {
	f.record("GetBotUser")
	if f.GetBotUserFunc != nil {
		return f.GetBotUserFunc()
	}
	return &discordgo.User{ID: "fake-bot-id", Username: "fake-bot"}, nil
}

func (f *FakeSession) UserChannelCreate(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("UserChannelCreate")
	if f.UserChannelCreateFunc != nil {
		return f.UserChannelCreateFunc(recipientID, options...)
	}
	return &discordgo.Channel{ID: "fake-dm-channel-123"}, nil
}

func (f *FakeSession) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	f.record("GuildMember")
	if f.GuildMemberFunc != nil {
		return f.GuildMemberFunc(guildID, userID, options...)
	}
	return &discordgo.Member{
		User:  &discordgo.User{ID: userID, Username: "fake-user"},
		Roles: []string{},
	}, nil
}

func (f *FakeSession) GuildMemberRoleAdd(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	f.record("GuildMemberRoleAdd")
	if f.GuildMemberRoleAddFunc != nil {
		return f.GuildMemberRoleAddFunc(guildID, userID, roleID, options...)
	}
	return nil
}

func (f *FakeSession) GuildMemberRoleRemove(guildID, userID, roleID string, options ...discordgo.RequestOption) error {
	f.record("GuildMemberRoleRemove")
	if f.GuildMemberRoleRemoveFunc != nil {
		return f.GuildMemberRoleRemoveFunc(guildID, userID, roleID, options...)
	}
	return nil
}

// --- Channel Methods Implementation ---

func (f *FakeSession) GetChannel(channelID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("GetChannel")
	if f.GetChannelFunc != nil {
		return f.GetChannelFunc(channelID, options...)
	}
	return &discordgo.Channel{ID: channelID}, nil
}

func (f *FakeSession) ChannelDelete(channelID string, options ...discordgo.RequestOption) error {
	f.record("ChannelDelete")
	if f.ChannelDeleteFunc != nil {
		return f.ChannelDeleteFunc(channelID, options...)
	}
	return nil
}

func (f *FakeSession) ChannelEdit(channelID string, data *discordgo.ChannelEdit, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("ChannelEdit")
	if f.ChannelEditFunc != nil {
		return f.ChannelEditFunc(channelID, data, options...)
	}
	return &discordgo.Channel{ID: channelID}, nil
}

// --- Guild Methods Implementation ---

func (f *FakeSession) Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
	f.record("Guild")
	if f.GuildFunc != nil {
		return f.GuildFunc(guildID, options...)
	}
	return &discordgo.Guild{ID: guildID, Name: "Fake Guild"}, nil
}

func (f *FakeSession) GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	f.record("GuildChannels")
	if f.GuildChannelsFunc != nil {
		return f.GuildChannelsFunc(guildID, options...)
	}
	return []*discordgo.Channel{}, nil
}

func (f *FakeSession) GuildChannelCreate(guildID, name string, ctype discordgo.ChannelType, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("GuildChannelCreate")
	if f.GuildChannelCreateFunc != nil {
		return f.GuildChannelCreateFunc(guildID, name, ctype, options...)
	}
	return &discordgo.Channel{ID: "fake-channel-123", Name: name, GuildID: guildID}, nil
}

func (f *FakeSession) GuildRoleCreate(guildID string, params *discordgo.RoleParams, options ...discordgo.RequestOption) (*discordgo.Role, error) {
	f.record("GuildRoleCreate")
	if f.GuildRoleCreateFunc != nil {
		return f.GuildRoleCreateFunc(guildID, params, options...)
	}
	return &discordgo.Role{ID: "fake-role-123"}, nil
}

func (f *FakeSession) GuildRoleDelete(guildID, roleID string, options ...discordgo.RequestOption) error {
	f.record("GuildRoleDelete")
	if f.GuildRoleDeleteFunc != nil {
		return f.GuildRoleDeleteFunc(guildID, roleID, options...)
	}
	return nil
}

// --- Thread Methods Implementation ---

func (f *FakeSession) ThreadsActive(channelID string, options ...discordgo.RequestOption) (*discordgo.ThreadsList, error) {
	f.record("ThreadsActive")
	if f.ThreadsActiveFunc != nil {
		return f.ThreadsActiveFunc(channelID, options...)
	}
	return &discordgo.ThreadsList{Threads: []*discordgo.Channel{}}, nil
}

func (f *FakeSession) MessageThreadStartComplex(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("MessageThreadStartComplex")
	if f.MessageThreadStartComplexFunc != nil {
		return f.MessageThreadStartComplexFunc(channelID, messageID, data, options...)
	}
	return &discordgo.Channel{ID: "fake-thread-123"}, nil
}

func (f *FakeSession) ThreadStartComplex(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	f.record("ThreadStartComplex")
	if f.ThreadStartComplexFunc != nil {
		return f.ThreadStartComplexFunc(channelID, data, options...)
	}
	return &discordgo.Channel{ID: "fake-thread-123"}, nil
}

func (f *FakeSession) ThreadMemberAdd(threadID, memberID string, options ...discordgo.RequestOption) error {
	f.record("ThreadMemberAdd")
	if f.ThreadMemberAddFunc != nil {
		return f.ThreadMemberAddFunc(threadID, memberID, options...)
	}
	return nil
}

// --- Reaction Methods Implementation ---

func (f *FakeSession) MessageReactionAdd(channelID, messageID, emojiID string) error {
	f.record("MessageReactionAdd")
	if f.MessageReactionAddFunc != nil {
		return f.MessageReactionAddFunc(channelID, messageID, emojiID)
	}
	return nil
}

// --- Webhook Methods Implementation ---

func (f *FakeSession) WebhookExecute(webhookID, token string, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("WebhookExecute")
	if f.WebhookExecuteFunc != nil {
		return f.WebhookExecuteFunc(webhookID, token, wait, data, options...)
	}
	return &discordgo.Message{ID: "fake-webhook-msg-123"}, nil
}

func (f *FakeSession) WebhookMessageEdit(webhookID, token, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.record("WebhookMessageEdit")
	if f.WebhookMessageEditFunc != nil {
		return f.WebhookMessageEditFunc(webhookID, token, messageID, data, options...)
	}
	return &discordgo.Message{ID: messageID}, nil
}

// --- Scheduled Event Methods Implementation ---

func (f *FakeSession) GuildScheduledEvents(guildID string, userCount bool, options ...discordgo.RequestOption) ([]*discordgo.GuildScheduledEvent, error) {
	f.record("GuildScheduledEvents")
	if f.GuildScheduledEventsFunc != nil {
		return f.GuildScheduledEventsFunc(guildID, userCount, options...)
	}
	return []*discordgo.GuildScheduledEvent{}, nil
}

func (f *FakeSession) GuildScheduledEventCreate(guildID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	f.record("GuildScheduledEventCreate")
	if f.GuildScheduledEventCreateFunc != nil {
		return f.GuildScheduledEventCreateFunc(guildID, params, options...)
	}
	return &discordgo.GuildScheduledEvent{ID: "fake-event-123"}, nil
}

func (f *FakeSession) GuildScheduledEventEdit(guildID, eventID string, params *discordgo.GuildScheduledEventParams, options ...discordgo.RequestOption) (*discordgo.GuildScheduledEvent, error) {
	f.record("GuildScheduledEventEdit")
	if f.GuildScheduledEventEditFunc != nil {
		return f.GuildScheduledEventEditFunc(guildID, eventID, params, options...)
	}
	return &discordgo.GuildScheduledEvent{ID: eventID}, nil
}

func (f *FakeSession) GuildScheduledEventDelete(guildID, eventID string, options ...discordgo.RequestOption) error {
	f.record("GuildScheduledEventDelete")
	if f.GuildScheduledEventDeleteFunc != nil {
		return f.GuildScheduledEventDeleteFunc(guildID, eventID, options...)
	}
	return nil
}

// --- Application Command Methods Implementation ---

func (f *FakeSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	f.record("ApplicationCommandCreate")
	if f.ApplicationCommandCreateFunc != nil {
		return f.ApplicationCommandCreateFunc(appID, guildID, cmd, options...)
	}
	return &discordgo.ApplicationCommand{ID: "fake-cmd-123", Name: cmd.Name}, nil
}

func (f *FakeSession) ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	f.record("ApplicationCommands")
	if f.ApplicationCommandsFunc != nil {
		return f.ApplicationCommandsFunc(appID, guildID, options...)
	}
	return []*discordgo.ApplicationCommand{}, nil
}

func (f *FakeSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	f.record("ApplicationCommandDelete")
	if f.ApplicationCommandDeleteFunc != nil {
		return f.ApplicationCommandDeleteFunc(appID, guildID, cmdID, options...)
	}
	return nil
}

func (f *FakeSession) ApplicationCommandPermissionsEdit(appID, guildID, cmdID string, permissions *discordgo.ApplicationCommandPermissionsList, options ...discordgo.RequestOption) error {
	f.record("ApplicationCommandPermissionsEdit")
	if f.ApplicationCommandPermissionsEditFunc != nil {
		return f.ApplicationCommandPermissionsEditFunc(appID, guildID, cmdID, permissions, options...)
	}
	return nil
}

// --- Handler/Lifecycle Methods Implementation ---

func (f *FakeSession) AddHandler(handler interface{}) func() {
	f.record("AddHandler")
	if f.AddHandlerFunc != nil {
		return f.AddHandlerFunc(handler)
	}
	return func() {} // No-op unsubscribe function
}

func (f *FakeSession) Open() error {
	f.record("Open")
	if f.OpenFunc != nil {
		return f.OpenFunc()
	}
	return nil
}

func (f *FakeSession) Close() error {
	f.record("Close")
	if f.CloseFunc != nil {
		return f.CloseFunc()
	}
	return nil
}

// Interface assertion - compile-time check that FakeSession implements Session
var _ Session = (*FakeSession)(nil)
