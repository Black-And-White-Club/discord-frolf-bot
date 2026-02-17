package scorecardupload

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// HandleScorecardUploadButton handles the upload scorecard button interaction.
func (m *scorecardUploadManager) HandleScorecardUploadButton(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	return m.operationWrapper(ctx, "HandleScorecardUploadButton", func(ctx context.Context) (ScorecardUploadOperationResult, error) {
		customID := i.MessageComponentData().CustomID
		parts := strings.Split(customID, "|")
		roundID := ""
		if len(parts) >= 2 {
			roundID = parts[1]
		}

		m.logger.InfoContext(ctx, "Handling scorecard upload button",
			attr.String("round_id", roundID),
			attr.String("user_id", i.Member.User.ID),
			attr.String("guild_id", i.GuildID),
		)

		modal := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: fmt.Sprintf("scorecard_upload_modal|%s", roundID),
				Title:    "Upload Scorecard",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "udisc_url_input",
								Label:       "UDisc URL (or reply with file)",
								Style:       discordgo.TextInputShort,
								Placeholder: "https://udisc.com/... (leave empty to upload file)",
								Required:    false,
								MaxLength:   1000,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID:    "notes_input",
								Label:       "Notes (Optional)",
								Style:       discordgo.TextInputParagraph,
								Placeholder: "Any notes about this scorecard?",
								Required:    false,
								MaxLength:   500,
							},
						},
					},
				},
			},
		}

		err := m.session.InteractionRespond(i.Interaction, modal)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to send modal", attr.Error(err))
			return ScorecardUploadOperationResult{Error: err}, err
		}

		return ScorecardUploadOperationResult{
			Success: "modal_sent",
		}, nil
	})
}

// HandleScorecardUploadModalSubmit handles the scorecard upload modal submission.
func (m *scorecardUploadManager) HandleScorecardUploadModalSubmit(ctx context.Context, i *discordgo.InteractionCreate) (ScorecardUploadOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "scorecard_upload_modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "modal")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)

	return m.operationWrapper(ctx, "HandleScorecardUploadModalSubmit", func(ctx context.Context) (ScorecardUploadOperationResult, error) {
		data := i.ModalSubmitData()

		// Extract roundID from customID
		parts := strings.Split(data.CustomID, "|")
		roundID := ""
		if len(parts) >= 2 {
			roundID = parts[1]
		}

		if roundID == "" {
			m.logger.ErrorContext(ctx, "Missing roundID in modal customID")
			return ScorecardUploadOperationResult{Error: fmt.Errorf("missing round ID")}, fmt.Errorf("missing round ID")
		}

		// Parse roundID as UUID
		parsedRoundID, err := uuid.Parse(roundID)
		if err != nil {
			m.logger.ErrorContext(ctx, "Invalid roundID format", attr.Error(err))
			return ScorecardUploadOperationResult{Error: fmt.Errorf("invalid round ID format")}, fmt.Errorf("invalid round ID format: %w", err)
		}

		// Extract form values
		var uDiscURL, notes string
		for _, component := range data.Components {
			if actionRow, ok := component.(*discordgo.ActionsRow); ok {
				for _, comp := range actionRow.Components {
					if textInput, ok := comp.(*discordgo.TextInput); ok {
						switch textInput.CustomID {
						case "udisc_url_input":
							uDiscURL = textInput.Value
						case "notes_input":
							notes = textInput.Value
						}
					}
				}
			}
		}

		guildID := sharedtypes.GuildID(i.GuildID)
		userID := sharedtypes.DiscordID(i.Member.User.ID)
		channelID := i.ChannelID
		messageID := ""
		if i.Message != nil {
			messageID = i.Message.ID
		}

		// Route based on whether URL was provided
		uDiscURL = strings.TrimSpace(uDiscURL)
		if uDiscURL != "" {
			if err := validateUDiscURL(uDiscURL); err != nil {
				m.logger.WarnContext(ctx, "Rejected invalid UDisc URL",
					attr.String("guild_id", i.GuildID),
					attr.String("user_id", i.Member.User.ID),
					attr.Error(err),
				)
				_ = m.sendUploadError(ctx, m.session, i.Interaction, "Please provide a valid HTTPS URL on udisc.com.")
				return ScorecardUploadOperationResult{Error: err}, nil
			}

			// URL-based import
			importID, err := m.publishScorecardURLEvent(ctx, guildID, sharedtypes.RoundID(parsedRoundID), userID, channelID, messageID, uDiscURL, notes)
			if err != nil {
				m.logger.ErrorContext(ctx, "Failed to publish scorecard URL event", attr.Error(err))
				_ = m.sendUploadError(ctx, m.session, i.Interaction, "Failed to upload scorecard from URL. Please try again later.")
				return ScorecardUploadOperationResult{}, err
			}

			err = m.sendUploadConfirmation(ctx, m.session, i.Interaction, importID)
			if err != nil {
				m.logger.ErrorContext(ctx, "Failed to send upload confirmation", attr.Error(err))
				return ScorecardUploadOperationResult{}, err
			}

			return ScorecardUploadOperationResult{
				Success: importID,
			}, nil
		}

		// File upload flow - prompt user to upload file
		err = m.sendFileUploadPrompt(ctx, m.session, i.Interaction, sharedtypes.RoundID(parsedRoundID), notes, messageID)
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to send file upload prompt", attr.Error(err))
			_ = m.sendUploadError(ctx, m.session, i.Interaction, "Sorry, there was a problem prompting for file upload. Please try again later.")
			return ScorecardUploadOperationResult{}, err
		}

		return ScorecardUploadOperationResult{
			Success: "file_upload_prompted",
		}, nil
	})
}
