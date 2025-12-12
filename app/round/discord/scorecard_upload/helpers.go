package scorecardupload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// sendUploadConfirmation sends an ephemeral response confirming the upload started.
func (m *scorecardUploadManager) sendUploadConfirmation(ctx context.Context, s discord.Session, i *discordgo.Interaction, importID string) error {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Scorecard import started! Import ID: `%s`\n\nI'll match the players and notify you when ready.", importID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	err := s.InteractionRespond(i, response)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send upload confirmation", attr.Error(err))
		return err
	}

	return nil
}

// sendFileUploadPrompt sends an ephemeral message instructing the user to upload a CSV/XLSX file.
func (m *scorecardUploadManager) sendFileUploadPrompt(ctx context.Context, s discord.Session, i *discordgo.Interaction, roundID sharedtypes.RoundID, notes string) error {
	// Store pending upload expectation
	userID := i.Member.User.ID
	channelID := i.ChannelID
	key := fmt.Sprintf("%s:%s", userID, channelID)

	m.pendingMutex.Lock()
	m.pendingUploads[key] = &pendingUpload{
		RoundID:   roundID,
		Notes:     notes,
		CreatedAt: time.Now(),
	}
	m.pendingMutex.Unlock()

	m.logger.InfoContext(ctx, "Stored pending upload expectation",
		attr.String("user_id", userID),
		attr.String("channel_id", channelID),
		attr.String("round_id", roundID.String()),
	)

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("📁 **Please upload your scorecard file**\n\n"+
				"Reply to this message with a CSV or XLSX file from UDisc.\n"+
				"Round ID: `%s`\n"+
				"Notes: %s\n\n"+
				"I'll process it and match the players automatically.\n\n"+
				"_This upload prompt expires in 5 minutes._",
				roundID.String(), notes),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}

	err := s.InteractionRespond(i, response)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send file upload prompt", attr.Error(err))
		return err
	}

	return nil
}

// sendUploadError sends an ephemeral error response.
// This will be used when error handling is fully implemented.
func (m *scorecardUploadManager) sendUploadError(ctx context.Context, s discord.Session, i *discordgo.Interaction, errorMsg string) error {
	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Scorecard upload failed: %s", errorMsg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	err := s.InteractionRespond(i, response)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send upload error response", attr.Error(err))
		return err
	}

	return nil
}

// downloadAttachment downloads a file from a Discord CDN URL.
func (m *scorecardUploadManager) downloadAttachment(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download attachment: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment data: %w", err)
	}

	return data, nil
}
