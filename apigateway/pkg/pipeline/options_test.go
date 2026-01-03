package pipeline

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestTransformBlock_WithRetry(t *testing.T) {
	// Policy: 2 retries, 10ms backoff
	policy := RetryPolicy{
		MaxRetries: 2,
		Backoff:    10 * time.Millisecond,
	}

	callCount := int32(0)
	transformBlock := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			count := atomic.AddInt32(&callCount, 1)
			val := input.(int)

			if count < 3 {
				return nil, errors.New("temporary error")
			}

			return val * 2, nil
		},
		WithRetryPolicy(policy),
	)

	results := make([]int, 0)
	done := make(chan bool)

	action := NewActionBlock(func(input interface{}) error {
		results = append(results, input.(int))
		done <- true
		return nil
	})

	LinkTo(transformBlock, action, nil)

	// Post message
	if !transformBlock.Post(10) {
		t.Fatal("Failed to post message to transform block")
	}
	transformBlock.Complete()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for action block")
	}

	err := WaitAll(transformBlock, action)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	if len(results) != 1 || results[0] != 20 {
		t.Errorf("Expected result 20, got %v", results)
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 3 {
		t.Errorf("Expected 3 calls (2 retries + 1 success), got %d", finalCount)
	}
}

func TestTransformBlock_RetryFailure(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries: 1,
		Backoff:    10 * time.Millisecond,
	}

	callCount := int32(0)
	transformBlock := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			atomic.AddInt32(&callCount, 1)
			return nil, errors.New("permanent error")
		},
		WithRetryPolicy(policy),
	)

	// Post message
	if !transformBlock.Post(10) {
		t.Fatal("Failed to post message to transform block")
	}
	transformBlock.Complete()

	err := WaitAll(transformBlock)
	if err == nil {
		t.Fatal("Expected error from WaitAll, got nil")
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 2 {
		t.Errorf("Expected 2 calls (1 retry + 1 fail), got %d", finalCount)
	}
}

func TestActionBlock_WithRetry(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries: 2,
		Backoff:    10 * time.Millisecond,
	}

	callCount := int32(0)
	actionBlock := NewActionBlock(
		func(input interface{}) error {
			count := atomic.AddInt32(&callCount, 1)
			if count < 3 {
				return errors.New("temporary error")
			}
			return nil
		},
		WithRetryPolicy(policy),
	)

	// Post message
	if !actionBlock.Post("test") {
		t.Fatal("Failed to post message to action block")
	}
	actionBlock.Complete()

	err := WaitAll(actionBlock)
	if err != nil {
		t.Fatalf("WaitAll failed: %v", err)
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 3 {
		t.Errorf("Expected 3 calls (2 retries + 1 success), got %d", finalCount)
	}
}

func TestActionBlock_RetryFailure(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries: 1,
		Backoff:    10 * time.Millisecond,
	}

	callCount := int32(0)
	actionBlock := NewActionBlock(
		func(input interface{}) error {
			atomic.AddInt32(&callCount, 1)
			return errors.New("permanent error")
		},
		WithRetryPolicy(policy),
	)

	// Post message
	if !actionBlock.Post("test") {
		t.Fatal("Failed to post message to action block")
	}
	actionBlock.Complete()

	err := WaitAll(actionBlock)
	if err == nil {
		t.Fatal("Expected error from WaitAll, got nil")
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 2 {
		t.Errorf("Expected 2 calls (1 retry + 1 fail), got %d", finalCount)
	}
}

func TestTransformBlock_NoRetry(t *testing.T) {
	callCount := int32(0)
	transformBlock := NewTransformBlock(
		func(input interface{}) (interface{}, error) {
			atomic.AddInt32(&callCount, 1)
			return nil, errors.New("error")
		},
		// No retry policy specified
	)

	// Post message
	if !transformBlock.Post(10) {
		t.Fatal("Failed to post message to transform block")
	}
	transformBlock.Complete()

	err := WaitAll(transformBlock)
	if err == nil {
		t.Fatal("Expected error from WaitAll, got nil")
	}

	finalCount := atomic.LoadInt32(&callCount)
	if finalCount != 1 {
		t.Errorf("Expected 1 call (no retry), got %d", finalCount)
	}
}

func TestRetryPolicy_DefaultValues(t *testing.T) {
	options := DefaultBlockOptions()
	
	if options.RetryPolicy != nil {
		t.Error("Expected nil RetryPolicy by default")
	}
	
	if options.ConcurrencyDegree != 1 {
		t.Errorf("Expected ConcurrencyDegree 1, got %d", options.ConcurrencyDegree)
	}
	
	if options.BufferSize != 0 {
		t.Errorf("Expected BufferSize 0, got %d", options.BufferSize)
	}
}

func TestRetryPolicy_WithOptions(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries: 5,
		Backoff:    100 * time.Millisecond,
	}
	
	options := applyOptions([]Option{
		WithRetryPolicy(policy),
		WithConcurrencyDegree(10),
		WithBufferSize(100),
	})
	
	if options.RetryPolicy == nil {
		t.Fatal("Expected RetryPolicy to be set")
	}
	
	if options.RetryPolicy.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", options.RetryPolicy.MaxRetries)
	}
	
	if options.RetryPolicy.Backoff != 100*time.Millisecond {
		t.Errorf("Expected Backoff 100ms, got %v", options.RetryPolicy.Backoff)
	}
	
	if options.ConcurrencyDegree != 10 {
		t.Errorf("Expected ConcurrencyDegree 10, got %d", options.ConcurrencyDegree)
	}
	
	if options.BufferSize != 100 {
		t.Errorf("Expected BufferSize 100, got %d", options.BufferSize)
	}
}