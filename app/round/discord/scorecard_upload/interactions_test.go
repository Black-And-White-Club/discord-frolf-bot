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
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_MissingRoundID_ReturnsError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

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
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			"user-id:channel-id": {
				RoundID:   sharedtypes.RoundID(uuid.New()),
				GuildID:   sharedtypes.GuildID("guild-id"),
				CreatedAt: time.Now(),
			},
		},
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
	fakeSession := discord.NewFakeSession()

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
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			"user-id:channel-id": {
				RoundID:   sharedtypes.RoundID(uuid.New()),
				GuildID:   sharedtypes.GuildID("guild-id"),
				CreatedAt: time.Now(),
			},
		},
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err == nil {
		t.Fatalf("expected error")
	}
	if res.Error == nil {
		t.Fatalf("expected result error")
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_InvalidUDiscURL_ReturnsValidationError(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

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
					&discordgo.TextInput{CustomID: "udisc_url_input", Value: "https://example.com/scorecard.csv"},
				}},
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "notes_input", Value: ""},
				}},
			},
		},
	}}

	published := false
	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		published = true
		return nil
	}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		if resp == nil || resp.Data == nil {
			t.Fatalf("expected non-nil response data")
		}
		if resp.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
			t.Fatalf("expected ephemeral response")
		}
		if !strings.Contains(resp.Data.Content, "valid HTTPS URL on udisc.com") {
			t.Fatalf("unexpected validation content: %q", resp.Data.Content)
		}
		return nil
	}

	m := &scorecardUploadManager{
		session:   fakeSession,
		publisher: fakePublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	res, err := m.HandleScorecardUploadModalSubmit(context.Background(), i)
	if err != nil {
		t.Fatalf("expected nil error return, got %v", err)
	}
	if res.Error == nil {
		t.Fatalf("expected validation error in result")
	}
	if published {
		t.Fatalf("expected no publish call for invalid URL")
	}
}

