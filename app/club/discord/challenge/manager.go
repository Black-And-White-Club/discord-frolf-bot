package challenge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	wmmessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	nc "github.com/nats-io/nats.go"
)

const (
	challengeAcceptPrefix   = "challenge_accept|"
	challengeDeclinePrefix  = "challenge_decline|"
	challengeSchedulePrefix = "challenge_schedule|"
	challengeListLimit      = 10
)

const challengeRoundAnnouncementTTL = 15 * time.Minute

type Manager interface {
	HandleChallengeCommand(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleAcceptButton(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleDeclineButton(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleScheduleButton(ctx context.Context, i *discordgo.InteractionCreate) error
	HandleChallengeFact(ctx context.Context, topic string, payload *clubevents.ChallengeFactPayloadV1) error
}

type manager struct {
	session             discord.Session
	publisher           eventbus.EventBus
	logger              *slog.Logger
	helper              utils.Helpers
	cfg                 *config.Config
	guildConfigResolver guildconfig.GuildConfigResolver
	metrics             discordmetrics.DiscordMetrics
	createRoundManager  createround.CreateRoundManager
	listChallenges      func(ctx context.Context, guildID string, statuses []clubtypes.ChallengeStatus) (*clubevents.ChallengeListResponsePayloadV1, error)
	getChallengeDetail  func(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error)
	roundAnnouncements  sync.Map
}

type challengeScheduleValidatorConfigurer interface {
	SetChallengeScheduleValidator(createround.ChallengeScheduleValidator)
}

func NewManager(
	session discord.Session,
	publisher eventbus.EventBus,
	logger *slog.Logger,
	helper utils.Helpers,
	cfg *config.Config,
	guildConfigResolver guildconfig.GuildConfigResolver,
	metrics discordmetrics.DiscordMetrics,
	createRoundManager createround.CreateRoundManager,
) Manager {
	mgr := &manager{
		session:             session,
		publisher:           publisher,
		logger:              logger,
		helper:              helper,
		cfg:                 cfg,
		guildConfigResolver: guildConfigResolver,
		metrics:             metrics,
		createRoundManager:  createRoundManager,
	}
	mgr.listChallenges = mgr.requestChallengeList
	mgr.getChallengeDetail = mgr.requestChallengeDetail
	if validatorConfigurer, ok := createRoundManager.(challengeScheduleValidatorConfigurer); ok {
		validatorConfigurer.SetChallengeScheduleValidator(mgr.validateScheduleRequest)
	}
	return mgr
}

func (m *manager) HandleChallengeCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "challenge")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "application_command")

	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return m.respondEphemeral(i, "Choose a challenge subcommand.")
	}

	sub := options[0]
	switch sub.Name {
	case "open":
		return m.handleOpen(ctx, i, userID, sub.Options)
	case "schedule":
		return m.handleSchedule(ctx, i, userID, sub.Options)
	case "withdraw":
		return m.handleWithdraw(ctx, i, userID, sub.Options)
	case "link":
		return m.handleLink(ctx, i, userID, sub.Options)
	case "unlink":
		return m.handleUnlink(ctx, i, userID, sub.Options)
	case "hide":
		return m.handleHide(ctx, i, userID, sub.Options)
	case "list":
		return m.handleList(ctx, i)
	default:
		return m.respondEphemeral(i, "Unknown challenge subcommand.")
	}
}

func (m *manager) HandleAcceptButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	return m.handleRespondButton(ctx, i, challengeAcceptPrefix, "accept")
}

func (m *manager) HandleDeclineButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	return m.handleRespondButton(ctx, i, challengeDeclinePrefix, "decline")
}

func (m *manager) HandleScheduleButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	return m.handleScheduleButton(ctx, i)
}

