package history

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/discordutils"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

const correlationIDKey = "correlation_id"

// HandleHistoryCommand dispatches /history subcommands.
func (hm *historyManager) HandleHistoryCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.User == nil {
		hm.logger.WarnContext(ctx, "History command received without member context (DM?)")
		return
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "history")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "application_command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		hm.logger.WarnContext(ctx, "No options provided for history command")
		return
	}

	subCommand := options[0].Name
	switch subCommand {
	case "member":
		hm.handleMemberHistory(ctx, i, options[0].Options)
	case "chart":
		hm.handleMemberChart(ctx, i, options[0].Options)
	default:
		hm.logger.WarnContext(ctx, "Unknown subcommand", attr.String("subcommand", subCommand))
		if err := hm.respondWithError(ctx, i, "Unknown subcommand"); err != nil {
			hm.logger.ErrorContext(ctx, "Failed to respond with error", attr.Error(err))
		}
	}
}

// handleMemberHistory requests tag history for a specific member.
func (hm *historyManager) handleMemberHistory(ctx context.Context, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var userID string
	limit := 50 // Default limit

	for _, opt := range options {
		if opt.Name == "user" {
			userID = opt.UserValue(nil).ID
		}
		if opt.Name == "limit" {
			limit = int(opt.IntValue())
			if limit > 100 {
				limit = 100
			}
		}
	}

	// Default to self if no user specified
	if userID == "" {
		userID = i.Member.User.ID
	}

	// Defer the response since this may take a moment
	err := hm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to defer interaction", attr.Error(err))
		return
	}

	// Generate correlation ID
	correlationID := uuid.New().String()

	// Store interaction reference for the response handler
	if err := hm.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
		hm.logger.ErrorContext(ctx, "Failed to store interaction", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to process request")
		return
	}

	guildID := i.GuildID
	payload := &leaderboardevents.TagHistoryRequestedPayloadV1{
		GuildID:  guildID,
		MemberID: userID,
		Limit:    limit,
	}

	msg, err := hm.helper.CreateNewMessage(payload, leaderboardevents.LeaderboardTagHistoryRequestedV1)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to create message", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to process request")
		return
	}

	if msg.Metadata == nil {
		msg.Metadata = message.Metadata{}
	}
	msg.Metadata.Set("guild_id", guildID)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := hm.publisher.Publish(leaderboardevents.LeaderboardTagHistoryRequestedV1, msg); err != nil {
		hm.logger.ErrorContext(ctx, "Failed to publish tag history request", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to request tag history")
		return
	}

	_, err = hm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &[]string{"📊 Fetching tag history..."}[0],
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to edit interaction response", attr.Error(err))
	}
}

// handleMemberChart requests a PNG tag history chart for a member.
func (hm *historyManager) handleMemberChart(ctx context.Context, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var userID string
	for _, opt := range options {
		if opt.Name == "user" {
			userID = opt.UserValue(nil).ID
		}
	}

	// Default to self if no user specified
	if userID == "" {
		userID = i.Member.User.ID
	}

	// Defer the response since chart generation may take a moment
	err := hm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to defer interaction", attr.Error(err))
		return
	}

	// Generate correlation ID
	correlationID := uuid.New().String()

	// Store interaction reference for the response handler
	if err := hm.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
		hm.logger.ErrorContext(ctx, "Failed to store interaction", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to process request")
		return
	}

	guildID := i.GuildID
	payload := &leaderboardevents.TagGraphRequestedPayloadV1{
		GuildID:  guildID,
		MemberID: userID,
	}

	msg, err := hm.helper.CreateNewMessage(payload, leaderboardevents.LeaderboardTagGraphRequestedV1)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to create message", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to process request")
		return
	}

	if msg.Metadata == nil {
		msg.Metadata = message.Metadata{}
	}
	msg.Metadata.Set("guild_id", guildID)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := hm.publisher.Publish(leaderboardevents.LeaderboardTagGraphRequestedV1, msg); err != nil {
		hm.logger.ErrorContext(ctx, "Failed to publish tag graph request", attr.Error(err))
		hm.followupWithError(ctx, i, "Failed to request tag chart")
		return
	}

	_, err = hm.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &[]string{"📈 Generating tag chart..."}[0],
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to edit interaction response", attr.Error(err))
	}
}

