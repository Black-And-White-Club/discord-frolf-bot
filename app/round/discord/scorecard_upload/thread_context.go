package scorecardupload

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

func (m *scorecardUploadManager) EnsureRoundThreadInstructions(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	parentChannelID, eventMessageID string,
) error {
	if parentChannelID == "" {
		return fmt.Errorf("missing parent channel id for round thread instructions")
	}
	if eventMessageID == "" {
		return fmt.Errorf("missing event message id for round thread instructions")
	}

	threadID, err := m.resolveOrCreateRoundThread(ctx, parentChannelID, eventMessageID, roundID)
	if err != nil {
		return err
	}

	m.threadMutex.Lock()
	threadCtx, exists := m.threadContexts[threadID]
	if !exists {
		threadCtx = &threadUploadContext{}
		m.threadContexts[threadID] = threadCtx
	}
	threadCtx.RoundID = roundID
	threadCtx.GuildID = guildID
	threadCtx.EventMessageID = eventMessageID
	if threadCtx.CreatedAt.IsZero() {
		threadCtx.CreatedAt = time.Now()
	}
	alreadyPosted := threadCtx.InstructionsPosted
	alreadyPosting := threadCtx.InstructionsPosting
	if !alreadyPosted && !alreadyPosting {
		threadCtx.InstructionsPosting = true
	}
	m.threadMutex.Unlock()

	if alreadyPosted || alreadyPosting {
		return nil
	}

	_, err = m.session.ChannelMessageSend(threadID, roundThreadUploadInstructions(roundID))
	if err != nil {
		m.threadMutex.Lock()
		if current, ok := m.threadContexts[threadID]; ok {
			current.InstructionsPosting = false
			current.CreatedAt = time.Now()
		}
		m.threadMutex.Unlock()
		return fmt.Errorf("failed to send round upload instructions: %w", err)
	}

	m.threadMutex.Lock()
	if current, ok := m.threadContexts[threadID]; ok {
		current.InstructionsPosted = true
		current.InstructionsPosting = false
		current.CreatedAt = time.Now()
	}
	m.threadMutex.Unlock()

	m.logger.InfoContext(ctx, "Posted round scorecard upload instructions",
		attr.String("thread_id", threadID),
		attr.String("round_id", roundID.String()),
		attr.String("guild_id", string(guildID)),
	)

	return nil
}

func (m *scorecardUploadManager) resolveOrCreateRoundThread(
	ctx context.Context,
	parentChannelID, eventMessageID string,
	roundID sharedtypes.RoundID,
) (string, error) {
	if message, err := m.session.ChannelMessage(parentChannelID, eventMessageID); err == nil && message != nil && message.Thread != nil {
		return message.Thread.ID, nil
	}

	threadName := fmt.Sprintf("📋 Scorecards %s", shortRoundID(roundID))
	thread, err := m.session.MessageThreadStartComplex(parentChannelID, eventMessageID, &discordgo.ThreadStart{
		Name: threadName,
		Type: discordgo.ChannelTypeGuildPublicThread,
	})
	if err == nil && thread != nil {
		return thread.ID, nil
	}
	if err != nil && !threadCreationAlreadyExists(err) {
		return "", fmt.Errorf("failed creating scorecard upload thread: %w", err)
	}

	// Discord can return "already exists" or, in rare cases, no thread object on a
	// successful creation response. In both cases, fetch the message and use the
	// attached thread reference.
	message, fetchErr := m.session.ChannelMessage(parentChannelID, eventMessageID)
	if fetchErr != nil || message == nil || message.Thread == nil {
		if err != nil {
			return "", fmt.Errorf("thread exists but could not be fetched: %w", err)
		}
		return "", fmt.Errorf("thread exists but could not be fetched")
	}
	return message.Thread.ID, nil
}

func threadCreationAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	// Discord API error code 160004 indicates a thread already exists for this message.
	// We keep string matching as a fallback because discordgo may surface typed or plain errors.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "thread already exists") || strings.Contains(msg, "160004")
}

func shortRoundID(roundID sharedtypes.RoundID) string {
	id := roundID.String()
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func roundThreadUploadInstructions(roundID sharedtypes.RoundID) string {
	return fmt.Sprintf(
		"Round is live. Upload scorecards in this thread.\n"+
			"- Attach a `.csv` or `.xlsx` file\n"+
			"- Or paste a `https://udisc.com/...` URL\n"+
			"- Partial uploads are allowed and will merge into the round\n"+
			"Round ID: `%s`",
		roundID.String(),
	)
}

func (m *scorecardUploadManager) threadUploadContext(channelID string) (*threadUploadContext, bool) {
	m.threadMutex.RLock()
	defer m.threadMutex.RUnlock()

	ctx, ok := m.threadContexts[channelID]
	if !ok || ctx == nil {
		return nil, false
	}

	out := *ctx
	return &out, true
}
