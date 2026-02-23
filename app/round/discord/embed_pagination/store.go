package embedpagination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

const (
	defaultRequestTimeout = 2 * time.Second
	defaultSnapshotTTL    = 30 * 24 * time.Hour
)

type PersistenceConfig struct {
	EventBus       eventbus.EventBus
	Helper         utils.Helpers
	Logger         *slog.Logger
	RequestTimeout time.Duration
	SnapshotTTL    time.Duration
}

type eventSnapshotStore struct {
	eventBus eventbus.EventBus
	helper   utils.Helpers
	logger   *slog.Logger

	requestTimeout time.Duration
	snapshotTTL    time.Duration

	subscribeOnce sync.Once
	subscribeErr  error

	getMu       sync.Mutex
	getInflight map[string]chan roundevents.PaginationSnapshotGetResultPayloadV1

	upsertMu       sync.Mutex
	upsertInflight map[string]chan roundevents.PaginationSnapshotUpsertResultPayloadV1

	deleteMu       sync.Mutex
	deleteInflight map[string]chan roundevents.PaginationSnapshotDeleteResultPayloadV1
}

var configuredStore struct {
	mu    sync.RWMutex
	store *eventSnapshotStore
}

func ConfigurePersistence(cfg PersistenceConfig) {
	if cfg.EventBus == nil || cfg.Helper == nil {
		clearConfiguredStore()
		return
	}

	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	ttl := cfg.SnapshotTTL
	if ttl <= 0 {
		ttl = defaultSnapshotTTL
	}

	store := &eventSnapshotStore{
		eventBus:       cfg.EventBus,
		helper:         cfg.Helper,
		logger:         cfg.Logger,
		requestTimeout: timeout,
		snapshotTTL:    ttl,
		getInflight:    make(map[string]chan roundevents.PaginationSnapshotGetResultPayloadV1),
		upsertInflight: make(map[string]chan roundevents.PaginationSnapshotUpsertResultPayloadV1),
		deleteInflight: make(map[string]chan roundevents.PaginationSnapshotDeleteResultPayloadV1),
	}

	configuredStore.mu.Lock()
	configuredStore.store = store
	configuredStore.mu.Unlock()
}

func clearConfiguredStore() {
	configuredStore.mu.Lock()
	configuredStore.store = nil
	configuredStore.mu.Unlock()
}

func currentStore() *eventSnapshotStore {
	configuredStore.mu.RLock()
	defer configuredStore.mu.RUnlock()
	return configuredStore.store
}

func Set(snapshot *Snapshot) {
	if snapshot == nil || snapshot.MessageID == "" {
		return
	}

	store := currentStore()
	if store != nil {
		if err := store.Set(context.Background(), snapshot); err != nil {
			store.logWarn("backend pagination snapshot upsert failed, using in-memory fallback", "message_id", snapshot.MessageID, "error", err)
		}
	}

	setInMemory(snapshot)
}

func Delete(messageID string) {
	if messageID == "" {
		return
	}

	store := currentStore()
	if store != nil {
		if err := store.Delete(context.Background(), messageID); err != nil {
			store.logWarn("backend pagination snapshot delete failed, using in-memory fallback", "message_id", messageID, "error", err)
		}
	}

	deleteInMemory(messageID)
}

func Get(messageID string) (*Snapshot, bool) {
	if messageID == "" {
		return nil, false
	}

	store := currentStore()
	if store != nil {
		snapshot, found, err := store.Get(context.Background(), messageID)
		if err == nil {
			if found && snapshot != nil {
				setInMemory(snapshot)
			}
			return snapshot, found
		}
		store.logWarn("backend pagination snapshot get failed, using in-memory fallback", "message_id", messageID, "error", err)
	}

	return getInMemory(messageID)
}

func Update(messageID string, mutate func(snapshot *Snapshot) bool) (*Snapshot, bool) {
	if messageID == "" || mutate == nil {
		return nil, false
	}

	store := currentStore()
	if store != nil {
		snapshot, found, err := store.Update(context.Background(), messageID, mutate)
		if err == nil {
			if found && snapshot != nil {
				setInMemory(snapshot)
			}
			return snapshot, found
		}
		store.logWarn("backend pagination snapshot update failed, using in-memory fallback", "message_id", messageID, "error", err)
	}

	return updateInMemory(messageID, mutate)
}