func (m *manager) HandleChallengeFact(ctx context.Context, topic string, payload *clubevents.ChallengeFactPayloadV1) error {
	if payload == nil {
		return nil
	}

	challenge := payload.Challenge
	guildID := ""
	if challenge.DiscordGuildID != nil {
		guildID = *challenge.DiscordGuildID
	}
	if guildID == "" && challenge.MessageBinding != nil {
		guildID = challenge.MessageBinding.GuildID
	}
	if guildID == "" {
		return nil
	}

	channelID := ""
	messageID := ""
	if challenge.MessageBinding != nil {
		channelID = challenge.MessageBinding.ChannelID
		messageID = challenge.MessageBinding.MessageID
	}
	if channelID == "" {
		resolvedChannelID, err := m.resolveChallengeChannel(ctx, guildID)
		if err != nil {
			m.logger.WarnContext(ctx, "unable to resolve challenge channel", attr.Error(err), attr.String("guild_id", guildID))
			return nil
		}
		channelID = resolvedChannelID
	}

	embed := buildChallengeEmbed(challenge)
	components := buildChallengeComponents(m.cfg, challenge)

	if messageID != "" {
		_, err := m.session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			ID:         messageID,
			Channel:    channelID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		})
		if err == nil {
			if isRoundLinkedTopic(topic) {
				return m.sendChallengeRoundAnnouncement(ctx, channelID, challenge)
			}
			return nil
		}
		m.logger.WarnContext(ctx, "failed to edit existing challenge card; posting replacement",
			attr.Error(err),
			attr.String("guild_id", guildID),
			attr.String("channel_id", channelID),
			attr.String("message_id", messageID),
			attr.String("challenge_id", challenge.ID),
		)
	}

	msg, err := m.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return err
	}

	if err := m.publishBind(ctx, challenge.ID, guildID, channelID, msg.ID); err != nil {
		m.logger.WarnContext(ctx, "failed to publish challenge message binding",
			attr.Error(err),
			attr.String("challenge_id", challenge.ID),
			attr.String("message_id", msg.ID),
		)
	}

	if isRoundLinkedTopic(topic) {
		return m.sendChallengeRoundAnnouncement(ctx, channelID, challenge)
	}

	return nil
}

func (m *manager) handleOpen(ctx context.Context, i *discordgo.InteractionCreate, actorID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	targetUser := commandUserOption(options, "user")
	if targetUser == nil {
		return m.respondEphemeral(i, "Choose a player to challenge.")
	}
	if targetUser.ID == actorID {
		return m.respondEphemeral(i, "You cannot challenge yourself.")
	}

	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	correlationID := uuid.NewString()
	payload := &clubevents.ChallengeOpenRequestedPayloadV1{
		GuildID:          i.GuildID,
		ActorExternalID:  actorID,
		TargetExternalID: targetUser.ID,
	}

	if err := m.publishRequest(ctx, clubevents.ChallengeOpenRequestedV1, payload, i.GuildID, correlationID); err != nil {
		return m.editDeferredResponse(i, "Unable to request the challenge right now.")
	}

	return m.editDeferredResponse(i, fmt.Sprintf("Challenge requested for <@%s>. A challenge card will be posted shortly.", targetUser.ID))
}

func (m *manager) handleSchedule(ctx context.Context, i *discordgo.InteractionCreate, _ string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	challengeID := commandStringOption(options, "challenge_id")
	if challengeID == "" {
		return m.respondEphemeral(i, "Provide a challenge ID to schedule.")
	}

	return m.startChallengeScheduling(ctx, i, challengeID)
}

func (m *manager) handleWithdraw(ctx context.Context, i *discordgo.InteractionCreate, actorID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	challengeID := commandStringOption(options, "challenge_id")
	if challengeID == "" {
		return m.respondEphemeral(i, "Provide a challenge ID to withdraw.")
	}

	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	challenge, err := m.loadChallengeDetail(ctx, i.GuildID, challengeID)
	if err != nil {
		return m.editDeferredResponse(i, err.Error())
	}
	if err := m.ensureParticipantOrPrivileged(ctx, i, challenge, "withdraw that challenge"); err != nil {
		return m.editDeferredResponse(i, err.Error())
	}

	return m.publishDeferredAction(
		ctx,
		i,
		clubevents.ChallengeWithdrawRequestedV1,
		&clubevents.ChallengeWithdrawRequestedPayloadV1{
			GuildID:         i.GuildID,
			ActorExternalID: actorID,
			ChallengeID:     challengeID,
		},
		"Challenge withdrawal requested. The card will update shortly.",
	)
}

