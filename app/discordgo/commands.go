package discord

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

const guildCommandManifestVersion = "2026-02-18.1"

// GuildCommandManifestVersion returns the current guild command manifest version.
func GuildCommandManifestVersion() string {
	return guildCommandManifestVersion
}

// RegisterCommands reconciles slash commands with Discord state using upsert + prune semantics.
func RegisterCommands(s Session, logger *slog.Logger, guildID string) error {
	appID, err := s.GetBotUser()
	if err != nil {
		return fmt.Errorf("failed to retrieve bot user: %w", err)
	}

	targetGuildID := guildID
	if targetGuildID == "" {
		logger.Info("Reconciling global commands for multi-tenant deployment")
	} else {
		logger.Info("Reconciling commands for specific guild", attr.String("guild_id", targetGuildID))
	}

	desired := desiredCommands(targetGuildID)
	desiredByName := make(map[string]*discordgo.ApplicationCommand, len(desired))
	for _, cmd := range desired {
		if cmd == nil || cmd.Name == "" {
			return fmt.Errorf("desired command has empty name")
		}
		desiredByName[cmd.Name] = cmd
	}

	var existing []*discordgo.ApplicationCommand
	if err := RetryDiscordAPI(logger, "list application commands", func() error {
		var listErr error
		existing, listErr = s.ApplicationCommands(appID.ID, targetGuildID)
		return listErr
	}); err != nil {
		return fmt.Errorf("failed to list application commands: %w", err)
	}

	existingByName := make(map[string]*discordgo.ApplicationCommand, len(existing))
	for _, cmd := range existing {
		if cmd == nil || cmd.Name == "" {
			continue
		}
		existingByName[cmd.Name] = cmd
	}

	for name, desiredCmd := range desiredByName {
		current, exists := existingByName[name]
		if !exists {
			if err := RetryDiscordAPI(logger, "create application command "+name, func() error {
				_, createErr := s.ApplicationCommandCreate(appID.ID, targetGuildID, desiredCmd)
				return createErr
			}); err != nil {
				return fmt.Errorf("failed to create command %q: %w", name, err)
			}

			logger.Info("Registered command",
				attr.String("command_name", name),
				attr.String("guild_id", targetGuildID))
			continue
		}

		if commandShapeEqual(current, desiredCmd) {
			logger.Debug("Command already up to date",
				attr.String("command_name", name),
				attr.String("guild_id", targetGuildID))
			continue
		}

		if current.ID == "" {
			return fmt.Errorf("existing command %q has empty ID; cannot update", name)
		}

		if err := RetryDiscordAPI(logger, "edit application command "+name, func() error {
			_, editErr := s.ApplicationCommandEdit(appID.ID, targetGuildID, current.ID, desiredCmd)
			return editErr
		}); err != nil {
			return fmt.Errorf("failed to update command %q: %w", name, err)
		}

		logger.Info("Updated command",
			attr.String("command_name", name),
			attr.String("guild_id", targetGuildID))
	}

	for name, current := range existingByName {
		if _, ok := desiredByName[name]; ok {
			continue
		}
		if current == nil || current.ID == "" {
			continue
		}

		if err := RetryDiscordAPI(logger, "delete application command "+name, func() error {
			return s.ApplicationCommandDelete(appID.ID, targetGuildID, current.ID)
		}); err != nil {
			return fmt.Errorf("failed to delete deprecated command %q: %w", name, err)
		}

		logger.Info("Deleted deprecated command",
			attr.String("command_name", name),
			attr.String("guild_id", targetGuildID))
	}

	return nil
}

func desiredCommands(targetGuildID string) []*discordgo.ApplicationCommand {
	// For multi-tenant mode (empty guildID), only setup/reset are global.
	if targetGuildID == "" {
		return []*discordgo.ApplicationCommand{
			{
				Name:                     "frolf-setup",
				Description:              "Set up Frolf Bot for this server (Admin only)",
				DefaultMemberPermissions: int64Ptr(discordgo.PermissionAdministrator),
			},
			{
				Name:                     "frolf-reset",
				Description:              "Reset Frolf Bot configuration for this server (Admin only)",
				DefaultMemberPermissions: int64Ptr(discordgo.PermissionAdministrator),
			},
		}
	}

	return []*discordgo.ApplicationCommand{
		{
			Name:        "updaterole",
			Description: "Request a role for a user (Requires Editor role or higher)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to request a role for",
					Required:    true,
				},
			},
		},
		{
			Name:        "createround",
			Description: "Create a new frolf round (Available to all players)",
		},
		{
			Name:        "claimtag",
			Description: "Claim a specific tag number on the leaderboard (Available to all players)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "tag",
					Description: "Tag number to claim (1-100)",
					Required:    true,
					MinValue:    float64Ptr(1),
					MaxValue:    100,
				},
			},
		},
		{
			Name:        "set-udisc-name",
			Description: "Set your UDisc username and name for scorecard matching (Available to all players)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "username",
					Description: "Your UDisc username (e.g., @johndoe)",
					Required:    false,
					MaxLength:   100,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Your name as shown on UDisc rounds",
					Required:    false,
					MaxLength:   100,
				},
			},
		},
		{
			Name:        "dashboard",
			Description: "Get a link to access the Frolf PWA dashboard",
		},
		{
			Name:        "invite",
			Description: "Get a link to manage club invites (Editor/Admin only)",
		},
		{
			Name:        "season",
			Description: "Manage and view seasons (Admin only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "start",
					Description: "Start a new season",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "Name of the new season",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "standings",
					Description: "View season standings",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "season_id",
							Description: "ID of the season (optional, defaults to current)",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "end",
					Description: "End the current season",
				},
			},
			DefaultMemberPermissions: int64Ptr(discordgo.PermissionAdministrator),
		},
	}
}

func commandShapeEqual(a, b *discordgo.ApplicationCommand) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Name == b.Name &&
		a.Description == b.Description &&
		int64PtrEqual(a.DefaultMemberPermissions, b.DefaultMemberPermissions) &&
		reflect.DeepEqual(a.Options, b.Options)
}

func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
