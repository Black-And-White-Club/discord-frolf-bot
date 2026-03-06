package scorecardupload

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_scorecardUploadManager_EnsureRoundThreadInstructions_ConcurrentCallsPostOnce(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			fakeSession := discord.NewFakeSession()

			threadID := "thread-id"
			parentChannelID := "parent-channel-id"
			eventMessageID := "event-message-id"
			roundID := sharedtypes.RoundID(uuid.New())
			guildID := sharedtypes.GuildID("guild-id")

			fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
				if channelID != parentChannelID {
					t.Fatalf("channel_id mismatch: got %q want %q", channelID, parentChannelID)
				}
				if messageID != eventMessageID {
					t.Fatalf("message_id mismatch: got %q want %q", messageID, eventMessageID)
				}
				return &discordgo.Message{ID: eventMessageID, Thread: &discordgo.Channel{ID: threadID}}, nil
			}

			var sendCalls atomic.Int32
			firstSendStarted := make(chan struct{})
			releaseFirstSend := make(chan struct{})
			fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
				if channelID != threadID {
					t.Fatalf("thread_id mismatch: got %q want %q", channelID, threadID)
				}
				count := sendCalls.Add(1)
				if count == 1 {
					close(firstSendStarted)
					<-releaseFirstSend
				}
				return &discordgo.Message{ID: "sent"}, nil
			}

			m := &scorecardUploadManager{
				session:        fakeSession,
				logger:         discardLogger(),
				threadContexts: make(map[string]*threadUploadContext),
			}

			firstErrCh := make(chan error, 1)
			go func() {
				firstErrCh <- m.EnsureRoundThreadInstructions(context.Background(), guildID, roundID, parentChannelID, eventMessageID)
			}()

			select {
			case <-firstSendStarted:
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for first instruction send")
			}

			secondErr := m.EnsureRoundThreadInstructions(context.Background(), guildID, roundID, parentChannelID, eventMessageID)
			if secondErr != nil {
				t.Fatalf("unexpected error on second call: %v", secondErr)
			}
			if got := sendCalls.Load(); got != 1 {
				t.Fatalf("expected only one in-flight send, got %d", got)
			}

			close(releaseFirstSend)

			select {
			case firstErr := <-firstErrCh:
				if firstErr != nil {
					t.Fatalf("unexpected error on first call: %v", firstErr)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("timed out waiting for first call to complete")
			}

			if got := sendCalls.Load(); got != 1 {
				t.Fatalf("expected exactly one send total, got %d", got)
			}

			threadCtx, ok := m.threadUploadContext(threadID)
			if !ok {
				t.Fatalf("expected stored thread context")
			}
			if !threadCtx.InstructionsPosted {
				t.Fatalf("expected InstructionsPosted=true")
			}
			if threadCtx.InstructionsPosting {
				t.Fatalf("expected InstructionsPosting=false")
			}
		})
	}
}
