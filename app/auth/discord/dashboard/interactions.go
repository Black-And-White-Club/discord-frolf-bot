package dashboard

import (
	"context"

	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// DashboardInteractionData stored for correlation with response
type DashboardInteractionData struct {
	InteractionToken string `json:"interaction_token"`
	UserID           string `json:"user_id"`
	GuildID          string `json:"guild_id"`
}

// HandleDashboardCommand handles the /dashboard slash command
func (m *dashboardManager) HandleDashboardCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	ctx, span := m.tracer.Start(ctx, "dashboard.HandleDashboardCommand")
	defer span.End()

	m.logger.InfoContext(ctx, "Processing /dashboard command",
		attr.String("user_id", i.Member.User.ID),
		attr.String("guild_id", i.GuildID),
	)

	// 1. Acknowledge interaction immediately (ephemeral, deferred)
	if err := m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		m.logger.ErrorContext(ctx, "Failed to acknowledge interaction", attr.Error(err))
		return err
	}

	// 2. Get guild config for role mapping
	guildConfig, err := m.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to get guild config", attr.Error(err))
		return m.sendErrorFollowup(i, "Unable to load server configuration. Please try again.")
	}

	// 3. Map Discord roles to PWA role
	pwaRole := m.permMapper.MapMemberRole(i.Member, guildConfig)

	m.logger.InfoContext(ctx, "Mapped PWA role",
		attr.String("user_id", i.Member.User.ID),
		attr.String("pwa_role", pwaRole.String()),
	)

	// 4. Generate correlation ID and store interaction
	correlationID := uuid.New().String()

	interactionData := DashboardInteractionData{
		InteractionToken: i.Token,
		UserID:           i.Member.User.ID,
		GuildID:          i.GuildID,
	}

	if err := m.interactionStore.Set(ctx, correlationID, interactionData); err != nil {
		m.logger.ErrorContext(ctx, "Failed to store interaction", attr.Error(err))
		return m.sendErrorFollowup(i, "Internal error. Please try again.")
	}

	// 5. Publish magic link request event
	payload := authevents.MagicLinkRequestedPayload{
		UserID:        i.Member.User.ID,
		GuildID:       i.GuildID,
		Role:          pwaRole.String(),
		CorrelationID: correlationID,
	}

	msg, err := m.helper.CreateNewMessage(payload, authevents.MagicLinkRequestedV1)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to create magic link request message", attr.Error(err))
		m.interactionStore.Delete(ctx, correlationID)
		return m.sendErrorFollowup(i, "Unable to request dashboard link. Please try again.")
	}

	if err := m.eventBus.Publish(authevents.MagicLinkRequestedV1, msg); err != nil {
		m.logger.ErrorContext(ctx, "Failed to publish magic link request", attr.Error(err))
		m.interactionStore.Delete(ctx, correlationID)
		return m.sendErrorFollowup(i, "Unable to request dashboard link. Please try again.")
	}

	m.logger.InfoContext(ctx, "Published magic link request",
		attr.String("correlation_id", correlationID),
		attr.String("user_id", i.Member.User.ID),
	)

	// Response will be sent by the Watermill handler when backend responds
	return nil
}

func (m *dashboardManager) sendErrorFollowup(i *discordgo.InteractionCreate, message string) error {
	_, err := m.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: message,
	})
	return err
}
