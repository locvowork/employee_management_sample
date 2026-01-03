package pipeline

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBufferBlock_Concurrency(t *testing.T) {
	const numWorkers = 5
	const numMessages = 100

	buffer := NewBufferBlock(
		WithBufferSize(numMessages),
		WithConcurrencyDegree(numWorkers),
	)

	var processedCount int32
	var mu sync.Mutex
	processed := make(map[int]bool)

	action := NewActionBlock(func(input interface{}) error {
		atomic.AddInt32(&processedCount, 1)
		
		mu.Lock()
		processed[input.(int)] = true
		mu.Unlock()
		
		return nil
	})

	LinkTo(buffer, action, nil)

	// Post many messages
	for i := 0; i < numMessages; i++ {
		if !buffer.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	buffer.Complete()

	err := WaitAll(buffer, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}

	// Verify all unique messages were processed
	mu.Lock()
	if len(processed) != numMessages {
		t.Errorf("Expected %d unique messages, got %d", numMessages, len(processed))
	}
	mu.Unlock()
}

func TestTransformBlock_Concurrency(t *testing.T) {
	const numWorkers = 5
	const numMessages = 100

	transform := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			// Simulate some processing time
			time.Sleep(1 * time.Millisecond)
			return input.(int) * 2, nil
		},
		WithConcurrencyDegree(numWorkers),
		WithBufferSize(numMessages),
	)

	var processedCount int32
	action := NewActionBlock(func(input interface{}) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	LinkTo(transform, action, nil)

	// Post many messages
	for i := 0; i < numMessages; i++ {
		if !transform.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	transform.Complete()

	err := WaitAll(transform, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}
}

func TestActionBlock_Concurrency(t *testing.T) {
	const numWorkers = 5
	const numMessages = 100

	var processedCount int32
	action := NewActionBlock(
		func(input interface{}) error {
			// Simulate some processing time
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&processedCount, 1)
			return nil
		},
		WithConcurrencyDegree(numWorkers),
		WithBufferSize(numMessages),
	)

	// Post many messages
	for i := 0; i < numMessages; i++ {
		if !action.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	action.Complete()

	err := WaitAll(action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}
}

func TestConcurrencyWithRetry(t *testing.T) {
	const numWorkers = 3
	const numMessages = 20

	policy := RetryPolicy{
		MaxRetries: 2,
		Backoff:    5 * time.Millisecond,
	}

	var attemptCount int32
	transform := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			val := input.(int)
			// Fail first attempt for even numbers, succeed on retry
			attempt := atomic.AddInt32(&attemptCount, 1)
			if val%2 == 0 && attempt%3 != 0 {
				return nil, errors.New("temporary error")
			}
			return val * 2, nil
		},
		WithRetryPolicy(policy),
		WithConcurrencyDegree(numWorkers),
		WithBufferSize(numMessages),
	)

	var processedCount int32
	action := NewActionBlock(func(input interface{}) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	LinkTo(transform, action, nil)

	// Post messages
	for i := 0; i < numMessages; i++ {
		if !transform.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	transform.Complete()

	err := WaitAll(transform, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}
}

func TestSequentialProcessing(t *testing.T) {
	const numMessages = 10

	// Default concurrency degree is 1 (sequential)
	transform := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			return input.(int), nil
		},
	)

	var processedCount int32
	action := NewActionBlock(func(input interface{}) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	LinkTo(transform, action, nil)

	// Post messages
	for i := 0; i < numMessages; i++ {
		if !transform.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	transform.Complete()

	err := WaitAll(transform, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}
}

func TestHighConcurrency(t *testing.T) {
	const numWorkers = 50
	const numMessages = 200

	buffer := NewBufferBlock(
		WithBufferSize(numMessages),
		WithConcurrencyDegree(numWorkers),
	)

	var processedCount int32
	action := NewActionBlock(func(input interface{}) error {
		atomic.AddInt32(&processedCount, 1)
		return nil
	})

	LinkTo(buffer, action, nil)

	// Post many messages
	for i := 0; i < numMessages; i++ {
		if !buffer.Post(i) {
			t.Fatalf("Failed to post message %d", i)
		}
	}

	buffer.Complete()

	err := WaitAll(buffer, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	// Verify all messages were processed
	finalCount := atomic.LoadInt32(&processedCount)
	if finalCount != numMessages {
		t.Errorf("Expected %d messages processed, got %d", numMessages, finalCount)
	}
}