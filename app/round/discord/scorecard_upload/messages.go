package scorecardupload

import (
	"context"
	"errors"
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

	if msg == nil || msg.Message == nil {
		m.logger.WarnContext(ctx, "Ignoring scorecard upload message with nil payload")
		return
	}

	if msg.Author == nil {
		m.logger.WarnContext(ctx, "Ignoring scorecard upload message with nil author",
			attr.String("channel_id", msg.ChannelID),
			attr.String("guild_id", msg.GuildID),
		)
		return
	}

	if s == nil {
		m.logger.WarnContext(ctx, "Ignoring scorecard upload message with nil discord session",
			attr.String("channel_id", msg.ChannelID),
			attr.String("guild_id", msg.GuildID),
			attr.String("user_id", msg.Author.ID),
		)
		return
	}

	// Ignore bot messages
	if msg.Author.Bot {
		return
	}

	if !m.allowUploadIngress(msg.GuildID, msg.Author.ID) {
		m.logger.WarnContext(ctx, "Rate limited scorecard upload ingress",
			attr.String("user_id", msg.Author.ID),
			attr.String("guild_id", msg.GuildID),
			attr.String("channel_id", msg.ChannelID),
		)
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "Too many upload attempts. Please wait a minute and try again.")
		return
	}

	scorecardFile := firstScorecardAttachment(msg.Attachments)
	udiscURL := extractFirstUDiscURL(msg.Content)
	if scorecardFile == nil && udiscURL == "" {
		return
	}

	// Look up pending upload for this user/channel
	key := fmt.Sprintf("%s:%s", msg.Author.ID, msg.ChannelID)
	m.pendingMutex.RLock()
	_, exists := m.pendingUploads[key]
	m.pendingMutex.RUnlock()

	threadCtx, threadCtxExists := m.threadUploadContext(msg.ChannelID)
	if !exists && !threadCtxExists {
		if scorecardFile != nil {
			m.logger.WarnContext(ctx, "File upload received but no pending upload found",
				attr.String("filename", scorecardFile.Filename),
				attr.String("user_id", msg.Author.ID),
				attr.String("channel_id", msg.ChannelID),
			)
		}
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "No pending scorecard upload found. Please click the 'Upload Scorecard' button first.")
		return
	}

	if scorecardFile == nil && udiscURL != "" {
		m.handleRoundThreadURLUpload(ctx, s, msg, key, udiscURL, exists, threadCtx, threadCtxExists)
		return
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

	// Download the file after confirming there's a pending upload context.
	if scorecardFile.Size > maxAttachmentBytes {
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "File too large. Maximum size is 10MB.")
		return
	}

	shouldInlineFileData := scorecardFile.URL == "" || scorecardFile.Size <= 0 || scorecardFile.Size <= maxInlineFileDataBytes

	var (
		fileData []byte
		err      error
	)
	if shouldInlineFileData {
		fileData, err = m.downloadAttachment(ctx, scorecardFile.URL)
		if err != nil {
			if errors.Is(err, errAttachmentTooLarge) {
				m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "File too large. Maximum size is 10MB.")
				return
			}

			m.logger.ErrorContext(ctx, "Failed to download attachment",
				attr.Error(err),
				attr.String("url", scorecardFile.URL),
			)
			m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "Failed to download file. Please try again.")
			return
		}
	} else {
		m.logger.InfoContext(ctx, "Skipping attachment download for large scorecard; publishing URL reference",
			attr.String("filename", scorecardFile.Filename),
			attr.Int("attachment_size", scorecardFile.Size),
			attr.String("url", scorecardFile.URL),
		)
	}

	var (
		guildID        sharedtypes.GuildID
		roundID        sharedtypes.RoundID
		notes          string
		eventMessageID string
	)
	if exists {
		m.pendingMutex.Lock()
		pending, pendingExists := m.pendingUploads[key]
		if pendingExists {
			delete(m.pendingUploads, key) // Consume the pending upload only after file retrieval succeeded.
			guildID = pending.GuildID
			roundID = pending.RoundID
			notes = pending.Notes
			eventMessageID = pending.EventMessageID
		}
		m.pendingMutex.Unlock()

		if guildID == "" || roundID.String() == "" {
			m.logger.WarnContext(ctx, "Pending upload no longer exists after attachment download",
				attr.String("user_id", msg.Author.ID),
				attr.String("channel_id", msg.ChannelID),
			)
			m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "No pending scorecard upload found. Please click the 'Upload Scorecard' button first.")
			return
		}
	} else {
		guildID = threadCtx.GuildID
		roundID = threadCtx.RoundID
		eventMessageID = threadCtx.EventMessageID
		notes = ""
	}

	m.logger.InfoContext(ctx, "Processing file upload with pending context",
		attr.String("filename", scorecardFile.Filename),
		attr.Int("file_size", scorecardFile.Size),
		attr.Bool("file_data_inline", len(fileData) > 0),
		attr.String("round_id", roundID.String()),
		attr.String("user_id", msg.Author.ID),
		attr.Bool("thread_auto_ingress", threadCtxExists && !exists),
	)

	// Publish scorecard upload event
	importID, err := m.publishScorecardUploadEvent(
		ctx,
		guildID,
		roundID,
		sharedtypes.DiscordID(msg.Author.ID),
		msg.ChannelID,
		eventMessageID,
		fileData,
		scorecardFile.URL,
		scorecardFile.Filename,
		notes,
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
		fmt.Sprintf("✅ Scorecard import started! Import ID: `%s`\n\nI'll match the players and notify you when ready.", importID))
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send file upload confirmation",
			attr.Error(err),
		)
	}
}