func (m *manager) handleLink(ctx context.Context, i *discordgo.InteractionCreate, actorID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	challengeID := commandStringOption(options, "challenge_id")
	roundID := commandStringOption(options, "round_id")
	if challengeID == "" || roundID == "" {
		return m.respondEphemeral(i, "Provide both a challenge ID and a round ID.")
	}
	if _, err := uuid.Parse(roundID); err != nil {
		return m.respondEphemeral(i, "Provide a valid round ID.")
	}

	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	challenge, err := m.loadChallengeDetail(ctx, i.GuildID, challengeID)
	if err != nil {
		return m.editDeferredResponse(i, err.Error())
	}
	if err := m.ensureParticipantOrPrivileged(ctx, i, challenge, "link a round to that challenge"); err != nil {
		return m.editDeferredResponse(i, err.Error())
	}

	return m.publishDeferredAction(
		ctx,
		i,
		clubevents.ChallengeRoundLinkRequestedV1,
		&clubevents.ChallengeRoundLinkRequestedPayloadV1{
			GuildID:         i.GuildID,
			ActorExternalID: actorID,
			ChallengeID:     challengeID,
			RoundID:         roundID,
		},
		"Challenge round link requested. The card will update shortly.",
	)
}

func (m *manager) handleUnlink(ctx context.Context, i *discordgo.InteractionCreate, actorID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	challengeID := commandStringOption(options, "challenge_id")
	if challengeID == "" {
		return m.respondEphemeral(i, "Provide a challenge ID to unlink.")
	}

	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	challenge, err := m.loadChallengeDetail(ctx, i.GuildID, challengeID)
	if err != nil {
		return m.editDeferredResponse(i, err.Error())
	}
	if err := m.ensureParticipantOrPrivileged(ctx, i, challenge, "unlink that challenge round"); err != nil {
		return m.editDeferredResponse(i, err.Error())
	}

	return m.publishDeferredAction(
		ctx,
		i,
		clubevents.ChallengeRoundUnlinkRequestedV1,
		&clubevents.ChallengeRoundUnlinkRequestedPayloadV1{
			GuildID:         i.GuildID,
			ActorExternalID: actorID,
			ChallengeID:     challengeID,
		},
		"Challenge round unlink requested. The card will update shortly.",
	)
}

func (m *manager) handleHide(ctx context.Context, i *discordgo.InteractionCreate, actorID string, options []*discordgo.ApplicationCommandInteractionDataOption) error {
	challengeID := commandStringOption(options, "challenge_id")
	if challengeID == "" {
		return m.respondEphemeral(i, "Provide a challenge ID to hide.")
	}

	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	if _, err := m.loadChallengeDetail(ctx, i.GuildID, challengeID); err != nil {
		return m.editDeferredResponse(i, err.Error())
	}
	if err := m.ensurePrivileged(ctx, i, "hide that challenge"); err != nil {
		return m.editDeferredResponse(i, err.Error())
	}

	return m.publishDeferredAction(
		ctx,
		i,
		clubevents.ChallengeHideRequestedV1,
		&clubevents.ChallengeHideRequestedPayloadV1{
			GuildID:         i.GuildID,
			ActorExternalID: actorID,
			ChallengeID:     challengeID,
		},
		"Challenge hide requested. The card will update shortly.",
	)
}

func (m *manager) handleList(ctx context.Context, i *discordgo.InteractionCreate) error {
	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	response, err := m.listChallenges(ctx, i.GuildID, []clubtypes.ChallengeStatus{
		clubtypes.ChallengeStatusOpen,
		clubtypes.ChallengeStatusAccepted,
	})
	if err != nil {
		m.logger.WarnContext(ctx, "failed to list challenges", attr.Error(err), attr.String("guild_id", i.GuildID))
		return m.editDeferredResponse(i, "Unable to load active challenges right now.")
	}
	if response == nil {
		return m.editDeferredResponse(i, "Unable to load active challenges right now.")
	}

	return m.editDeferredResponse(i, formatChallengeListSummary(m.cfg, response))
}

func (m *manager) handleScheduleButton(ctx context.Context, i *discordgo.InteractionCreate) error {
	customID := i.MessageComponentData().CustomID
	challengeID := strings.TrimPrefix(customID, challengeSchedulePrefix)
	if challengeID == "" || challengeID == customID {
		return fmt.Errorf("invalid challenge schedule custom id: %s", customID)
	}

	return m.startChallengeScheduling(ctx, i, challengeID)
}

