package storage

import (
	"context"
	"errors"
	"sync"
	"time"
)

type item[T any] struct {
	value     T
	expiresAt time.Time
}

// FakeStorage is a modular fake implementation of ISInterface for testing.
type FakeStorage[T any] struct {
	data map[string]item[T]
	mu   sync.RWMutex

	// Programmable behaviors
	SetFunc    func(ctx context.Context, correlationID string, interaction T) error
	GetFunc    func(ctx context.Context, correlationID string) (T, error)
	DeleteFunc func(ctx context.Context, correlationID string)

	DefaultTTL time.Duration
	Calls      []string
	muCalls    sync.Mutex
}

// NewFakeStorage creates a new instance of FakeStorage.
func NewFakeStorage[T any]() *FakeStorage[T] {
	return &FakeStorage[T]{
		data:       make(map[string]item[T]),
		DefaultTTL: 1 * time.Hour,
	}
}

func (f *FakeStorage[T]) RecordCall(method string) {
	f.muCalls.Lock()
	defer f.muCalls.Unlock()
	f.Calls = append(f.Calls, method)
}

func (f *FakeStorage[T]) GetCalls() []string {
	f.muCalls.Lock()
	defer f.muCalls.Unlock()
	out := make([]string, len(f.Calls))
	copy(out, f.Calls)
	return out
}

// Get retrieves the item from the in-memory map.
func (f *FakeStorage[T]) Get(ctx context.Context, correlationID string) (T, error) {
	f.RecordCall("Get")
	if f.GetFunc != nil {
		return f.GetFunc(ctx, correlationID)
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if correlationID == "" {
		var zero T
		return zero, errors.New("correlation ID is empty")
	}

	val, ok := f.data[correlationID]
	if !ok {
		var zero T
		return zero, errors.New("item not found or expired")
	}

	if !val.expiresAt.IsZero() && time.Now().After(val.expiresAt) {
		var zero T
		return zero, errors.New("item not found or expired")
	}

	return val.value, nil
}

// Set stores the item in the in-memory map.
func (f *FakeStorage[T]) Set(ctx context.Context, correlationID string, interaction T) error {
	f.RecordCall("Set")
	if f.SetFunc != nil {
		return f.SetFunc(ctx, correlationID, interaction)
	}

	if correlationID == "" {
		return errors.New("correlation ID is empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.data[correlationID] = item[T]{
		value:     interaction,
		expiresAt: time.Now().Add(f.DefaultTTL),
	}
	return nil
}

// Delete removes the item from the in-memory map.
func (f *FakeStorage[T]) Delete(ctx context.Context, correlationID string) {
	f.RecordCall("Delete")
	if f.DeleteFunc != nil {
		f.DeleteFunc(ctx, correlationID)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, correlationID)
}
