package handlers

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
)

// HandleMagicLinkGenerated handles the magic link generated event from the backend.
func (h *AuthHandlers) HandleMagicLinkGenerated(
	ctx context.Context,
	payload *authevents.MagicLinkGeneratedPayload,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		h.logger.ErrorContext(ctx, "Received nil payload for magic link generated event")
		return nil, nil // Don't retry - this is a programming error
	}

	h.logger.InfoContext(ctx, "Handling magic link generated event",
		attr.String("correlation_id", payload.CorrelationID),
		attr.String("user_id", payload.UserID),
		attr.Bool("success", payload.Success),
	)

	// 1. Retrieve stored interaction data
	storedData, err := h.interactionStore.Get(ctx, payload.CorrelationID)
	if err != nil {
		h.logger.ErrorContext(ctx, "Failed to retrieve stored interaction",
			attr.Error(err),
			attr.String("correlation_id", payload.CorrelationID),
		)
		// Can't respond to user without interaction data
		return nil, nil // Don't retry - interaction may have expired
	}

	interactionData, ok := storedData.(dashboard.DashboardInteractionData)
	if !ok {
		h.logger.ErrorContext(ctx, "Invalid interaction data type",
			attr.String("correlation_id", payload.CorrelationID),
		)
		return nil, nil
	}

	// 2. Build interaction for followup
	interaction := &discordgo.Interaction{
		Token: interactionData.InteractionToken,
		AppID: h.cfg.Discord.AppID,
	}

	// 3. Handle error response from backend
	if !payload.Success {
		h.logger.WarnContext(ctx, "Backend returned error for magic link",
			attr.String("error", payload.Error),
			attr.String("user_id", payload.UserID),
		)
		_, _ = h.session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: fmt.Sprintf("Unable to generate dashboard link: %s", payload.Error),
		})
		h.interactionStore.Delete(ctx, payload.CorrelationID)
		return nil, nil
	}

	// 4. Validate URL before attempting to send
	if payload.URL == "" {
		h.logger.ErrorContext(ctx, "Magic link URL is empty despite success=true",
			attr.String("user_id", payload.UserID),
			attr.String("correlation_id", payload.CorrelationID),
		)
		_, _ = h.session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: "Unable to generate dashboard link. Please try again.",
		})
		h.interactionStore.Delete(ctx, payload.CorrelationID)
		return nil, nil
	}

	// 5. Try to send via DM first
	if err := h.sendMagicLinkDM(ctx, payload.UserID, payload.GuildID, payload.URL); err != nil {
		h.logger.WarnContext(ctx, "Could not DM user, sending ephemeral",
			attr.Error(err),
			attr.String("user_id", payload.UserID),
		)
		// Fallback to ephemeral followup; if this also fails, don't delete
		// the interaction so the message can be retried.
		if err := h.sendMagicLinkFollowup(interaction, payload.URL); err != nil {
			return nil, err
		}
		h.interactionStore.Delete(ctx, payload.CorrelationID)
		return nil, nil
	}

	// 5. Confirm DM sent via ephemeral followup
	_, _ = h.session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: "Check your DMs for your dashboard link.",
	})

	h.interactionStore.Delete(ctx, payload.CorrelationID)

	h.logger.InfoContext(ctx, "Magic link sent successfully",
		attr.String("user_id", payload.UserID),
		attr.String("correlation_id", payload.CorrelationID),
	)

	return nil, nil
}

func (h *AuthHandlers) sendMagicLinkDM(ctx context.Context, userID, guildID, url string) error {
	channel, err := h.session.UserChannelCreate(userID)
	if err != nil {
		return fmt.Errorf("failed to create DM channel: %w", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Frolf Dashboard Access",
		Description: "Click the link below to access your dashboard. This link is valid for 24 hours and is unique to you.",
		Color:       0x5865F2, // Discord blurple
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Dashboard Link",
				Value: url,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Do not share this link with others",
		},
	}

	// Create the button component
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Open Dashboard",
					Style: discordgo.LinkButton,
					URL:   url,
				},
			},
		},
	}

	_, err = h.session.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("failed to send DM: %w", err)
	}

	return nil
}

func (h *AuthHandlers) sendMagicLinkFollowup(interaction *discordgo.Interaction, url string) error {
	// Create the button component
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label: "Open Dashboard",
					Style: discordgo.LinkButton,
					URL:   url,
				},
			},
		},
	}

	_, err := h.session.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
		Flags: discordgo.MessageFlagsEphemeral,
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Frolf Dashboard Access",
				Description: "I couldn't send you a DM. Here's your dashboard link (only visible to you):",
				Color:       0x5865F2,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Dashboard Link",
						Value: url,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "This message is only visible to you. Do not share this link.",
				},
			},
		},
		Components: components,
	})
	return err
}