func Test_scorecardUploadManager_HandleScorecardUploadModalSubmit_URLFlow_PublishError_Propagates(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

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

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		return fmt.Errorf("publish failed")
	}

	// Publish failures should respond ephemerally with an error message (best-effort)
	// and still return the publish error to the caller.
	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
	}
	m := &scorecardUploadManager{
		session:   fakeSession,
		publisher: fakePublisher,
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
	fakeSession := discord.NewFakeSession()

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

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
		return fmt.Errorf("respond failed")
	}

	m := &scorecardUploadManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			"user-id:channel-id": {
				RoundID:   sharedtypes.RoundID(uuid.New()),
				GuildID:   sharedtypes.GuildID("guild-id"),
				CreatedAt: time.Now(),
			},
		},
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
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

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
		session:   fakeSession,
		publisher: fakePublisher,
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

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		if topic != roundevents.ScorecardUploadedV1 {
			t.Fatalf("unexpected topic: %q", topic)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message")
		}
		var payload roundevents.ScorecardUploadedPayloadV1
		if err := json.Unmarshal(messages[0].Payload, &payload); err != nil {
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
	}

	fakeSession.ChannelMessageSendFunc = func(cID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if cID != channelID {
			t.Fatalf("channel_id mismatch: got %q want %q", cID, channelID)
		}
		if !strings.Contains(content, "Scorecard uploaded successfully") {
			t.Fatalf("unexpected confirmation content: %q", content)
		}
		if !strings.Contains(content, "Import ID") {
			t.Fatalf("expected import id in confirmation: %q", content)
		}
		return &discordgo.Message{ID: "confirmation"}, nil
	}

	m.HandleFileUploadMessage(fakeSession, msg)

	// Pending should be consumed
	key := fmt.Sprintf("%s:%s", userID, channelID)
	m.pendingMutex.RLock()
	_, stillThere := m.pendingUploads[key]
	m.pendingMutex.RUnlock()
	if stillThere {
		t.Fatalf("expected pending upload to be consumed")
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_LargeAttachment_PublishesURLReferenceWithoutDownload(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

	downloadRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadRequests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should-not-download"))
	}))
	defer server.Close()

	userID := "user-id"
	guildID := "guild-id"
	channelID := "channel-id"
	messageID := "message-id"
	roundUUID := uuid.New()
	fileName := "scorecard.csv"

	msg := &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        messageID,
		GuildID:   guildID,
		ChannelID: channelID,
		Author:    &discordgo.User{ID: userID, Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{
				Filename: fileName,
				URL:      server.URL,
				Size:     maxInlineFileDataBytes + 1,
			},
		},
	}}

	m := &scorecardUploadManager{
		session:   fakeSession,
		publisher: fakePublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			fmt.Sprintf("%s:%s", userID, channelID): {
				RoundID:        sharedtypes.RoundID(roundUUID),
				GuildID:        sharedtypes.GuildID(guildID),
				EventMessageID: messageID,
			},
		},
	}

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		if topic != roundevents.ScorecardUploadedV1 {
			t.Fatalf("unexpected topic: %q", topic)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message")
		}

		var payload roundevents.ScorecardUploadedPayloadV1
		if err := json.Unmarshal(messages[0].Payload, &payload); err != nil {
			t.Fatalf("failed to unmarshal payload: %v", err)
		}
		if len(payload.FileData) != 0 {
			t.Fatalf("expected no inline file data for large attachment, got %d bytes", len(payload.FileData))
		}
		if payload.FileURL != server.URL {
			t.Fatalf("file URL mismatch: got %q want %q", payload.FileURL, server.URL)
		}
		return nil
	}

	fakeSession.ChannelMessageSendFunc = func(cID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		return &discordgo.Message{ID: "confirmation"}, nil
	}

	m.HandleFileUploadMessage(fakeSession, msg)

	if downloadRequests != 0 {
		t.Fatalf("expected attachment download to be skipped, got %d request(s)", downloadRequests)
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_DownloadFailure_SendsError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

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

	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if !strings.Contains(content, "Failed to download file") {
			t.Fatalf("unexpected error content: %q", content)
		}
		return &discordgo.Message{ID: "err"}, nil
	}

	m := &scorecardUploadManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			"user-id:channel-id": {
				RoundID:   sharedtypes.RoundID(uuid.New()),
				GuildID:   sharedtypes.GuildID("guild-id"),
				CreatedAt: time.Now(),
			},
		},
	}

	m.HandleFileUploadMessage(fakeSession, msg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_FileTooLarge_SendsError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	big := make([]byte, maxAttachmentBytes+1)
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

	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if !strings.Contains(content, "File too large") {
			t.Fatalf("unexpected error content: %q", content)
		}
		return &discordgo.Message{ID: "err"}, nil
	}

	m := &scorecardUploadManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			"user-id:channel-id": {
				RoundID:   sharedtypes.RoundID(uuid.New()),
				GuildID:   sharedtypes.GuildID("guild-id"),
				CreatedAt: time.Now(),
			},
		},
	}

	m.HandleFileUploadMessage(fakeSession, msg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_PublishError_SendsErrorAndConsumesPending(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

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

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		return fmt.Errorf("publish failed")
	}

	fakeSession.ChannelMessageSendFunc = func(cID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if !strings.Contains(content, "Failed to process scorecard upload") {
			t.Fatalf("unexpected error content: %q", content)
		}
		return &discordgo.Message{ID: "err"}, nil
	}

	m := &scorecardUploadManager{
		session:   fakeSession,
		publisher: fakePublisher,
		logger:    discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: map[string]*pendingUpload{
			key: {RoundID: sharedtypes.RoundID(uuid.New()), Notes: ""},
		},
	}

	m.HandleFileUploadMessage(fakeSession, msg)

	m.pendingMutex.RLock()
	_, stillThere := m.pendingUploads[key]
	m.pendingMutex.RUnlock()
	if stillThere {
		t.Fatalf("expected pending upload to be consumed")
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_BotOrNoAttachments_Ignored(t *testing.T) {
	fakeSession := discord.NewFakeSession()

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
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	// No expectations; these should return early.
	m.HandleFileUploadMessage(fakeSession, botMsg)
	m.HandleFileUploadMessage(fakeSession, noAttachMsg)
}

func Test_scorecardUploadManager_HandleFileUploadMessage_InvalidPayloadsIgnored(t *testing.T) {
	fakeSession := discord.NewFakeSession()
	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		t.Fatalf("unexpected ChannelMessageSend call for invalid payload path")
		return nil, nil
	}

	m := &scorecardUploadManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	m.HandleFileUploadMessage(fakeSession, nil)
	m.HandleFileUploadMessage(fakeSession, &discordgo.MessageCreate{})
	m.HandleFileUploadMessage(fakeSession, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-no-author",
		ChannelID: "channel-id",
	}})
	m.HandleFileUploadMessage(nil, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-no-session",
		ChannelID: "channel-id",
		Author:    &discordgo.User{ID: "user-id", Bot: false},
	}})
}

