package scorecardupload

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_scorecardUploadManager_sendUploadConfirmation_RespondError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	sess := discordmocks.NewMockSession(ctrl)
	inter := &discordgo.Interaction{ID: "i1"}

	sess.EXPECT().InteractionRespond(gomock.Eq(inter), gomock.Any()).Return(errors.New("boom")).Times(1)

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := m.sendUploadConfirmation(context.Background(), sess, inter, "import-1"); err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_sendUploadError_RespondError(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	sess := discordmocks.NewMockSession(ctrl)
	inter := &discordgo.Interaction{ID: "i1"}

	sess.EXPECT().InteractionRespond(gomock.Eq(inter), gomock.Any()).Return(errors.New("boom")).Times(1)

	m := &scorecardUploadManager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := m.sendUploadError(context.Background(), sess, inter, "nope"); err == nil {
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

func Test_scorecardUploadManager_sendFileUploadPrompt_StoresPendingAndResponds(t *testing.T) {
	// This overlaps with existing coverage, but hits the prompt + pending-store path directly.
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	sess := discordmocks.NewMockSession(ctrl)
	inter := &discordgo.Interaction{ID: "i1", ChannelID: "c1", Member: &discordgo.Member{User: &discordgo.User{ID: "u1"}}}
	rID := sharedtypes.RoundID(uuid.New())

	sess.EXPECT().InteractionRespond(gomock.Eq(inter), gomock.Any()).Return(nil).Times(1)

	m := &scorecardUploadManager{
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		pendingUploads: make(map[string]*pendingUpload),
	}

	if err := m.sendFileUploadPrompt(context.Background(), sess, inter, rID, "notes"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
