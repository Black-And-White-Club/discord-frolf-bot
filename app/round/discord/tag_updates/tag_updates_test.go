package tagupdates

import (
	"context"
	"log/slog"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	gc_mocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	helpermocks "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	discordmetricsmocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

func TestNewTagUpdateManager_Constructs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sess := discordmocks.NewMockSession(ctrl)
	var bus eventbus.EventBus
	logger := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	helper := helpermocks.NewMockHelpers(ctrl)
	cfg := &config.Config{}
	tracer := noop.NewTracerProvider().Tracer("test")
	metrics := discordmetricsmocks.NewMockDiscordMetrics(ctrl)
	resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)

	mgr := NewTagUpdateManager(sess, bus, logger, helper, cfg, tracer, metrics, resolver)
	if mgr == nil {
		t.Fatalf("expected non-nil manager")
	}
}

// implements io.Writer to discard logs in tests
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { return len(p), nil }

func Test_format_and_parseParticipantLine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mgr := &tagUpdateManager{}

	tn := sharedtypes.TagNumber(7)
	line := mgr.formatParticipantLine(sharedtypes.DiscordID("u1"), &tn)
	if line != "<@u1> Tag: 7" {
		t.Fatalf("unexpected format: %q", line)
	}
	uid, parsedTag, ok := mgr.parseParticipantLine(context.Background(), line)
	if !ok || string(uid) != "u1" || parsedTag == nil || int(*parsedTag) != 7 {
		t.Fatalf("failed to parse formatted line: uid=%v tag=%v ok=%v", uid, parsedTag, ok)
	}
	// no tag variant
	line2 := mgr.formatParticipantLine(sharedtypes.DiscordID("u2"), nil)
	if line2 != "<@u2>" {
		t.Fatalf("unexpected no-tag format: %q", line2)
	}
}

func TestUpdateTagsInEmbed_BasicFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sess := discordmocks.NewMockSession(ctrl)
	resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
	mgr := &tagUpdateManager{session: sess, guildConfigResolver: resolver, operationWrapper: func(ctx context.Context, op string, fn func(context.Context) (TagUpdateOperationResult, error)) (TagUpdateOperationResult, error) {
		return fn(ctx)
	}}

	// existing message with embed and two participant fields
	msg := &discordgo.Message{Embeds: []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{Name: "Accepted", Value: "<@u1> Tag: 3\n<@u2>"}}}}}
	sess.EXPECT().ChannelMessage("c1", "m1").Return(msg, nil)
	// expect edit with updated embed
	sess.EXPECT().ChannelMessageEditComplex(gomock.Any()).DoAndReturn(func(edit *discordgo.MessageEdit, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
		if edit.Embeds == nil || len(*edit.Embeds) != 1 || len((*edit.Embeds)[0].Fields) != 1 {
			t.Fatalf("unexpected embeds in edit")
		}
		val := (*edit.Embeds)[0].Fields[0].Value
		if val != "<@u1> Tag: 5\n<@u2>" {
			t.Fatalf("unexpected updated value: %q", val)
		}
		return &discordgo.Message{ID: "updated"}, nil
	})

	tn := sharedtypes.TagNumber(5)
	res, err := mgr.UpdateTagsInEmbed(context.Background(), "c1", "m1", map[sharedtypes.DiscordID]*sharedtypes.TagNumber{"u1": &tn})
	if err != nil || res.Error != nil || res.Success == nil {
		t.Fatalf("expected success, got res=%v err=%v", res, err)
	}
}

func TestUpdateDiscordEmbedsWithTagChanges_Variants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sess := discordmocks.NewMockSession(ctrl)
	resolver := gc_mocks.NewMockGuildConfigResolver(ctrl)
	mgr := &tagUpdateManager{session: sess, guildConfigResolver: resolver, operationWrapper: func(ctx context.Context, op string, fn func(context.Context) (TagUpdateOperationResult, error)) (TagUpdateOperationResult, error) {
		return fn(ctx)
	}, logger: slog.Default()}

	// success path: resolver returns EventChannelID and UpdateTagsInEmbed gets called (mock via ChannelMessage)
	resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g1").Return(&storage.GuildConfig{EventChannelID: "c1"}, nil)
	sess.EXPECT().ChannelMessage("c1", "m1").Return(&discordgo.Message{Embeds: []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{Name: "Accepted", Value: "<@u1>"}}}}}, nil)
	sess.EXPECT().ChannelMessageEditComplex(gomock.Any()).Return(&discordgo.Message{ID: "ok"}, nil)

	payload := roundevents.TagsUpdatedForScheduledRoundsPayloadV1{UpdatedRounds: []roundevents.RoundUpdateInfoV1{{GuildID: "g1", EventMessageID: "m1", State: roundtypes.RoundStateUpcoming}}}
	tn := sharedtypes.TagNumber(9)
	if res, err := mgr.UpdateDiscordEmbedsWithTagChanges(context.Background(), payload, map[sharedtypes.DiscordID]*sharedtypes.TagNumber{"u1": &tn}); err != nil || res.Error != nil {
		t.Fatalf("expected success, got res=%v err=%v", res, err)
	}

	// resolver error
	resolver.EXPECT().GetGuildConfigWithContext(gomock.Any(), "g2").Return(nil, context.DeadlineExceeded)
	payload2 := roundevents.TagsUpdatedForScheduledRoundsPayloadV1{UpdatedRounds: []roundevents.RoundUpdateInfoV1{{GuildID: "g2", EventMessageID: "m2", State: roundtypes.RoundStateUpcoming}}}
	if res, err := mgr.UpdateDiscordEmbedsWithTagChanges(context.Background(), payload2, map[sharedtypes.DiscordID]*sharedtypes.TagNumber{}); err != nil && res.Error == nil {
		t.Fatalf("should not return underlying error; expected wrapped in result only")
	}
}
