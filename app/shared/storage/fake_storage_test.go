package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestFakeStorage_Concurrency(t *testing.T) {
	fs := NewFakeStorage[string]()
	ctx := context.Background()
	numRoutines := 100
	numIterations := 100

	var wg sync.WaitGroup
	wg.Add(numRoutines * 3) // Set, Get, Delete

	// Concurrent Sets
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				fs.Set(ctx, key, "value")
			}
		}(i)
	}

	// Concurrent Gets
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, _ = fs.Get(ctx, key)
			}
		}(i)
	}

	// Concurrent Deletes
	for i := 0; i < numRoutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				fs.Delete(ctx, key)
			}
		}(i)
	}

	wg.Wait()
}

func TestFakeStorage_Expiration(t *testing.T) {
	fs := NewFakeStorage[string]()
	fs.DefaultTTL = 10 * time.Millisecond
	ctx := context.Background()

	fs.Set(ctx, "short-lived", "value")

	val, err := fs.Get(ctx, "short-lived")
	if err != nil || val != "value" {
		t.Errorf("Expected value, got error: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	_, err = fs.Get(ctx, "short-lived")
	if err == nil {
		t.Error("Expected error for expired item, got nil")
	}
}

func TestFakeStorage_EmptyCorrelationID(t *testing.T) {
	fs := NewFakeStorage[string]()
	ctx := context.Background()

	err := fs.Set(ctx, "", "value")
	if err == nil {
		t.Error("Expected error for empty correlation ID in Set, got nil")
	}

	_, err = fs.Get(ctx, "")
	if err == nil {
		t.Error("Expected error for empty correlation ID in Get, got nil")
	}
}
