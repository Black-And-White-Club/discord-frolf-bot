package handlers

import (
	"context"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	userdiscord "github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/discord/udisc"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// FakeUserDiscord is a programmable fake for UserDiscordInterface
type FakeUserDiscord struct {
	GetRoleManagerFunc   func() role.RoleManager
	GetSignupManagerFunc func() signup.SignupManager
	GetUDiscManagerFunc  func() udisc.UDiscManager
	SyncGuildMemberFunc  func(ctx context.Context, guildID, userID string) error

	// Holds the sub-fakes
	RoleManager   FakeRoleManager
	SignupManager FakeSignupManager
	UDiscManager  FakeUDiscManager
}

func (f *FakeUserDiscord) GetRoleManager() role.RoleManager {
	if f.GetRoleManagerFunc != nil {
		return f.GetRoleManagerFunc()
	}
	return &f.RoleManager
}

func (f *FakeUserDiscord) GetSignupManager() signup.SignupManager {
	if f.GetSignupManagerFunc != nil {
		return f.GetSignupManagerFunc()
	}
	return &f.SignupManager
}

func (f *FakeUserDiscord) GetUDiscManager() udisc.UDiscManager {
	if f.GetUDiscManagerFunc != nil {
		return f.GetUDiscManagerFunc()
	}
	return &f.UDiscManager
}

func (f *FakeUserDiscord) SyncGuildMember(ctx context.Context, guildID, userID string) error {
	if f.SyncGuildMemberFunc != nil {
		return f.SyncGuildMemberFunc(ctx, guildID, userID)
	}
	return nil
}

// FakeRoleManager implements role.RoleManager
type FakeRoleManager struct {
	AddRoleToUserFunc            func(ctx context.Context, guildID string, userID sharedtypes.DiscordID, roleID string) (role.RoleOperationResult, error)
	EditRoleUpdateResponseFunc   func(ctx context.Context, correlationID string, content string) (role.RoleOperationResult, error)
	HandleRoleRequestCommandFunc func(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error)
	HandleRoleButtonPressFunc    func(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error)
	HandleRoleCancelButtonFunc   func(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error)
	RespondToRoleRequestFunc     func(ctx context.Context, interactionID, interactionToken string, targetUserID sharedtypes.DiscordID) (role.RoleOperationResult, error)
	RespondToRoleButtonPressFunc func(ctx context.Context, interactionID, interactionToken string, requesterID sharedtypes.DiscordID, selectedRole string, targetUserID sharedtypes.DiscordID) (role.RoleOperationResult, error)
}

func (f *FakeRoleManager) AddRoleToUser(ctx context.Context, guildID string, userID sharedtypes.DiscordID, roleID string) (role.RoleOperationResult, error) {
	if f.AddRoleToUserFunc != nil {
		return f.AddRoleToUserFunc(ctx, guildID, userID, roleID)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) EditRoleUpdateResponse(ctx context.Context, correlationID string, content string) (role.RoleOperationResult, error) {
	if f.EditRoleUpdateResponseFunc != nil {
		return f.EditRoleUpdateResponseFunc(ctx, correlationID, content)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) HandleRoleRequestCommand(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error) {
	if f.HandleRoleRequestCommandFunc != nil {
		return f.HandleRoleRequestCommandFunc(ctx, i)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) HandleRoleButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error) {
	if f.HandleRoleButtonPressFunc != nil {
		return f.HandleRoleButtonPressFunc(ctx, i)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) HandleRoleCancelButton(ctx context.Context, i *discordgo.InteractionCreate) (role.RoleOperationResult, error) {
	if f.HandleRoleCancelButtonFunc != nil {
		return f.HandleRoleCancelButtonFunc(ctx, i)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) RespondToRoleRequest(ctx context.Context, interactionID, interactionToken string, targetUserID sharedtypes.DiscordID) (role.RoleOperationResult, error) {
	if f.RespondToRoleRequestFunc != nil {
		return f.RespondToRoleRequestFunc(ctx, interactionID, interactionToken, targetUserID)
	}
	return role.RoleOperationResult{}, nil
}

func (f *FakeRoleManager) RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken string, requesterID sharedtypes.DiscordID, selectedRole string, targetUserID sharedtypes.DiscordID) (role.RoleOperationResult, error) {
	if f.RespondToRoleButtonPressFunc != nil {
		return f.RespondToRoleButtonPressFunc(ctx, interactionID, interactionToken, requesterID, selectedRole, targetUserID)
	}
	return role.RoleOperationResult{}, nil
}

// FakeSignupManager implements signup.SignupManager
type FakeSignupManager struct {
	SendSignupModalFunc          func(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error)
	HandleSignupModalSubmitFunc  func(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error)
	MessageReactionAddFunc       func(s discord.Session, r *discordgo.MessageReactionAdd) (signup.SignupOperationResult, error)
	HandleSignupReactionAddFunc  func(ctx context.Context, r *discordgo.MessageReactionAdd) (signup.SignupOperationResult, error)
	HandleSignupButtonPressFunc  func(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error)
	SendSignupResultFunc         func(ctx context.Context, interactionToken string, success bool, failureReason ...string) (signup.SignupOperationResult, error)
	TrackChannelForReactionsFunc func(channelID string)
	SyncMemberFunc               func(ctx context.Context, guildID, userID string) error
}

func (f *FakeSignupManager) SendSignupModal(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error) {
	if f.SendSignupModalFunc != nil {
		return f.SendSignupModalFunc(ctx, i)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) HandleSignupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error) {
	if f.HandleSignupModalSubmitFunc != nil {
		return f.HandleSignupModalSubmitFunc(ctx, i)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) MessageReactionAdd(s discord.Session, r *discordgo.MessageReactionAdd) (signup.SignupOperationResult, error) {
	if f.MessageReactionAddFunc != nil {
		return f.MessageReactionAddFunc(s, r)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) HandleSignupReactionAdd(ctx context.Context, r *discordgo.MessageReactionAdd) (signup.SignupOperationResult, error) {
	if f.HandleSignupReactionAddFunc != nil {
		return f.HandleSignupReactionAddFunc(ctx, r)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) HandleSignupButtonPress(ctx context.Context, i *discordgo.InteractionCreate) (signup.SignupOperationResult, error) {
	if f.HandleSignupButtonPressFunc != nil {
		return f.HandleSignupButtonPressFunc(ctx, i)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) SendSignupResult(ctx context.Context, interactionToken string, success bool, failureReason ...string) (signup.SignupOperationResult, error) {
	if f.SendSignupResultFunc != nil {
		return f.SendSignupResultFunc(ctx, interactionToken, success, failureReason...)
	}
	return signup.SignupOperationResult{}, nil
}

func (f *FakeSignupManager) TrackChannelForReactions(channelID string) {
	if f.TrackChannelForReactionsFunc != nil {
		f.TrackChannelForReactionsFunc(channelID)
	}
}

func (f *FakeSignupManager) SyncMember(ctx context.Context, guildID, userID string) error {
	if f.SyncMemberFunc != nil {
		return f.SyncMemberFunc(ctx, guildID, userID)
	}
	return nil
}

// FakeUDiscManager implements udisc.UDiscManager
type FakeUDiscManager struct {
	HandleSetUDiscNameCommandFunc func(ctx context.Context, i *discordgo.InteractionCreate) (udisc.UDiscOperationResult, error)
}

func (f *FakeUDiscManager) HandleSetUDiscNameCommand(ctx context.Context, i *discordgo.InteractionCreate) (udisc.UDiscOperationResult, error) {
	if f.HandleSetUDiscNameCommandFunc != nil {
		return f.HandleSetUDiscNameCommandFunc(ctx, i)
	}
	return udisc.UDiscOperationResult{}, nil
}

// Ensure interface compliance
var _ userdiscord.UserDiscordInterface = (*FakeUserDiscord)(nil)
var _ role.RoleManager = (*FakeRoleManager)(nil)
var _ signup.SignupManager = (*FakeSignupManager)(nil)
var _ udisc.UDiscManager = (*FakeUDiscManager)(nil)
