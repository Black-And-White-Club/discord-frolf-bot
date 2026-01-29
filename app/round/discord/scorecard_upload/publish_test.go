package scorecardupload

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func Test_scorecardUploadManager_publishScorecardURLEvent_SetsMetadataAndPayload(t *testing.T) {
	fakePublisher := &testutils.FakeEventBus{}

	guildID := sharedtypes.GuildID("g1")
	roundID := sharedtypes.RoundID(uuid.New())
	userID := sharedtypes.DiscordID("u1")
	channelID := "c1"
	messageID := "m1"
	url := "https://udisc.com/scorecard?id=123"
	notes := "notes"

	fakePublisher.PublishFunc = func(topic string, msgs ...*message.Message) error {
		if topic != roundevents.ScorecardURLRequestedV1 {
			t.Fatalf("topic mismatch: got %q", topic)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		msg := msgs[0]
		if msg.Metadata.Get("event_name") != roundevents.ScorecardURLRequestedV1 {
			t.Fatalf("expected event_name metadata")
		}
		if msg.Metadata.Get("domain") != "scorecard" {
			t.Fatalf("expected domain metadata")
		}
		if msg.Metadata.Get("guild_id") != string(guildID) {
			t.Fatalf("expected guild_id metadata")
		}
		if msg.Metadata.Get("import_id") == "" {
			t.Fatalf("expected import_id metadata")
		}

		var payload roundevents.ScorecardURLRequestedPayloadV1
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if payload.ImportID == "" {
			t.Fatalf("expected ImportID")
		}
		if payload.GuildID != guildID {
			t.Fatalf("guild mismatch: got %q", payload.GuildID)
		}
		if payload.RoundID != roundID {
			t.Fatalf("round mismatch: got %q", payload.RoundID)
		}
		if payload.UserID != userID {
			t.Fatalf("user mismatch: got %q", payload.UserID)
		}
		if payload.ChannelID != channelID {
			t.Fatalf("channel mismatch: got %q", payload.ChannelID)
		}
		if payload.MessageID != messageID {
			t.Fatalf("message mismatch: got %q", payload.MessageID)
		}
		if payload.UDiscURL != url {
			t.Fatalf("url mismatch: got %q", payload.UDiscURL)
		}
		if payload.Notes != notes {
			t.Fatalf("notes mismatch: got %q", payload.Notes)
		}
		if payload.Timestamp.IsZero() {
			t.Fatalf("expected timestamp")
		}

		// Ensure import_id metadata matches payload
		if msg.Metadata.Get("import_id") != payload.ImportID {
			t.Fatalf("import_id metadata mismatch")
		}
		return nil
	}

	m := &scorecardUploadManager{publisher: fakePublisher, logger: discardLogger()}
	importID, err := m.publishScorecardURLEvent(context.Background(), guildID, roundID, userID, channelID, messageID, url, notes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if importID == "" {
		t.Fatalf("expected importID")
	}
}

func Test_scorecardUploadManager_publishScorecardUploadEvent_PublishFailure_ReturnsError(t *testing.T) {
	fakePublisher := &testutils.FakeEventBus{}
	fakePublisher.PublishFunc = func(topic string, msgs ...*message.Message) error {
		return errors.New("publish failed")
	}

	m := &scorecardUploadManager{publisher: fakePublisher, logger: discardLogger()}
	importID, err := m.publishScorecardUploadEvent(
		context.Background(),
		sharedtypes.GuildID("g1"),
		sharedtypes.RoundID(uuid.New()),
		sharedtypes.DiscordID("u1"),
		"c1",
		"m1",
		[]byte("data"),
		"http://example.com/scorecard.csv",
		"scorecard.csv",
		"",
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if importID != "" {
		t.Fatalf("expected empty importID on error, got %q", importID)
	}
}
