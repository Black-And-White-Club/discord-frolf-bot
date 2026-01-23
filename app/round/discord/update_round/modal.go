package updateround

import (
	"context"
	"fmt"
	"strings"
	"time"

	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type interactionResponder interface {
	InteractionRespond(
		*discordgo.Interaction,
		*discordgo.InteractionResponse,
		...discordgo.RequestOption,
	) error
}

// SendUpdateRoundModal sends a modal to allow updating round details
func (urm *updateRoundManager) SendUpdateRoundModal(
	ctx context.Context,
	i *discordgo.InteractionCreate,
	roundID sharedtypes.RoundID,
) (UpdateRoundOperationResult, error) {

	if err := ctx.Err(); err != nil {
		return UpdateRoundOperationResult{Error: err}, err
	}
	if i == nil || i.Interaction == nil {
		err := fmt.Errorf("interaction is nil or incomplete")
		return UpdateRoundOperationResult{Error: err}, err
	}

	userID := getUserIDFromInteraction(i)
	if userID == "" {
		err := fmt.Errorf("user ID is missing")
		return UpdateRoundOperationResult{Error: err}, err
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "send_update_round_modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "command")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)

	return urm.operationWrapper(ctx, "send_update_round_modal", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		messageID := ""
		if i.Message != nil {
			messageID = i.Message.ID
		}

		customID := fmt.Sprintf("update_round_modal|%s|%s", roundID, messageID)

		err := urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Title:    "Update Round",
				CustomID: customID,
				Components: []discordgo.MessageComponent{
					textRow("title", "Title", 100),
					textAreaRow("description", "Description", 500),
					textRow("start_time", "Start Time", 30),
					textRow("timezone", "Timezone", 50),
					textRow("location", "Location", 100),
				},
			},
		})
		if err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}

		return UpdateRoundOperationResult{Success: "modal sent"}, nil
	})
}

// HandleUpdateRoundModalSubmit processes the submission of the update round modal
func (urm *updateRoundManager) HandleUpdateRoundModalSubmit(
	ctx context.Context,
	i *discordgo.InteractionCreate,
) (UpdateRoundOperationResult, error) {

	if i == nil || i.Interaction == nil {
		err := fmt.Errorf("interaction is nil or incomplete")
		return UpdateRoundOperationResult{Error: err}, err
	}

	userID := getUserIDFromInteraction(i)
	if userID == "" {
		err := fmt.Errorf("user ID is missing")
		return UpdateRoundOperationResult{Error: err}, err
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_update_round_modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_submit")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)

	return urm.operationWrapper(ctx, "handle_update_round_modal_submit", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		data := i.ModalSubmitData()
		parts := strings.Split(data.CustomID, "|")
		if len(parts) < 3 {
			respondError(urm.session, i.Interaction, "Invalid modal format. Please try again.")
			return UpdateRoundOperationResult{Error: fmt.Errorf("invalid modal custom_id")}, nil
		}

		roundUUID, err := uuid.Parse(parts[1])
		if err != nil {
			respondError(urm.session, i.Interaction, "Invalid round ID.")
			return UpdateRoundOperationResult{Error: err}, nil
		}

		title := strings.TrimSpace(getInput(data, 0))
		description := strings.TrimSpace(getInput(data, 1))
		startTime := strings.TrimSpace(getInput(data, 2))
		timezone := strings.TrimSpace(getInput(data, 3))
		location := strings.TrimSpace(getInput(data, 4))

		if title == "" && description == "" && startTime == "" && timezone == "" && location == "" {
			respondError(urm.session, i.Interaction, "Please fill at least one field.")
			return UpdateRoundOperationResult{Error: fmt.Errorf("no fields provided")}, nil
		}

		if timezone == "" {
			timezone = "America/Chicago"
		}

		payload := discordroundevents.RoundUpdateModalSubmittedPayloadV1{
			GuildID:     sharedtypes.GuildID(i.GuildID),
			RoundID:     sharedtypes.RoundID(roundUUID),
			UserID:      sharedtypes.DiscordID(userID),
			ChannelID:   i.ChannelID,
			MessageID:   parts[2],
			Title:       optionalTitle(title),
			Description: optionalDescription(description),
			StartTime:   optionalString(startTime),
			Location:    optionalLocation(location),
			Timezone:    optionalTimezone(timezone),
		}

		_ = urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Round update request received.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		msg, err := urm.createEvent(ctx, discordroundevents.RoundUpdateModalSubmittedV1, payload, i)
		if err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}

		msg.Metadata.Set("submitted_at", time.Now().UTC().Format(time.RFC3339))
		msg.Metadata.Set("user_id", userID)
		msg.Metadata.Set("user_timezone", timezone)
		msg.Metadata.Set("raw_start_time", startTime)

		if err := urm.publisher.Publish(discordroundevents.RoundUpdateModalSubmittedV1, msg); err != nil {
			return UpdateRoundOperationResult{Error: err}, err
		}

		return UpdateRoundOperationResult{Success: "round update request published"}, nil
	})
}

// HandleUpdateRoundModalCancel handles a user's cancellation of the update round modal.
func (urm *updateRoundManager) HandleUpdateRoundModalCancel(
	ctx context.Context,
	i *discordgo.InteractionCreate,
) (UpdateRoundOperationResult, error) {

	if i == nil || i.Interaction == nil {
		err := fmt.Errorf("interaction is nil or incomplete")
		return UpdateRoundOperationResult{Error: err}, err
	}

	userID := getUserIDFromInteraction(i)
	if userID == "" {
		err := fmt.Errorf("user ID is missing")
		return UpdateRoundOperationResult{Error: err}, err
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "handle_update_round_modal_cancel")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal_cancel")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, i.GuildID)

	return urm.operationWrapper(ctx, "handle_update_round_modal_cancel", func(ctx context.Context) (UpdateRoundOperationResult, error) {
		// Acknowledge cancellation to the user (ephemeral)
		_ = urm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Round update cancelled.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

		return UpdateRoundOperationResult{Success: "cancelled"}, nil
	})
}

/* ---------- Helpers ---------- */

func optionalTitle(v string) *roundtypes.Title {
	if v == "" {
		return nil
	}
	t := roundtypes.Title(v)
	return &t
}

func optionalDescription(v string) *roundtypes.Description {
	if v == "" {
		return nil
	}
	d := roundtypes.Description(v)
	return &d
}

func optionalLocation(v string) *roundtypes.Location {
	if v == "" {
		return nil
	}
	d := roundtypes.Location(v)
	return &d
}

func optionalTimezone(v string) *roundtypes.Timezone {
	if v == "" {
		return nil
	}
	tz := roundtypes.Timezone(v)
	return &tz
}

func optionalString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func respondError(session interactionResponder, i *discordgo.Interaction, msg string) {
	_ = session.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func getInput(data discordgo.ModalSubmitInteractionData, idx int) string {
	return data.Components[idx].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value
}

func getUserIDFromInteraction(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func textRow(id, label string, max int) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:  id,
				Label:     label,
				Style:     discordgo.TextInputShort,
				MaxLength: max,
			},
		},
	}
}

func textAreaRow(id, label string, max int) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.TextInput{
				CustomID:  id,
				Label:     label,
				Style:     discordgo.TextInputParagraph,
				MaxLength: max,
			},
		},
	}
}
