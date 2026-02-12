package season

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (sm *seasonManager) HandleSeasonCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "season")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "application_command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		sm.logger.WarnContext(ctx, "No options provided for season command")
		return
	}

	subCommand := options[0].Name
	switch subCommand {
	case "start":
		sm.handleStartSeason(ctx, i, options[0].Options)
	case "standings":
		sm.handleGetStandings(ctx, i, options[0].Options)
	default:
		sm.logger.WarnContext(ctx, "Unknown subcommand", attr.String("subcommand", subCommand))
		sm.respondWithError(ctx, i, "Unknown subcommand")
	}
}

func (sm *seasonManager) handleStartSeason(ctx context.Context, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var name string
	for _, opt := range options {
		if opt.Name == "name" {
			name = opt.StringValue()
		}
	}

	if name == "" {
		sm.respondWithError(ctx, i, "Season name is required")
		return
	}

	seasonID := uuid.New().String()
	guildID := i.GuildID

	sm.logger.InfoContext(ctx, "Handling start season request",
		attr.String("season_name", name),
		attr.String("season_id", seasonID),
		attr.String("guild_id", guildID))

	// Defer the response to prevent timeout
	err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to defer interaction", attr.Error(err))
		return
	}

	payload := &leaderboardevents.StartNewSeasonPayloadV1{
		GuildID:    sharedtypes.GuildID(guildID),
		SeasonID:   seasonID,
		SeasonName: name,
		// ChannelID:  i.ChannelID, // Checking if this field exists
	}

	msg, err := sm.helper.CreateNewMessage(payload, leaderboardevents.LeaderboardStartNewSeasonV1)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to create message", attr.Error(err))
		sm.followupWithError(ctx, i, "Failed to process request")
		return
	}
	msg.Metadata.Set("guild_id", guildID)
	msg.Metadata.Set("channel_id", i.ChannelID)

	if err := sm.publisher.Publish(leaderboardevents.LeaderboardStartNewSeasonV1, msg); err != nil {
		sm.logger.ErrorContext(ctx, "Failed to publish event", attr.Error(err))
		sm.followupWithError(ctx, i, "Failed to start season")
		return
	}

	// Update the deferred response
	_, err = sm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &[]string{fmt.Sprintf("Starting new season **%s**...", name)}[0],
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to edit interaction response", attr.Error(err))
	}
}

func (sm *seasonManager) handleGetStandings(ctx context.Context, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var seasonID string
	for _, opt := range options {
		if opt.Name == "season_id" {
			val := opt.StringValue()
			if val != "" {
				if _, err := uuid.Parse(val); err != nil {
					sm.respondWithError(ctx, i, "Invalid season_id provided. Must be a valid UUID.")
					return
				}
				seasonID = val
			}
		}
	}

	guildID := i.GuildID

	sm.logger.InfoContext(ctx, "Handling get season standings request",
		attr.String("season_id", seasonID),
		attr.String("guild_id", guildID))

	// Defer the response
	err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to defer interaction", attr.Error(err))
		return
	}

	payload := &leaderboardevents.GetSeasonStandingsPayloadV1{
		GuildID:  sharedtypes.GuildID(guildID),
		SeasonID: seasonID,
	}

	msg, err := sm.helper.CreateNewMessage(payload, leaderboardevents.LeaderboardGetSeasonStandingsV1)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to create message", attr.Error(err))
		sm.followupWithError(ctx, i, "Failed to process request")
		return
	}
	msg.Metadata.Set("guild_id", guildID)
	msg.Metadata.Set("channel_id", i.ChannelID)

	if err := sm.publisher.Publish(leaderboardevents.LeaderboardGetSeasonStandingsV1, msg); err != nil {
		sm.logger.ErrorContext(ctx, "Failed to publish event", attr.Error(err))
		sm.followupWithError(ctx, i, "Failed to fetch standings")
		return
	}

	respContent := "Fetching season standings..."
	if seasonID != "" {
		respContent = fmt.Sprintf("Fetching standings for season ID: %s...", seasonID)
	}

	_, err = sm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &respContent,
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to edit interaction response", attr.Error(err))
	}
}

func (sm *seasonManager) respondWithError(ctx context.Context, i *discordgo.InteractionCreate, message string) {
	err := sm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Error: %s", message),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to respond with error", attr.Error(err))
	}
}

func (sm *seasonManager) followupWithError(ctx context.Context, i *discordgo.InteractionCreate, message string) {
	content := fmt.Sprintf("Error: %s", message)
	_, err := sm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		sm.logger.ErrorContext(ctx, "Failed to followup with error", attr.Error(err))
	}
}