// HandleTagHistoryResponse handles the tag history response from the backend.
func (hm *historyManager) HandleTagHistoryResponse(ctx context.Context, payload *leaderboardevents.TagHistoryResponsePayloadV1) {
	correlationID := correlationIDFromContext(ctx)
	if correlationID == "" {
		hm.logger.WarnContext(ctx, "Received tag history response without correlation ID")
		return
	}

	i, err := discordutils.GetInteraction(ctx, hm.interactionStore, correlationID)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to retrieve interaction for history response", attr.Error(err))
		return
	}
	defer hm.interactionStore.Delete(ctx, correlationID)

	if len(payload.Entries) == 0 {
		_, err := hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
			Content: &[]string{"No tag history found."}[0],
		})
		if err != nil {
			hm.logger.ErrorContext(ctx, "Failed to send empty history response", attr.Error(err))
		}
		return
	}

	var sb strings.Builder
	sb.WriteString("📊 **Tag History**\n\n")
	for _, entry := range payload.Entries {
		sb.WriteString(fmt.Sprintf("Tag **#%d**: <@%s>", entry.TagNumber, entry.NewMemberID))
		if entry.OldMemberID != "" {
			sb.WriteString(fmt.Sprintf(" (was <@%s>)", entry.OldMemberID))
		}
		sb.WriteString(fmt.Sprintf(" — %s (%s)\n", entry.Reason, entry.CreatedAt))
	}

	content := sb.String()
	// Check for length limit (Discord 2000 chars) using rune count for safety
	runes := []rune(content)
	if len(runes) > 2000 {
		content = string(runes[:1997]) + "..."
	}

	_, err = hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to send history response", attr.Error(err))
	}
}

// HandleTagHistoryFailed handles a failed tag history request.
func (hm *historyManager) HandleTagHistoryFailed(ctx context.Context, payload *leaderboardevents.TagHistoryFailedPayloadV1) {
	correlationID := correlationIDFromContext(ctx)
	if correlationID == "" {
		hm.logger.WarnContext(ctx, "Received tag history failure without correlation ID")
		return
	}

	i, err := discordutils.GetInteraction(ctx, hm.interactionStore, correlationID)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to retrieve interaction for history failure", attr.Error(err))
		return
	}
	defer hm.interactionStore.Delete(ctx, correlationID)

	// Sanitize reason to prevent markdown injection
	sanitizedReason := strings.ReplaceAll(payload.Reason, "`", "'")
	content := fmt.Sprintf("❌ Failed to fetch history: `%s`", sanitizedReason)
	_, err = hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to send history error response", attr.Error(err))
	}
}

// HandleTagGraphResponse handles the tag graph PNG response from the backend.
func (hm *historyManager) HandleTagGraphResponse(ctx context.Context, payload *leaderboardevents.TagGraphResponsePayloadV1) {
	correlationID := correlationIDFromContext(ctx)
	if correlationID == "" {
		hm.logger.WarnContext(ctx, "Received tag graph response without correlation ID")
		return
	}

	i, err := discordutils.GetInteraction(ctx, hm.interactionStore, correlationID)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to retrieve interaction for graph response", attr.Error(err))
		return
	}
	defer hm.interactionStore.Delete(ctx, correlationID)

	if len(payload.PNGData) == 0 {
		_, err := hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
			Content: &[]string{"❌ Generated chart was empty."}[0],
		})
		if err != nil {
			hm.logger.ErrorContext(ctx, "Failed to send empty graph response", attr.Error(err))
		}
		return
	}

	// Use FollowupMessageCreate to send the file attachment
	// Note: We can't easily attach files via InteractionResponseEdit in all discordgo versions,
	// so a followup is safer and cleaner for the image. We'll edit the original message to say "Chart generated:"

	_, err = hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &[]string{"📈 Chart generated:"}[0],
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to edit loading message", attr.Error(err))
	}

	reader := bytes.NewReader(payload.PNGData)
	file := &discordgo.File{
		Name:        "tag_history.png",
		ContentType: "image/png",
		Reader:      reader,
	}

	_, err = hm.session.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Files: []*discordgo.File{file},
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to send graph attachment", attr.Error(err))
	}
}

// HandleTagGraphFailed handles a failed tag graph request.
func (hm *historyManager) HandleTagGraphFailed(ctx context.Context, payload *leaderboardevents.TagGraphFailedPayloadV1) {
	correlationID := correlationIDFromContext(ctx)
	if correlationID == "" {
		hm.logger.WarnContext(ctx, "Received tag graph failure without correlation ID")
		return
	}

	i, err := discordutils.GetInteraction(ctx, hm.interactionStore, correlationID)
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to retrieve interaction for graph failure", attr.Error(err))
		return
	}
	defer hm.interactionStore.Delete(ctx, correlationID)

	// Sanitize reason to prevent markdown injection
	sanitizedReason := strings.ReplaceAll(payload.Reason, "`", "'")
	content := fmt.Sprintf("❌ Failed to generate chart: `%s`", sanitizedReason)
	_, err = hm.session.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to send graph error response", attr.Error(err))
	}
}

// respondWithError sends an ephemeral error response.
func (hm *historyManager) respondWithError(ctx context.Context, i *discordgo.InteractionCreate, msg string) error {
	err := hm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", msg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return err
}

// followupWithError sends a follow-up error message for deferred interactions.
func (hm *historyManager) followupWithError(ctx context.Context, i *discordgo.InteractionCreate, msg string) {
	_, err := hm.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("❌ %s", msg),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		hm.logger.ErrorContext(ctx, "Failed to send followup error", attr.Error(err))
	}
}

// correlationIDFromContext extracts the correlation ID from the context.
func correlationIDFromContext(ctx context.Context) string {
	if val := ctx.Value(correlationIDKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
