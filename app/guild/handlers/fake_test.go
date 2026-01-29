package handlers

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/reset"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guild/discord/setup"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/bwmarrin/discordgo"
)

// FakeGuildDiscord is a programmable fake for GuildDiscordInterface
type FakeGuildDiscord struct {
	GetSetupManagerFunc       func() setup.SetupManager
	GetResetManagerFunc       func() reset.ResetManager
	RegisterAllCommandsFunc   func(guildID string) error
	UnregisterAllCommandsFunc func(guildID string) error

	// Holds the sub-fakes
	SetupManager FakeSetupManager
	ResetManager FakeResetManager
}

func (f *FakeGuildDiscord) GetSetupManager() setup.SetupManager {
	if f.GetSetupManagerFunc != nil {
		return f.GetSetupManagerFunc()
	}
	return &f.SetupManager
}

func (f *FakeGuildDiscord) GetResetManager() reset.ResetManager {
	if f.GetResetManagerFunc != nil {
		return f.GetResetManagerFunc()
	}
	return &f.ResetManager
}

func (f *FakeGuildDiscord) RegisterAllCommands(guildID string) error {
	if f.RegisterAllCommandsFunc != nil {
		return f.RegisterAllCommandsFunc(guildID)
	}
	return nil
}

func (f *FakeGuildDiscord) UnregisterAllCommands(guildID string) error {
	if f.UnregisterAllCommandsFunc != nil {
		return f.UnregisterAllCommandsFunc(guildID)
	}
	return nil
}

// FakeSetupManager implements setup.SetupManager
type FakeSetupManager struct {
	HandleSetupCommandFunc     func(ctx context.Context, i *discordgo.InteractionCreate) error
	SendSetupModalFunc         func(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleSetupModalSubmitFunc func(ctx context.Context, i *discordgo.InteractionCreate) error
}

func (f *FakeSetupManager) HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.HandleSetupCommandFunc != nil {
		return f.HandleSetupCommandFunc(ctx, i)
	}
	return nil
}

func (f *FakeSetupManager) SendSetupModal(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.SendSetupModalFunc != nil {
		return f.SendSetupModalFunc(ctx, i)
	}
	return nil
}

func (f *FakeSetupManager) HandleSetupModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.HandleSetupModalSubmitFunc != nil {
		return f.HandleSetupModalSubmitFunc(ctx, i)
	}
	return nil
}

// FakeResetManager implements reset.ResetManager
type FakeResetManager struct {
	HandleResetCommandFunc       func(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetConfirmButtonFunc func(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleResetCancelButtonFunc  func(ctx context.Context, i *discordgo.InteractionCreate) error
	DeleteResourcesFunc          func(ctx context.Context, guildID string, state guildtypes.ResourceState) (map[string]guildtypes.DeletionResult, error)
}

func (f *FakeResetManager) HandleResetCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.HandleResetCommandFunc != nil {
		return f.HandleResetCommandFunc(ctx, i)
	}
	return nil
}

func (f *FakeResetManager) HandleResetConfirmButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.HandleResetConfirmButtonFunc != nil {
		return f.HandleResetConfirmButtonFunc(ctx, i)
	}
	return nil
}

func (f *FakeResetManager) HandleResetCancelButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	if f.HandleResetCancelButtonFunc != nil {
		return f.HandleResetCancelButtonFunc(ctx, i)
	}
	return nil
}

func (f *FakeResetManager) DeleteResources(ctx context.Context, guildID string, state guildtypes.ResourceState) (map[string]guildtypes.DeletionResult, error) {
	if f.DeleteResourcesFunc != nil {
		return f.DeleteResourcesFunc(ctx, guildID, state)
	}
	return map[string]guildtypes.DeletionResult{}, nil
}

// Ensure interface compliance
var _ discord.GuildDiscordInterface = (*FakeGuildDiscord)(nil)
var _ setup.SetupManager = (*FakeSetupManager)(nil)
var _ reset.ResetManager = (*FakeResetManager)(nil)