func (m *manager) startChallengeScheduling(ctx context.Context, i *discordgo.InteractionCreate, challengeID string) error {
	if m.createRoundManager == nil {
		return m.respondEphemeral(i, "Challenge scheduling is unavailable right now.")
	}

	result, err := m.createRoundManager.SendCreateRoundModal(
		createround.WithModalConfig(ctx, createround.ModalConfig{
			CustomID: createround.ChallengeScheduleModalCustomID(challengeID),
			Title:    "Schedule Challenge Round",
		}),
		i,
	)
	if err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (m *manager) handleRespondButton(ctx context.Context, i *discordgo.InteractionCreate, prefix, response string) error {
	userID, err := interactionUserID(i)
	if err != nil {
		return err
	}

	customID := i.MessageComponentData().CustomID
	challengeID := strings.TrimPrefix(customID, prefix)
	if challengeID == "" || challengeID == customID {
		return fmt.Errorf("invalid challenge button custom id: %s", customID)
	}

	if err := m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	}); err != nil {
		return err
	}

	payload := &clubevents.ChallengeRespondRequestedPayloadV1{
		GuildID:         i.GuildID,
		ActorExternalID: userID,
		ChallengeID:     challengeID,
		Response:        response,
	}

	if err := m.publishRequest(ctx, clubevents.ChallengeRespondRequestedV1, payload, i.GuildID, uuid.NewString()); err != nil {
		_, _ = m.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "Unable to process the challenge response right now.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return err
	}

	_, _ = m.session.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("Challenge %s requested. The card will update shortly.", response),
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	return nil
}

func (m *manager) publishImmediateAction(ctx context.Context, i *discordgo.InteractionCreate, topic string, payload any, successMessage string) error {
	if err := m.deferEphemeral(i); err != nil {
		return err
	}

	return m.publishDeferredAction(ctx, i, topic, payload, successMessage)
}

func (m *manager) publishDeferredAction(ctx context.Context, i *discordgo.InteractionCreate, topic string, payload any, successMessage string) error {
	if err := m.publishRequest(ctx, topic, payload, i.GuildID, uuid.NewString()); err != nil {
		return m.editDeferredResponse(i, "Unable to send that challenge request right now.")
	}

	return m.editDeferredResponse(i, successMessage)
}

func (m *manager) publishRequest(ctx context.Context, topic string, payload any, guildID, correlationID string) error {
	msg, err := m.helper.CreateNewMessage(payload, topic)
	if err != nil {
		return err
	}

	if msg.Metadata == nil {
		msg.Metadata = wmmessage.Metadata{}
	}
	if guildID != "" {
		msg.Metadata.Set("guild_id", guildID)
	}
	if correlationID != "" {
		msg.Metadata.Set("correlation_id", correlationID)
	}

	return m.publisher.Publish(topic, msg)
}

func (m *manager) publishBind(ctx context.Context, challengeID, guildID, channelID, messageID string) error {
	return m.publishRequest(ctx, clubevents.ChallengeMessageBindRequestedV1, &clubevents.ChallengeMessageBindRequestedPayloadV1{
		ChallengeID: challengeID,
		GuildID:     guildID,
		ChannelID:   channelID,
		MessageID:   messageID,
	}, guildID, uuid.NewString())
}

func (m *manager) sendChallengeRoundAnnouncement(ctx context.Context, channelID string, challenge clubtypes.ChallengeDetail) error {
	if challenge.LinkedRound == nil {
		return nil
	}

	announcementKey, ok := challengeRoundAnnouncementKey(challenge)
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	if !m.canSendRoundAnnouncement(announcementKey, now) {
		return nil
	}

	_, err := m.session.ChannelMessageSend(channelID, fmt.Sprintf(
		"Challenge Round Scheduled: %s challenged %s. Round ID: `%s`. Normal round rules apply to everyone who joins.",
		participantMention(challenge.ChallengerExternalID, challenge.ChallengerUserUUID),
		participantMention(challenge.DefenderExternalID, challenge.DefenderUserUUID),
		challenge.LinkedRound.RoundID,
	))
	if err != nil {
		m.logger.WarnContext(ctx, "failed to post challenge round announcement",
			attr.Error(err),
			attr.String("channel_id", channelID),
			attr.String("challenge_id", challenge.ID),
		)
		return nil
	}

	m.recordRoundAnnouncement(announcementKey, now)
	return nil
}

