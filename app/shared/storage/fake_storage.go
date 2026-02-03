package storage

import (
	"context"
	"errors"
)

// FakeStorage is a fake implementation of ISInterface for testing.
type FakeStorage[T any] struct {
	data map[string]T
}

// NewFakeStorage creates a new instance of FakeStorage.
func NewFakeStorage[T any]() *FakeStorage[T] {
	return &FakeStorage[T]{
		data: make(map[string]T),
	}
}

// Get retrieves the item from the in-memory map.
func (f *FakeStorage[T]) Get(ctx context.Context, correlationID string) (T, error) {
	val, ok := f.data[correlationID]
	if !ok {
		var zero T
		return zero, errors.New("interaction not found")
	}
	return val, nil
}

// Set stores the item in the in-memory map.
func (f *FakeStorage[T]) Set(ctx context.Context, correlationID string, interaction T) error {
	f.data[correlationID] = interaction
	return nil
}

// Delete removes the item from the in-memory map.
func (f *FakeStorage[T]) Delete(ctx context.Context, correlationID string) {
	delete(f.data, correlationID)
}
