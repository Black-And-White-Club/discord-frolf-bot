package scorecardupload

import (
	"context"
	"fmt"
	"strings"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// HandleFileUploadMessage handles file uploads from Discord messages.
func (m *scorecardUploadManager) HandleFileUploadMessage(s discord.Session, msg *discordgo.MessageCreate) {
	ctx := context.Background()

	// Ignore bot messages
	if msg.Author.Bot {
		return
	}

	// Check if message has attachments
	if len(msg.Attachments) == 0 {
		return
	}

	// Look for CSV or XLSX files
	var scorecardFile *discordgo.MessageAttachment
	for _, attachment := range msg.Attachments {
		if strings.HasSuffix(strings.ToLower(attachment.Filename), ".csv") ||
			strings.HasSuffix(strings.ToLower(attachment.Filename), ".xlsx") {
			scorecardFile = attachment
			break
		}
	}

	if scorecardFile == nil {
		return
	}

	m.logger.InfoContext(ctx, "Detected scorecard file upload",
		attr.String("filename", scorecardFile.Filename),
		attr.String("user_id", msg.Author.ID),
		attr.String("channel_id", msg.ChannelID),
		attr.String("guild_id", msg.GuildID),
	)

	// Download the file
	fileData, err := m.downloadAttachment(ctx, scorecardFile.URL)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to download attachment",
			attr.Error(err),
			attr.String("url", scorecardFile.URL),
		)
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "Failed to download file. Please try again.")
		return
	}

	// Validate file size (10MB limit)
	const maxFileSize = 10 * 1024 * 1024
	if len(fileData) > maxFileSize {
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "File too large. Maximum size is 10MB.")
		return
	}

	// Look up pending upload for this user/channel
	key := fmt.Sprintf("%s:%s", msg.Author.ID, msg.ChannelID)
	m.pendingMutex.Lock()
	pending, exists := m.pendingUploads[key]
	if exists {
		delete(m.pendingUploads, key) // Consume the pending upload
	}
	m.pendingMutex.Unlock()

	if !exists {
		m.logger.WarnContext(ctx, "File upload received but no pending upload found",
			attr.String("filename", scorecardFile.Filename),
			attr.String("user_id", msg.Author.ID),
			attr.String("channel_id", msg.ChannelID),
		)
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID,
			"No pending scorecard upload found. Please click the 'Upload Scorecard' button first.")
		return
	}

	m.logger.InfoContext(ctx, "Processing file upload with pending context",
		attr.String("filename", scorecardFile.Filename),
		attr.Int("file_size", len(fileData)),
		attr.String("round_id", pending.RoundID.String()),
		attr.String("user_id", msg.Author.ID),
	)

	// Publish scorecard upload event
	importID, err := m.publishScorecardUploadEvent(
		ctx,
		pending.GuildID,
		pending.RoundID,
		sharedtypes.DiscordID(msg.Author.ID),
		msg.ChannelID,
		pending.EventMessageID,
		fileData,
		scorecardFile.URL,
		scorecardFile.Filename,
		pending.Notes,
	)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to publish scorecard upload event",
			attr.Error(err),
		)
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID,
			"Failed to process scorecard upload. Please try again.")
		return
	}

	// Send confirmation
	_, err = s.ChannelMessageSend(msg.ChannelID,
		fmt.Sprintf("✅ Scorecard uploaded successfully! Import ID: `%s`\n\nI'll match the players and notify you when ready.", importID))
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send file upload confirmation",
			attr.Error(err),
		)
	}
}

// SendUploadError sends an error message to the specified channel.
func (m *scorecardUploadManager) SendUploadError(ctx context.Context, channelID, errorMsg string) error {
	_, err := m.session.ChannelMessageSend(channelID, fmt.Sprintf("❌ Scorecard import failed: %s", errorMsg))
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send upload error message",
			attr.Error(err),
			attr.String("channel_id", channelID),
		)
		return err
	}
	return nil
}

// sendFileUploadErrorMessage sends an error message in the channel.
func (m *scorecardUploadManager) sendFileUploadErrorMessage(ctx context.Context, s discord.Session, channelID, errorMsg string) {
	_, err := s.ChannelMessageSend(channelID, errorMsg)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send file upload error message",
			attr.Error(err),
			attr.String("channel_id", channelID),
		)
	}
}