func (m *manager) resolveChallengeChannel(ctx context.Context, guildID string) (string, error) {
	if guildID == "" || m.guildConfigResolver == nil {
		return "", fmt.Errorf("guild config unavailable")
	}

	cfg, err := m.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID)
	if err != nil {
		return "", err
	}
	if cfg == nil || cfg.EventChannelID == "" {
		return "", fmt.Errorf("event channel unavailable")
	}
	return cfg.EventChannelID, nil
}

func (m *manager) deferEphemeral(i *discordgo.InteractionCreate) error {
	return m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (m *manager) editDeferredResponse(i *discordgo.InteractionCreate, content string) error {
	_, err := m.session.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
	return err
}

func (m *manager) respondEphemeral(i *discordgo.InteractionCreate, content string) error {
	return m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func interactionUserID(i *discordgo.InteractionCreate) (string, error) {
	switch {
	case i.Member != nil && i.Member.User != nil:
		return i.Member.User.ID, nil
	case i.User != nil:
		return i.User.ID, nil
	default:
		return "", fmt.Errorf("interaction user unavailable")
	}
}

func commandUserOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.User {
	for _, option := range options {
		if option.Name == name {
			return option.UserValue(nil)
		}
	}
	return nil
}

func commandStringOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if option.Name == name {
			return option.StringValue()
		}
	}
	return ""
}

func (m *manager) requestChallengeList(ctx context.Context, guildID string, statuses []clubtypes.ChallengeStatus) (*clubevents.ChallengeListResponsePayloadV1, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild id is required")
	}

	response := &clubevents.ChallengeListResponsePayloadV1{}
	if err := m.requestReplyJSON(
		ctx,
		challengeRequestSubject(clubevents.ChallengeListRequestV1, guildID),
		&clubevents.ChallengeListRequestPayloadV1{
			GuildID:  guildID,
			Statuses: statuses,
		},
		response,
	); err != nil {
		return nil, err
	}
	return response, nil
}

func (m *manager) requestChallengeDetail(ctx context.Context, guildID, challengeID string) (*clubevents.ChallengeDetailResponsePayloadV1, error) {
	if guildID == "" {
		return nil, fmt.Errorf("guild id is required")
	}
	if challengeID == "" {
		return nil, fmt.Errorf("challenge id is required")
	}

	response := &clubevents.ChallengeDetailResponsePayloadV1{}
	if err := m.requestReplyJSON(
		ctx,
		challengeRequestSubject(clubevents.ChallengeDetailRequestV1, guildID),
		&clubevents.ChallengeDetailRequestPayloadV1{
			GuildID:     guildID,
			ChallengeID: challengeID,
		},
		response,
	); err != nil {
		return nil, err
	}
	return response, nil
}

func (m *manager) requestReplyJSON(ctx context.Context, subject string, requestPayload any, responsePayload any) error {
	if m.publisher == nil {
		return fmt.Errorf("event bus unavailable")
	}

	natsConn := m.publisher.GetNATSConnection()
	if natsConn == nil {
		return fmt.Errorf("nats connection unavailable")
	}

	payloadBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	inbox := nc.NewInbox()
	subscription, err := natsConn.SubscribeSync(inbox)
	if err != nil {
		return fmt.Errorf("subscribe reply inbox: %w", err)
	}
	defer func() {
		_ = subscription.Unsubscribe()
	}()

	requestMsg := nc.NewMsg(subject)
	requestMsg.Data = payloadBytes
	requestMsg.Header = nc.Header{}
	requestMsg.Header.Set("reply_to", inbox)

	if err := natsConn.PublishMsg(requestMsg); err != nil {
		return fmt.Errorf("publish request: %w", err)
	}

	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	replyMsg, err := subscription.NextMsg(timeout)
	if err != nil {
		return fmt.Errorf("await reply: %w", err)
	}
	if err := json.Unmarshal(replyMsg.Data, responsePayload); err != nil {
		return fmt.Errorf("decode reply: %w", err)
	}

	return nil
}

func challengeRequestSubject(baseSubject, guildID string) string {
	return baseSubject + "." + guildID
}

func validateChallengeSchedule(challenge *clubtypes.ChallengeDetail) error {
	if challenge == nil {
		return fmt.Errorf("That challenge wasn't found.")
	}

	if challenge.Status != clubtypes.ChallengeStatusAccepted {
		return fmt.Errorf("Only accepted challenges can schedule a round.")
	}
	if challenge.LinkedRound != nil && challenge.LinkedRound.IsActive {
		return fmt.Errorf("That challenge already has an active linked round.")
	}

	return nil
}

