package scorecardupload

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

const maxInlineFileDataBytes = 256 * 1024

// publishScorecardURLEvent publishes a scorecard URL requested event to the message bus.
func (m *scorecardUploadManager) publishScorecardURLEvent(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, channelID, messageID, uDiscURL, notes string) (string, error) {
	importID := uuid.New().String()

	payload := roundevents.ScorecardURLRequestedPayloadV1{
		ImportID:  importID,
		GuildID:   guildID,
		RoundID:   roundID,
		UserID:    userID,
		ChannelID: channelID,
		MessageID: messageID,
		UDiscURL:  uDiscURL,
		Notes:     notes,
		Timestamp: time.Now().UTC(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to marshal scorecard URL payload", attr.Error(err))
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("event_name", roundevents.ScorecardURLRequestedV1)
	msg.Metadata.Set("domain", "scorecard")
	msg.Metadata.Set("guild_id", string(guildID))
	msg.Metadata.Set("import_id", importID)

	err = m.publisher.Publish(roundevents.ScorecardURLRequestedV1, msg)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to publish scorecard URL requested event", attr.Error(err))
		return "", fmt.Errorf("failed to publish event: %w", err)
	}

	m.logger.InfoContext(ctx, "Published scorecard URL requested event",
		attr.String("import_id", importID),
		attr.String("guild_id", string(guildID)),
		attr.String("round_id", roundID.String()),
	)

	return importID, nil
}

// publishScorecardUploadEvent publishes a scorecard uploaded event (for file uploads) to the message bus.
// This will be used when file upload is fully implemented.
func (m *scorecardUploadManager) publishScorecardUploadEvent(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, channelID, messageID string, fileData []byte, fileURL, fileName, notes string) (string, error) {
	importID := uuid.New().String()

	inlineData := fileData
	if len(fileData) > maxInlineFileDataBytes && fileURL != "" {
		inlineData = nil
		m.logger.InfoContext(ctx, "Publishing scorecard upload by URL reference",
			attr.String("import_id", importID),
			attr.Int("inline_file_bytes", len(fileData)),
			attr.String("file_url", fileURL))
	}

	payload := roundevents.ScorecardUploadedPayloadV1{
		ImportID:  importID,
		GuildID:   guildID,
		RoundID:   roundID,
		UserID:    userID,
		ChannelID: channelID,
		MessageID: messageID,
		FileData:  inlineData,
		FileURL:   fileURL,
		FileName:  fileName,
		Notes:     notes,
		Timestamp: time.Now().UTC(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to marshal scorecard upload payload", attr.Error(err))
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("event_name", roundevents.ScorecardUploadedV1)
	msg.Metadata.Set("domain", "scorecard")
	msg.Metadata.Set("guild_id", string(guildID))
	msg.Metadata.Set("import_id", importID)

	err = m.publisher.Publish(roundevents.ScorecardUploadedV1, msg)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to publish scorecard uploaded event", attr.Error(err))
		return "", fmt.Errorf("failed to publish event: %w", err)
	}

	m.logger.InfoContext(ctx, "Published scorecard upload event",
		attr.String("import_id", importID),
		attr.String("guild_id", string(guildID)),
		attr.String("round_id", roundID.String()),
	)

	return importID, nil
}