func RenderPage(messageID string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, int, int, error) {
	if messageID == "" {
		return nil, nil, 0, 0, errors.New("message id is empty")
	}

	store := currentStore()
	if store != nil {
		embed, components, actualPage, totalPages, err := store.RenderPage(context.Background(), messageID, page)
		if err == nil {
			return embed, components, actualPage, totalPages, nil
		}
		store.logWarn("backend pagination render failed, using in-memory fallback", "message_id", messageID, "error", err)
	}

	return renderPageInMemory(messageID, page)
}

func (s *eventSnapshotStore) Set(ctx context.Context, snapshot *Snapshot) error {
	if snapshot == nil || snapshot.MessageID == "" {
		return nil
	}

	if err := s.ensureSubscribers(); err != nil {
		return err
	}

	persisted := cloneSnapshot(snapshot)

	existing, found, err := s.getRemote(ctx, snapshot.MessageID)
	if err != nil {
		return err
	}
	if found && existing != nil {
		persisted.CurrentPage = existing.CurrentPage
	}

	return s.upsertRemote(ctx, persisted)
}

func (s *eventSnapshotStore) Delete(ctx context.Context, messageID string) error {
	if messageID == "" {
		return nil
	}

	if err := s.ensureSubscribers(); err != nil {
		return err
	}

	requestID := watermill.NewUUID()
	waiter := make(chan roundevents.PaginationSnapshotDeleteResultPayloadV1, 1)
	s.addDeleteInflight(requestID, waiter)
	defer s.removeDeleteInflight(requestID)

	payload := roundevents.PaginationSnapshotDeleteRequestedPayloadV1{
		RequestID: requestID,
		MessageID: messageID,
	}

	msg, err := s.helper.CreateNewMessage(payload, roundevents.PaginationSnapshotDeleteRequestedV1)
	if err != nil {
		return fmt.Errorf("create snapshot delete message: %w", err)
	}
	if msg == nil {
		return errors.New("create snapshot delete message returned nil")
	}

	if err := s.eventBus.Publish(roundevents.PaginationSnapshotDeleteRequestedV1, msg); err != nil {
		return fmt.Errorf("publish snapshot delete request: %w", err)
	}

	waitCtx, cancel := s.timeoutContext(ctx)
	defer cancel()

	select {
	case result := <-waiter:
		if result.Error != "" {
			return errors.New(result.Error)
		}
		if !result.Success {
			return errors.New("snapshot delete failed")
		}
		return nil
	case <-waitCtx.Done():
		return fmt.Errorf("snapshot delete timed out: %w", waitCtx.Err())
	}
}

func (s *eventSnapshotStore) Get(ctx context.Context, messageID string) (*Snapshot, bool, error) {
	if messageID == "" {
		return nil, false, nil
	}
	if err := s.ensureSubscribers(); err != nil {
		return nil, false, err
	}
	return s.getRemote(ctx, messageID)
}

func (s *eventSnapshotStore) Update(ctx context.Context, messageID string, mutate func(snapshot *Snapshot) bool) (*Snapshot, bool, error) {
	if messageID == "" || mutate == nil {
		return nil, false, nil
	}

	if err := s.ensureSubscribers(); err != nil {
		return nil, false, err
	}

	snapshot, found, err := s.getRemote(ctx, messageID)
	if err != nil || !found || snapshot == nil {
		return snapshot, found, err
	}

	updated := cloneSnapshot(snapshot)
	if updated == nil {
		return nil, false, nil
	}

	if mutate(updated) {
		if err := s.upsertRemote(ctx, updated); err != nil {
			return nil, false, err
		}
	}

	return cloneSnapshot(updated), true, nil
}

func (s *eventSnapshotStore) RenderPage(ctx context.Context, messageID string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent, int, int, error) {
	if messageID == "" {
		return nil, nil, 0, 0, errors.New("message id is empty")
	}
	if err := s.ensureSubscribers(); err != nil {
		return nil, nil, 0, 0, err
	}

	snapshot, found, err := s.getRemote(ctx, messageID)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	if !found || snapshot == nil {
		return nil, nil, 0, 0, fmt.Errorf("pagination snapshot not found for message %s", messageID)
	}

	embed, components, actualPage, totalPages := renderSnapshot(snapshot, page)
	snapshot.CurrentPage = actualPage

	if err := s.upsertRemote(ctx, snapshot); err != nil {
		return nil, nil, 0, 0, err
	}

	return embed, components, actualPage, totalPages, nil
}