func formatChallengeListSummary(cfg *config.Config, response *clubevents.ChallengeListResponsePayloadV1) string {
	if response == nil {
		return "No open or accepted challenges right now."
	}

	challenges := append([]clubtypes.ChallengeSummary(nil), response.Challenges...)
	sort.SliceStable(challenges, func(i, j int) bool {
		left := challenges[i]
		right := challenges[j]
		if left.Status != right.Status {
			return challengeStatusRank(left.Status) < challengeStatusRank(right.Status)
		}
		if !left.OpenedAt.Equal(right.OpenedAt) {
			return left.OpenedAt.After(right.OpenedAt)
		}
		return left.ID < right.ID
	})

	if len(challenges) == 0 {
		message := "No open or accepted challenges right now."
		if link := challengeBoardURL(cfg); link != "" {
			message += " Open in App: " + link
		}
		return message
	}

	lines := make([]string, 0, 7)
	lines = append(lines, fmt.Sprintf("Active challenges: %d", len(challenges)))

	limit := len(challenges)
	if limit > challengeListLimit {
		limit = challengeListLimit
	}
	for idx := 0; idx < limit; idx++ {
		challenge := challenges[idx]
		statusText := string(challenge.Status)
		if challenge.Status == clubtypes.ChallengeStatusAccepted {
			if challenge.LinkedRound != nil && challenge.LinkedRound.IsActive {
				statusText = fmt.Sprintf("accepted - round `%s`", challenge.LinkedRound.RoundID)
			} else {
				statusText = "accepted - awaiting round"
			}
		}

		lines = append(lines, fmt.Sprintf(
			"`%s` %s: %s vs %s",
			shortChallengeID(challenge.ID),
			statusText,
			participantMention(challenge.ChallengerExternalID, challenge.ChallengerUserUUID),
			participantMention(challenge.DefenderExternalID, challenge.DefenderUserUUID),
		))
	}

	if len(challenges) > limit {
		lines = append(lines, fmt.Sprintf("+%d more in app", len(challenges)-limit))
	}
	if link := challengeBoardURL(cfg); link != "" {
		lines = append(lines, "Open in App: "+link)
	}

	return strings.Join(lines, "\n")
}

func challengeStatusRank(status clubtypes.ChallengeStatus) int {
	switch status {
	case clubtypes.ChallengeStatusOpen:
		return 0
	case clubtypes.ChallengeStatusAccepted:
		return 1
	default:
		return 2
	}
}

func shortChallengeID(challengeID string) string {
	if len(challengeID) <= 8 {
		return challengeID
	}
	return challengeID[:8]
}

func challengeBoardURL(cfg *config.Config) string {
	baseURL := strings.TrimRight(openAppURL(cfg), "/")
	if baseURL == "" {
		return ""
	}
	return baseURL + "/challenges"
}

func isRoundLinkedTopic(topic string) bool {
	return strings.HasPrefix(topic, clubevents.ChallengeRoundLinkedV1)
}

func (m *manager) loadChallengeDetail(ctx context.Context, guildID, challengeID string) (*clubtypes.ChallengeDetail, error) {
	detailResponse, err := m.getChallengeDetail(ctx, guildID, challengeID)
	if err != nil {
		if m.logger != nil {
			m.logger.WarnContext(ctx, "failed to fetch challenge detail",
				attr.Error(err),
				attr.String("challenge_id", challengeID),
				attr.String("guild_id", guildID),
			)
		}
		return nil, fmt.Errorf("Unable to load that challenge right now.")
	}
	if detailResponse == nil || detailResponse.Challenge == nil {
		return nil, fmt.Errorf("That challenge wasn't found.")
	}

	return detailResponse.Challenge, nil
}

func (m *manager) validateScheduleRequest(ctx context.Context, i *discordgo.InteractionCreate, challengeID string) error {
	challenge, err := m.loadChallengeDetail(ctx, i.GuildID, challengeID)
	if err != nil {
		return err
	}
	if err := validateChallengeSchedule(challenge); err != nil {
		return err
	}
	return m.ensureParticipantOrPrivileged(ctx, i, challenge, "schedule that challenge")
}

