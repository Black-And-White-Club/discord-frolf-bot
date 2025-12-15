package scorecardupload

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	eventbusmocks "github.com/Black-And-White-Club/frolf-bot-shared/eventbus/mocks"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_MissingRoundID_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-id",
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Member:    &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Message:   &discordgo.Message{ID: "message-id"},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: "scorecard_upload_modal", // missing "|<roundID>"
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: ""},
				}},
			},
		},
	}}

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Error == nil {
		t.Fatalf("expected result error")
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_InvalidRoundIDFormat_ReturnsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-id",
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Member:    &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Message:   &discordgo.Message{ID: "message-id"},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: "scorecard_upload_modal|not-a-uuid",
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: "https://udisc.com/scorecard?id=123"},
				}},
			},
		},
	}}

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Error == nil {
		t.Fatalf("expected result error")
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_URLFlow_PublishError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

	roundID := uuid.New()
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-id",
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Member:    &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Message:   &discordgo.Message{ID: "message-id"},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: fmt.Sprintf("scorecard_upload_modal|%s", roundID.String()),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: "https://udisc.com/scorecard?id=123"},
				}},
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "notes_input", Value: ""},
				}},
			},
		},
	}}

	mockPublisher.EXPECT().
		Publish(gomock.Eq(roundevents.ScorecardURLRequestedTopic), gomock.Any()).
		Return(fmt.Errorf("publish failed")).
		Times(1)

	// Publish failures should respond ephemerally with an error message (best-effort)
	// and still return the publish error to the caller.
	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
			if resp == nil || resp.Data == nil {
				t.Fatalf("expected non-nil response data")
			}
			if resp.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
				t.Fatalf("expected ephemeral response")
			}
			if !strings.Contains(resp.Data.Content, "Failed to upload scorecard from URL") {
				t.Fatalf("unexpected error content: %q", resp.Data.Content)
			}
			return nil
		}).
		Times(1)
	m := &scorecardUploadManager{
		session:   mockSession,
		publisher: mockPublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	_, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadButton_RespondError_Propagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	roundID := uuid.New().String()
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      "interaction-id",
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "guild-id",
		Member:  &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Data: discordgo.MessageComponentInteractionData{
			CustomID:      fmt.Sprintf("round_upload_scorecard|%s", roundID),
			ComponentType: discordgo.ButtonComponent,
		},
	}}

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		Return(fmt.Errorf("respond failed")).
		Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadButton(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Error == nil {
		t.Fatalf("expected result error")
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_PendingExists_PublishesAndConfirms(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

	serverData := []byte("hello scorecard")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(serverData)
	}))
	defer server.Close()

	userID := "user-id"
	guildID := "guild-id"
	channelID := "channel-id"
	messageID := "message-id"
	roundUUID := uuid.New()
	notes := "notes"
	fileName := "scorecard.csv"

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        messageID,
		GuildID:   guildID,
		ChannelID: channelID,
		Author:    &discordgo.User{ID: userID, Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: fileName, URL: server.URL},
		},
	}}

	m := &scorecardUploadManager{
		session:   mockSession,
		publisher: mockPublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			fmt.Sprintf("%s:%s", userID, channelID): {
				RoundID:        sharedtypes.RoundID(roundUUID),
				GuildID:        sharedtypes.GuildID(guildID),
				Notes:          notes,
				EventMessageID: messageID,
			},
		},
	}

	mockPublisher.EXPECT().
		Publish(gomock.Eq(roundevents.ScorecardUploadedTopic), gomock.Any()).
		DoAndReturn(func(_ string, msg *message.Message) error {
			var payload roundevents.ScorecardUploadedPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				t.Fatalf("failed to unmarshal payload: %v", err)
			}
			if string(payload.GuildID) != guildID {
				t.Fatalf("guild_id mismatch: got %q want %q", payload.GuildID, guildID)
			}
			if string(payload.UserID) != userID {
				t.Fatalf("user_id mismatch: got %q want %q", payload.UserID, userID)
			}
			if payload.ChannelID != channelID {
				t.Fatalf("channel_id mismatch: got %q want %q", payload.ChannelID, channelID)
			}
			if payload.MessageID != messageID {
				t.Fatalf("message_id mismatch: got %q want %q", payload.MessageID, messageID)
			}
			if payload.RoundID.String() != roundUUID.String() {
				t.Fatalf("round_id mismatch: got %q want %q", payload.RoundID.String(), roundUUID.String())
			}
			if payload.FileName != fileName {
				t.Fatalf("filename mismatch: got %q want %q", payload.FileName, fileName)
			}
			if string(payload.FileData) != string(serverData) {
				t.Fatalf("file data mismatch: got %q want %q", string(payload.FileData), string(serverData))
			}
			if payload.Notes != notes {
				t.Fatalf("notes mismatch: got %q want %q", payload.Notes, notes)
			}
			if payload.ImportID == "" {
				t.Fatalf("expected import_id to be set")
			}
			return nil
		}).
		Times(1)

	mockSession.EXPECT().
		ChannelMessageSend(gomock.Eq(channelID), gomock.Any()).
		DoAndReturn(func(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if !strings.Contains(content, "Scorecard uploaded successfully") {
				t.Fatalf("unexpected confirmation content: %q", content)
			}
			if !strings.Contains(content, "Import ID") {
				t.Fatalf("expected import id in confirmation: %q", content)
			}
			return &discordgo.Message{ID: "confirmation"}, nil
		}).
		Times(1)

	m.HandleFileUploadMessage(mockSession, msg)

	// Pending should be consumed
	key := fmt.Sprintf("%s:%s", userID, channelID)
	m.pendingMutex.RLock()
	_, stillThere := m.pendingUploads[key]
	m.pendingMutex.RUnlock()
	if stillThere {
		t.Fatalf("expected pending upload to be consumed")
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_DownloadFailure_SendsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.csv", URL: server.URL},
		},
	}}

	mockSession.EXPECT().
		ChannelMessageSend(gomock.Eq("channel-id"), gomock.Any()).
		DoAndReturn(func(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if !strings.Contains(content, "Failed to download file") {
				t.Fatalf("unexpected error content: %q", content)
			}
			return &discordgo.Message{ID: "err"}, nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	m.HandleFileUploadMessage(mockSession, msg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_FileTooLarge_SendsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	const maxFileSize = 10 * 1024 * 1024
	big := make([]byte, maxFileSize+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(big)
	}))
	defer server.Close()

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.xlsx", URL: server.URL},
		},
	}}

	mockSession.EXPECT().
		ChannelMessageSend(gomock.Eq("channel-id"), gomock.Any()).
		DoAndReturn(func(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if !strings.Contains(content, "File too large") {
				t.Fatalf("unexpected error content: %q", content)
			}
			return &discordgo.Message{ID: "err"}, nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	m.HandleFileUploadMessage(mockSession, msg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_PublishError_SendsErrorAndConsumesPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file"))
	}))
	defer server.Close()

	userID := "user-id"
	channelID := "channel-id"
	key := fmt.Sprintf("%s:%s", userID, channelID)

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: channelID,
		Author:    &discordgo.User{ID: userID, Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.csv", URL: server.URL},
		},
	}}

	mockPublisher.EXPECT().
		Publish(gomock.Eq(roundevents.ScorecardUploadedTopic), gomock.Any()).
		Return(fmt.Errorf("publish failed")).
		Times(1)

	mockSession.EXPECT().
		ChannelMessageSend(gomock.Eq(channelID), gomock.Any()).
		DoAndReturn(func(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if !strings.Contains(content, "Failed to process scorecard upload") {
				t.Fatalf("unexpected error content: %q", content)
			}
			return &discordgo.Message{ID: "err"}, nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session:   mockSession,
		publisher: mockPublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			key: {RoundID: sharedtypes.RoundID(uuid.New()), Notes: ""},
		},
	}

	m.HandleFileUploadMessage(mockSession, msg)

	m.pendingMutex.RLock()
	_, stillThere := m.pendingUploads[key]
	m.pendingMutex.RUnlock()
	if stillThere {
		t.Fatalf("expected pending upload to be consumed")
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_BotOrNoAttachments_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	botMsg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: true},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.csv", URL: "https://example.invalid"},
		},
	}}

	noAttachMsg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:          "message-id-2",
		GuildID:     "guild-id",
		ChannelID:   "channel-id",
		Author:      &discordgo.User{ID: "user-id", Bot: false},
		Attachments: nil,
	}}

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	// No expectations; these should return early.
	m.HandleFileUploadMessage(mockSession, botMsg)
	m.HandleFileUploadMessage(mockSession, noAttachMsg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_NoPending_SendsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file"))
	}))
	defer server.Close()

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "scorecard.xlsx", URL: server.URL},
		},
	}}

	mockSession.EXPECT().
		ChannelMessageSend(gomock.Eq("channel-id"), gomock.Any()).
		DoAndReturn(func(_ string, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if !strings.Contains(content, "No pending scorecard upload found") {
				t.Fatalf("unexpected error content: %q", content)
			}
			return &discordgo.Message{ID: "err"}, nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	m.HandleFileUploadMessage(mockSession, msg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_NonScorecardAttachment_Ignored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "message-id",
		GuildID:   "guild-id",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{Filename: "notes.txt", URL: "https://example.invalid/notes.txt"},
		},
	}}

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	// No expectations: should return early without sending anything.
	m.HandleFileUploadMessage(mockSession, msg)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Test_scorecardUploadManager_HandleScorecardUploadButton_SendsModal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	roundID := uuid.New().String()
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:      "interaction-id",
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "guild-id",
		Member:  &discordgo.Member{User: &discordgo.User{ID: "user-id"}},
		Data: discordgo.MessageComponentInteractionData{
			CustomID:      fmt.Sprintf("round_upload_scorecard|%s", roundID),
			ComponentType: discordgo.ButtonComponent,
		},
	}}

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
			if resp.Type != discordgo.InteractionResponseModal {
				t.Fatalf("expected modal response type, got %v", resp.Type)
			}
			if resp.Data == nil {
				t.Fatalf("expected response data")
			}
			if want := fmt.Sprintf("scorecard_upload_modal|%s", roundID); resp.Data.CustomID != want {
				t.Fatalf("customID mismatch: got %q want %q", resp.Data.CustomID, want)
			}
			if resp.Data.Title != "Upload Scorecard" {
				t.Fatalf("title mismatch: got %q", resp.Data.Title)
			}
			// Basic shape: 2 action rows with expected text input IDs.
			if len(resp.Data.Components) != 2 {
				t.Fatalf("expected 2 components (action rows), got %d", len(resp.Data.Components))
			}
			return nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadButton(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("expected nil result error, got %v", res.Error)
	}
	if res.Success != "modal_sent" {
		t.Fatalf("expected success %q, got %v", "modal_sent", res.Success)
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_URLFlow_PublishesAndConfirms(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockPublisher := eventbusmocks.NewMockEventBus(ctrl)

	roundID := uuid.New()
	guildID := "guild-id"
	userID := "user-id"
	channelID := "channel-id"
	messageID := "message-id"
	url := "https://udisc.com/scorecard?id=123"
	notes := " some notes "

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-id",
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   guildID,
		ChannelID: channelID,
		Member:    &discordgo.Member{User: &discordgo.User{ID: userID}},
		Message:   &discordgo.Message{ID: messageID},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: fmt.Sprintf("scorecard_upload_modal|%s", roundID.String()),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: url},
				}},
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "notes_input", Value: notes},
				}},
			},
		},
	}}

	mockPublisher.EXPECT().
		Publish(gomock.Eq(roundevents.ScorecardURLRequestedTopic), gomock.Any()).
		DoAndReturn(func(_ string, msg *message.Message) error {
			var payload roundevents.ScorecardURLRequestedPayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				t.Fatalf("failed to unmarshal payload: %v", err)
			}
			if string(payload.GuildID) != guildID {
				t.Fatalf("guild_id mismatch: got %q want %q", payload.GuildID, guildID)
			}
			if string(payload.UserID) != userID {
				t.Fatalf("user_id mismatch: got %q want %q", payload.UserID, userID)
			}
			if payload.ChannelID != channelID {
				t.Fatalf("channel_id mismatch: got %q want %q", payload.ChannelID, channelID)
			}
			if payload.MessageID != messageID {
				t.Fatalf("message_id mismatch: got %q want %q", payload.MessageID, messageID)
			}
			if payload.RoundID.String() != roundID.String() {
				t.Fatalf("round_id mismatch: got %q want %q", payload.RoundID.String(), roundID.String())
			}
			if payload.UDiscURL != url {
				t.Fatalf("udisc_url mismatch: got %q want %q", payload.UDiscURL, url)
			}
			if payload.Notes != notes {
				t.Fatalf("notes mismatch: got %q want %q", payload.Notes, notes)
			}
			if payload.ImportID == "" {
				t.Fatalf("expected import_id to be set")
			}
			return nil
		}).
		Times(1)

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
			if resp.Data == nil {
				t.Fatalf("expected response data")
			}
			if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
				t.Fatalf("expected ephemeral response")
			}
			if !strings.Contains(resp.Data.Content, "Scorecard import started") {
				t.Fatalf("expected confirmation message, got %q", resp.Data.Content)
			}
			if !strings.Contains(resp.Data.Content, "Import ID") {
				t.Fatalf("expected import id in message, got %q", resp.Data.Content)
			}
			return nil
		}).
		Times(1)

	m := &scorecardUploadManager{
		session:   mockSession,
		publisher: mockPublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("expected nil result error, got %v", res.Error)
	}
	if _, ok := res.Success.(string); !ok {
		t.Fatalf("expected success to be import id string, got %T", res.Success)
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_FileFlow_PromptsAndStoresPending(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)

	roundID := uuid.New()
	guildID := "guild-id"
	userID := "user-id"
	channelID := "channel-id"
	messageID := "message-id"
	notes := "my notes"

	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-id",
		Type:      discordgo.InteractionModalSubmit,
		GuildID:   guildID,
		ChannelID: channelID,
		Member:    &discordgo.Member{User: &discordgo.User{ID: userID}},
		Message:   &discordgo.Message{ID: messageID},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: fmt.Sprintf("scorecard_upload_modal|%s", roundID.String()),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: ""},
				}},
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "notes_input", Value: notes},
				}},
			},
		},
	}}

	mockSession.EXPECT().
		InteractionRespond(gomock.Eq(i.Interaction), gomock.Any()).
		DoAndReturn(func(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
			if resp.Data == nil {
				t.Fatalf("expected response data")
			}
			if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
				t.Fatalf("expected ephemeral prompt")
			}
			if !strings.Contains(resp.Data.Content, "I've sent you a DM") {
				t.Fatalf("unexpected prompt content: %q", resp.Data.Content)
			}
			return nil
		}).
		Times(1)

	// Expect DM channel creation
	dmChannel := &discordgo.Channel{ID: "dm-channel-id"}
	mockSession.EXPECT().UserChannelCreate(userID).Return(dmChannel, nil).Times(1)

	// Expect DM message
	mockSession.EXPECT().ChannelMessageSend("dm-channel-id", gomock.Any()).Return(&discordgo.Message{}, nil).Times(1)

	m := &scorecardUploadManager{
		session: mockSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Error != nil {
		t.Fatalf("expected nil result error, got %v", res.Error)
	}
	if res.Success != "file_upload_prompted" {
		t.Fatalf("expected success %q, got %v", "file_upload_prompted", res.Success)
	}

	key := fmt.Sprintf("%s:%s", userID, "dm-channel-id")
	m.pendingMutex.RLock()
	pending := m.pendingUploads[key]
	m.pendingMutex.RUnlock()
	if pending == nil {
		t.Fatalf("expected pending upload to be stored for key %q", key)
	}
	if pending.RoundID.String() != roundID.String() {
		t.Fatalf("pending round id mismatch: got %q want %q", pending.RoundID.String(), roundID.String())
	}
	if pending.GuildID != sharedtypes.GuildID(guildID) {
		t.Fatalf("pending guild id mismatch: got %q want %q", pending.GuildID, guildID)
	}
	if pending.Notes != notes {
		t.Fatalf("pending notes mismatch: got %q want %q", pending.Notes, notes)
	}
}