func (m *scorecardUploadManager) handleRoundThreadURLUpload(
	ctx context.Context,
	s discord.Session,
	msg *discordgo.MessageCreate,
	pendingKey string,
	udiscURL string,
	hasPending bool,
	threadCtx *threadUploadContext,
	hasThreadCtx bool,
) {
	var (
		guildID        sharedtypes.GuildID
		roundID        sharedtypes.RoundID
		eventMessageID string
		notes          string
	)

	if hasPending {
		m.pendingMutex.Lock()
		pending, exists := m.pendingUploads[pendingKey]
		if exists {
			delete(m.pendingUploads, pendingKey)
			guildID = pending.GuildID
			roundID = pending.RoundID
			eventMessageID = pending.EventMessageID
			notes = pending.Notes
		}
		m.pendingMutex.Unlock()
		if guildID == "" || roundID.String() == "" {
			m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "No pending scorecard upload found. Please click the 'Upload Scorecard' button first.")
			return
		}
	} else if hasThreadCtx {
		guildID = threadCtx.GuildID
		roundID = threadCtx.RoundID
		eventMessageID = threadCtx.EventMessageID
	} else {
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "No pending scorecard upload found. Please click the 'Upload Scorecard' button first.")
		return
	}

	importID, err := m.publishScorecardURLEvent(ctx, guildID, roundID, sharedtypes.DiscordID(msg.Author.ID), msg.ChannelID, eventMessageID, udiscURL, notes)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to publish scorecard URL event from message",
			attr.Error(err),
			attr.String("channel_id", msg.ChannelID),
			attr.String("round_id", roundID.String()),
		)
		m.sendFileUploadErrorMessage(ctx, s, msg.ChannelID, "Failed to process scorecard URL. Please try again.")
		return
	}

	_, err = s.ChannelMessageSend(msg.ChannelID,
		fmt.Sprintf("✅ Scorecard import started! Import ID: `%s`\n\nI'll match the players and notify you when ready.", importID))
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send URL upload confirmation",
			attr.Error(err),
			attr.String("channel_id", msg.ChannelID),
		)
	}
}

func firstScorecardAttachment(attachments []*discordgo.MessageAttachment) *discordgo.MessageAttachment {
	for _, attachment := range attachments {
		filename := strings.ToLower(strings.TrimSpace(attachment.Filename))
		if strings.HasSuffix(filename, ".csv") || strings.HasSuffix(filename, ".xlsx") {
			return attachment
		}
	}
	return nil
}

func extractFirstUDiscURL(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	candidates := strings.Fields(content)
	for _, candidate := range candidates {
		trimmed := strings.Trim(candidate, " \t\r\n<>()[]{}.,!?:;\"'")
		if trimmed == "" {
			continue
		}
		if err := validateUDiscURL(trimmed); err == nil {
			return trimmed
		}
	}
	return ""
}

// SendUploadError sends an error message. If the channel is a guild channel,
// sends a DM to the user instead to avoid leaking import errors publicly.
func (m *scorecardUploadManager) SendUploadError(ctx context.Context, channelID, userID, errorMsg string) error {
	// Fallback for empty error message
	if errorMsg == "" {
		errorMsg = "An unknown error occurred while processing the scorecard."
	}

	// Get channel info to determine if it's a DM or guild channel
	channel, err := m.session.GetChannel(channelID)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to get channel info",
			attr.Error(err),
			attr.String("channel_id", channelID),
		)
		// Fallback: try to send to channel anyway
		_, sendErr := m.session.ChannelMessageSend(channelID, fmt.Sprintf("❌ Scorecard import failed: %s", errorMsg))
		if sendErr != nil {
			return sendErr
		}
		return nil
	}

	// If it's a DM channel (type 1), send error there
	if channel.Type == 1 {
		_, err := m.session.ChannelMessageSend(channelID, fmt.Sprintf("❌ Scorecard import failed: %s", errorMsg))
		if err != nil {
			m.logger.ErrorContext(ctx, "Failed to send upload error message to DM",
				attr.Error(err),
				attr.String("channel_id", channelID),
			)
			return err
		}
		return nil
	}

	// It's a guild channel - send DM to user instead
	if userID == "" {
		m.logger.WarnContext(ctx, "Cannot send DM: userID is empty, falling back to channel message")
		_, err := m.session.ChannelMessageSend(channelID, fmt.Sprintf("❌ Scorecard import failed: %s", errorMsg))
		return err
	}

	// Create DM channel with user
	dmChannel, err := m.session.UserChannelCreate(userID)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to create DM channel",
			attr.Error(err),
			attr.String("user_id", userID),
		)
		// Fallback: send ephemeral-style message in channel
		_, sendErr := m.session.ChannelMessageSend(channelID,
			fmt.Sprintf("<@%s> Your scorecard import failed (DM sent privately)", userID))
		if sendErr != nil {
			return fmt.Errorf("failed to create DM and send fallback: %w", err)
		}
		return err
	}

	// Send error via DM
	_, err = m.session.ChannelMessageSend(dmChannel.ID,
		fmt.Sprintf("❌ Scorecard import failed: %s\n\n(This message was sent privately to avoid cluttering the channel)", errorMsg))
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to send upload error via DM",
			attr.Error(err),
			attr.String("user_id", userID),
		)
		return err
	}

	m.logger.InfoContext(ctx, "Sent import error via DM to user",
		attr.String("user_id", userID),
		attr.String("original_channel", channelID))

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