func (m *manager) ensureParticipantOrPrivileged(ctx context.Context, i *discordgo.InteractionCreate, challenge *clubtypes.ChallengeDetail, action string) error {
	actorID, err := interactionUserID(i)
	if err != nil {
		return err
	}
	if isChallengeParticipant(challenge, actorID) {
		return nil
	}
	if allowed, roleErr := m.hasEditorOrAdminRole(ctx, i); roleErr == nil && allowed {
		return nil
	}

	return fmt.Errorf("Only challenge participants, editors, or admins can %s.", action)
}

func (m *manager) ensurePrivileged(ctx context.Context, i *discordgo.InteractionCreate, action string) error {
	allowed, err := m.hasEditorOrAdminRole(ctx, i)
	if err != nil {
		return fmt.Errorf("Unable to verify whether you can %s right now.", action)
	}
	if !allowed {
		return fmt.Errorf("Only editors or admins can %s.", action)
	}
	return nil
}

func (m *manager) hasEditorOrAdminRole(ctx context.Context, i *discordgo.InteractionCreate) (bool, error) {
	if i == nil || i.GuildID == "" || m.guildConfigResolver == nil {
		return false, fmt.Errorf("guild configuration unavailable")
	}

	guildConfig, err := m.guildConfigResolver.GetGuildConfigWithContext(ctx, i.GuildID)
	if err != nil {
		return false, err
	}
	if guildConfig == nil {
		return false, fmt.Errorf("guild configuration unavailable")
	}

	member, err := m.resolveInteractionMember(i)
	if err != nil {
		return false, err
	}
	if member == nil {
		return false, fmt.Errorf("guild member unavailable")
	}

	return memberHasRole(member, guildConfig.AdminRoleID) || memberHasRole(member, guildConfig.EditorRoleID), nil
}

func (m *manager) resolveInteractionMember(i *discordgo.InteractionCreate) (*discordgo.Member, error) {
	if i == nil {
		return nil, fmt.Errorf("interaction unavailable")
	}
	if i.Member != nil && len(i.Member.Roles) > 0 {
		return i.Member, nil
	}
	if m.session == nil || i.GuildID == "" {
		return i.Member, nil
	}

	userID, err := interactionUserID(i)
	if err != nil {
		return i.Member, err
	}

	member, err := m.session.GuildMember(i.GuildID, userID)
	if err != nil {
		return i.Member, err
	}
	if member != nil {
		return member, nil
	}
	return i.Member, nil
}

func isChallengeParticipant(challenge *clubtypes.ChallengeDetail, actorID string) bool {
	if challenge == nil || actorID == "" {
		return false
	}
	return challenge.ChallengerExternalID != nil && *challenge.ChallengerExternalID == actorID ||
		challenge.DefenderExternalID != nil && *challenge.DefenderExternalID == actorID
}

func memberHasRole(member *discordgo.Member, roleID string) bool {
	if member == nil || roleID == "" {
		return false
	}
	for _, memberRoleID := range member.Roles {
		if memberRoleID == roleID {
			return true
		}
	}
	return false
}

func challengeRoundAnnouncementKey(challenge clubtypes.ChallengeDetail) (string, bool) {
	if challenge.LinkedRound == nil || !challenge.LinkedRound.IsActive || challenge.LinkedRound.RoundID == "" || challenge.ID == "" {
		return "", false
	}
	return challenge.ID + ":" + challenge.LinkedRound.RoundID, true
}

func (m *manager) canSendRoundAnnouncement(key string, now time.Time) bool {
	if lastSentRaw, ok := m.roundAnnouncements.Load(key); ok {
		if lastSentAt, ok := lastSentRaw.(time.Time); ok && now.Sub(lastSentAt) < challengeRoundAnnouncementTTL {
			return false
		}
	}
	return true
}

func (m *manager) recordRoundAnnouncement(key string, now time.Time) {
	m.roundAnnouncements.Store(key, now)
	m.pruneRoundAnnouncements(now)
}

func (m *manager) pruneRoundAnnouncements(now time.Time) {
	m.roundAnnouncements.Range(func(key, value any) bool {
		lastSentAt, ok := value.(time.Time)
		if !ok || now.Sub(lastSentAt) >= challengeRoundAnnouncementTTL {
			m.roundAnnouncements.Delete(key)
		}
		return true
	})
}