func (s *eventSnapshotStore) ensureSubscribers() error {
	s.subscribeOnce.Do(func() {
		subscriptions := []struct {
			topic   string
			consume func(msgCh <-chan *message.Message)
		}{
			{topic: roundevents.PaginationSnapshotGetResultV1, consume: s.consumeGetResults},
			{topic: roundevents.PaginationSnapshotUpsertResultV1, consume: s.consumeUpsertResults},
			{topic: roundevents.PaginationSnapshotDeleteResultV1, consume: s.consumeDeleteResults},
		}

		for _, subscription := range subscriptions {
			msgCh, err := s.eventBus.Subscribe(context.Background(), subscription.topic)
			if err != nil {
				s.subscribeErr = fmt.Errorf("subscribe to %s: %w", subscription.topic, err)
				return
			}
			if msgCh == nil {
				s.subscribeErr = fmt.Errorf("subscribe to %s returned nil channel", subscription.topic)
				return
			}
			go subscription.consume(msgCh)
		}
	})

	return s.subscribeErr
}

func (s *eventSnapshotStore) consumeGetResults(msgCh <-chan *message.Message) {
	for msg := range msgCh {
		var payload roundevents.PaginationSnapshotGetResultPayloadV1
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			s.logWarn("failed to unmarshal snapshot get result", "error", err)
			msg.Ack()
			continue
		}
		msg.Ack()

		waiter, ok := s.popGetInflight(payload.RequestID)
		if !ok {
			continue
		}
		select {
		case waiter <- payload:
		default:
		}
	}
}

func (s *eventSnapshotStore) consumeUpsertResults(msgCh <-chan *message.Message) {
	for msg := range msgCh {
		var payload roundevents.PaginationSnapshotUpsertResultPayloadV1
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			s.logWarn("failed to unmarshal snapshot upsert result", "error", err)
			msg.Ack()
			continue
		}
		msg.Ack()

		waiter, ok := s.popUpsertInflight(payload.RequestID)
		if !ok {
			continue
		}
		select {
		case waiter <- payload:
		default:
		}
	}
}

func (s *eventSnapshotStore) consumeDeleteResults(msgCh <-chan *message.Message) {
	for msg := range msgCh {
		var payload roundevents.PaginationSnapshotDeleteResultPayloadV1
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			s.logWarn("failed to unmarshal snapshot delete result", "error", err)
			msg.Ack()
			continue
		}
		msg.Ack()

		waiter, ok := s.popDeleteInflight(payload.RequestID)
		if !ok {
			continue
		}
		select {
		case waiter <- payload:
		default:
		}
	}
}

func (s *eventSnapshotStore) getRemote(ctx context.Context, messageID string) (*Snapshot, bool, error) {
	requestID := watermill.NewUUID()
	waiter := make(chan roundevents.PaginationSnapshotGetResultPayloadV1, 1)
	s.addGetInflight(requestID, waiter)
	defer s.removeGetInflight(requestID)

	payload := roundevents.PaginationSnapshotGetRequestedPayloadV1{
		RequestID: requestID,
		MessageID: messageID,
	}

	msg, err := s.helper.CreateNewMessage(payload, roundevents.PaginationSnapshotGetRequestedV1)
	if err != nil {
		return nil, false, fmt.Errorf("create snapshot get message: %w", err)
	}
	if msg == nil {
		return nil, false, errors.New("create snapshot get message returned nil")
	}

	if err := s.eventBus.Publish(roundevents.PaginationSnapshotGetRequestedV1, msg); err != nil {
		return nil, false, fmt.Errorf("publish snapshot get request: %w", err)
	}

	waitCtx, cancel := s.timeoutContext(ctx)
	defer cancel()

	select {
	case result := <-waiter:
		if result.Error != "" {
			return nil, false, errors.New(result.Error)
		}
		if !result.Found || len(result.Snapshot) == 0 {
			return nil, false, nil
		}
		snapshot, err := unmarshalSnapshot(result.Snapshot)
		if err != nil {
			return nil, false, fmt.Errorf("decode snapshot payload: %w", err)
		}
		return snapshot, true, nil
	case <-waitCtx.Done():
		return nil, false, fmt.Errorf("snapshot get timed out: %w", waitCtx.Err())
	}
}

