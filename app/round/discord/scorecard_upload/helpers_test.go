package scorecardupload

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_scorecardUploadManager_sendUploadConfirmation_RespondError(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	inter := &discordgo.Interaction{ID: "i1"}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		return errors.New("boom")
	}

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := m.sendUploadConfirmation(context.Background(), fakeSession, inter, "import-1"); err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_sendUploadError_RespondError(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	inter := &discordgo.Interaction{ID: "i1"}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		return errors.New("boom")
	}

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := m.sendUploadError(context.Background(), fakeSession, inter, "nope"); err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_downloadAttachment_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(server.Close)

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if _, err := m.downloadAttachment(context.Background(), server.URL); err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_downloadAttachment_BadURL(t *testing.T) {
	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if _, err := m.downloadAttachment(context.Background(), "://bad-url"); err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_downloadAttachment_TooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, maxAttachmentBytes+1))
	}))
	t.Cleanup(server.Close)

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	_, err := m.downloadAttachment(context.Background(), server.URL)
	if !errors.Is(err, errAttachmentTooLarge) {
		t.Fatalf("expected errAttachmentTooLarge, got %v", err)
	}
}

func Test_scorecardUploadManager_sendFileUploadPrompt_StoresPendingAndResponds(t *testing.T) {
	// This overlaps with existing coverage, but hits the prompt + pending-store path directly.
	fakeSession := discord.NewFakeSession()
	inter := &discordgo.Interaction{ID: "i1", ChannelID: "c1", GuildID: "g1", Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}}}
	rID := sharedtypes.RoundID(uuid.New())

	fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
		return &discordgo.Channel{ID: "dm-channel-id"}, nil
	}

	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{}, nil
	}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		return nil
	}

	m := &scorecardUploadManager{
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		pendingUploads: make(map[string]*pendingUpload),
	}

	if err := m.sendFileUploadPrompt(context.Background(), fakeSession, inter, rID, "notes", "msg-123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify pending upload was stored with correct fields
	key := "u1:dm-channel-id"
	pending, exists := m.pendingUploads[key]
	if !exists {
		t.Fatalf("pending upload not stored")
	}
	if pending.EventMessageID != "msg-123" {
		t.Fatalf("EventMessageID not stored correctly: got %q, want %q", pending.EventMessageID, "msg-123")
	}
}