func Test_scorecardUploadManager_HandleFileUploadMessage_NoPending_SendsError(t *testing.T) {
	fakeSession := discord.NewFakeSession()

	downloadRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadRequests++
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

	fakeSession.ChannelMessageSendFunc = func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if !strings.Contains(content, "No pending scorecard upload found") {
			t.Fatalf("unexpected error content: %q", content)
		}
		return &discordgo.Message{ID: "err"}, nil
	}

	m := &scorecardUploadManager{
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	m.HandleFileUploadMessage(fakeSession, msg)

	if downloadRequests != 0 {
		t.Fatalf("expected no attachment download when no pending upload exists, got %d request(s)", downloadRequests)
	}
}

func Test_scorecardUploadManager_HandleFileUploadMessage_NonScorecardAttachment_Ignored(t *testing.T) {
	fakeSession := discord.NewFakeSession()

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
		session: fakeSession,
		logger:  discardLogger(),
		operationWrapper: func(ctx context.Context, _ string, fn func(context.Context) (ScorecardUploadOperationResult, error)) (ScorecardUploadOperationResult, error) {
			return fn(ctx)
		},
		pendingUploads: make(map[string]*pendingUpload),
	}

	// No expectations: should return early without sending anything.
	m.HandleFileUploadMessage(fakeSession, msg)
}

func Test_scorecardUploadManager_allowUploadIngress_EnforcesPerUserLimit(t *testing.T) {
	m := &scorecardUploadManager{}

	for i := 0; i < maxUploadsPerMinutePerUserID; i++ {
		if !m.allowUploadIngress("guild-id", "user-id") {
			t.Fatalf("unexpected rate-limit rejection at iteration %d", i)
		}
	}

	if m.allowUploadIngress("guild-id", "user-id") {
		t.Fatalf("expected final request to be rate limited")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func Test_scorecardUploadManager_HandleScorecardUploadButton_SendsModal(t *testing.T) {
	fakeSession := discord.NewFakeSession()

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

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
	}

	m := &scorecardUploadManager{
		session: fakeSession,
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
	fakeSession := discord.NewFakeSession()
	fakePublisher := &testutils.FakeEventBus{}

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

	fakePublisher.PublishFunc = func(topic string, messages ...*message.Message) error {
		if topic != roundevents.ScorecardURLRequestedV1 {
			t.Fatalf("unexpected topic: %q", topic)
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message")
		}
		var payload roundevents.ScorecardURLRequestedPayloadV1
		if err := json.Unmarshal(messages[0].Payload, &payload); err != nil {
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
	}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
	}

	m := &scorecardUploadManager{
		session:   fakeSession,
		publisher: fakePublisher,
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
	fakeSession := discord.NewFakeSession()

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

	fakeSession.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
		if recipientID != userID {
			t.Fatalf("expected DM channel creation for user %q, got %q", userID, recipientID)
		}
		return &discordgo.Channel{ID: "dm-channel-id"}, nil
	}

	fakeSession.ChannelMessageSendFunc = func(cID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
		if cID != "dm-channel-id" {
			t.Fatalf("expected DM message to channel %q, got %q", "dm-channel-id", cID)
		}
		if !strings.Contains(content, "Please upload your scorecard file") {
			t.Fatalf("unexpected DM content: %q", content)
		}
		return &discordgo.Message{}, nil
	}

	fakeSession.InteractionRespondFunc = func(i *discordgo.Interaction, resp *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
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
	}

	m := &scorecardUploadManager{
		session: fakeSession,
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