func (s *eventSnapshotStore) upsertRemote(ctx context.Context, snapshot *Snapshot) error {
	snapshotJSON, err := marshalSnapshot(snapshot)
	if err != nil {
		return err
	}

	requestID := watermill.NewUUID()
	waiter := make(chan roundevents.PaginationSnapshotUpsertResultPayloadV1, 1)
	s.addUpsertInflight(requestID, waiter)
	defer s.removeUpsertInflight(requestID)

	payload := roundevents.PaginationSnapshotUpsertRequestedPayloadV1{
		RequestID:  requestID,
		MessageID:  snapshot.MessageID,
		Snapshot:   snapshotJSON,
		TTLSeconds: int(s.snapshotTTL / time.Second),
	}

	msg, err := s.helper.CreateNewMessage(payload, roundevents.PaginationSnapshotUpsertRequestedV1)
	if err != nil {
		return fmt.Errorf("create snapshot upsert message: %w", err)
	}
	if msg == nil {
		return errors.New("create snapshot upsert message returned nil")
	}

	if err := s.eventBus.Publish(roundevents.PaginationSnapshotUpsertRequestedV1, msg); err != nil {
		return fmt.Errorf("publish snapshot upsert request: %w", err)
	}

	waitCtx, cancel := s.timeoutContext(ctx)
	defer cancel()

	select {
	case result := <-waiter:
		if result.Error != "" {
			return errors.New(result.Error)
		}
		if !result.Success {
			return errors.New("snapshot upsert failed")
		}
		return nil
	case <-waitCtx.Done():
		return fmt.Errorf("snapshot upsert timed out: %w", waitCtx.Err())
	}
}

func (s *eventSnapshotStore) timeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, s.requestTimeout)
}

func (s *eventSnapshotStore) addGetInflight(requestID string, waiter chan roundevents.PaginationSnapshotGetResultPayloadV1) {
	s.getMu.Lock()
	s.getInflight[requestID] = waiter
	s.getMu.Unlock()
}

func (s *eventSnapshotStore) popGetInflight(requestID string) (chan roundevents.PaginationSnapshotGetResultPayloadV1, bool) {
	s.getMu.Lock()
	defer s.getMu.Unlock()
	waiter, ok := s.getInflight[requestID]
	if ok {
		delete(s.getInflight, requestID)
	}
	return waiter, ok
}

func (s *eventSnapshotStore) removeGetInflight(requestID string) {
	s.getMu.Lock()
	delete(s.getInflight, requestID)
	s.getMu.Unlock()
}

func (s *eventSnapshotStore) addUpsertInflight(requestID string, waiter chan roundevents.PaginationSnapshotUpsertResultPayloadV1) {
	s.upsertMu.Lock()
	s.upsertInflight[requestID] = waiter
	s.upsertMu.Unlock()
}

func (s *eventSnapshotStore) popUpsertInflight(requestID string) (chan roundevents.PaginationSnapshotUpsertResultPayloadV1, bool) {
	s.upsertMu.Lock()
	defer s.upsertMu.Unlock()
	waiter, ok := s.upsertInflight[requestID]
	if ok {
		delete(s.upsertInflight, requestID)
	}
	return waiter, ok
}

func (s *eventSnapshotStore) removeUpsertInflight(requestID string) {
	s.upsertMu.Lock()
	delete(s.upsertInflight, requestID)
	s.upsertMu.Unlock()
}

func (s *eventSnapshotStore) addDeleteInflight(requestID string, waiter chan roundevents.PaginationSnapshotDeleteResultPayloadV1) {
	s.deleteMu.Lock()
	s.deleteInflight[requestID] = waiter
	s.deleteMu.Unlock()
}

func (s *eventSnapshotStore) popDeleteInflight(requestID string) (chan roundevents.PaginationSnapshotDeleteResultPayloadV1, bool) {
	s.deleteMu.Lock()
	defer s.deleteMu.Unlock()
	waiter, ok := s.deleteInflight[requestID]
	if ok {
		delete(s.deleteInflight, requestID)
	}
	return waiter, ok
}

func (s *eventSnapshotStore) removeDeleteInflight(requestID string) {
	s.deleteMu.Lock()
	delete(s.deleteInflight, requestID)
	s.deleteMu.Unlock()
}

func marshalSnapshot(snapshot *Snapshot) (json.RawMessage, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	return json.RawMessage(payload), nil
}

func unmarshalSnapshot(payload json.RawMessage) (*Snapshot, error) {
	if len(payload) == 0 {
		return nil, errors.New("snapshot payload is empty")
	}

	var snapshot Snapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	return cloneSnapshot(&snapshot), nil
}

func (s *eventSnapshotStore) logWarn(msg string, args ...any) {
	if s.logger != nil {
		s.logger.Warn(msg, args...)
	}
}
