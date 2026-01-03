package pipeline

import (
	"fmt"
	"sync"
	"time"
)

// TransformFunc defines the function signature for transformation
// TransformFunc is a function that transforms an input value to an output value
// and returns an error if the transformation fails
type TransformFunc func(interface{}) (interface{}, error)

// TransformBlock represents a block that transforms input messages
// It supports configurable retry policies and concurrency for parallel processing
type TransformBlock struct {
	*BaseBlock
	input      chan interface{}
	transform  TransformFunc
	targets    []*Target
	targetsMux sync.RWMutex
	stopOnce   sync.Once
	options    BlockOptions
}

// NewTransformBlock creates a new TransformBlock with the specified transform function and options
// Default behavior: no retry, sequential processing (1 worker)
func NewTransformBlock(transform TransformFunc, opts ...Option) *TransformBlock {
	options := applyOptions(opts)
	
	b := &TransformBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}, options.BufferSize),
		transform: transform,
		targets:   make([]*Target, 0),
		options:   options,
	}

	// Start multiple worker goroutines based on concurrency degree
	b.wg.Add(options.ConcurrencyDegree)
	for i := 0; i < options.ConcurrencyDegree; i++ {
		go b.process()
	}

	return b
}

// Post sends a message to the transform block
func (b *TransformBlock) Post(message interface{}) bool {
	if b.IsCompleted() {
		return false
	}

	select {
	case b.input <- message:
		return true
	default:
		return false
	}
}

// LinkTo links this block to a target block with an optional filter function
func (b *TransformBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop for a single worker
func (b *TransformBlock) process() {
	defer b.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			b.Fault(fmt.Errorf("panic in TransformBlock: %v", r))
		}
		// Close target channels when all processing is done
		b.targetsMux.RLock()
		for _, t := range b.targets {
			close(t.ch)
		}
		b.targetsMux.RUnlock()
		b.SignalCompletion()
	}()

	for {
		select {
		case <-b.ctx.Done():
			return

		case msg, ok := <-b.input:
			if !ok {
				return
			}

			// Apply the transform function with retry if configured
			result, err := b.executeTransform(msg)
			if err != nil {
				b.Fault(err)
				continue
			}

			if result == nil {
				continue
			}

			// Get a copy of targets to avoid holding the lock while sending
			b.targetsMux.RLock()
			targets := make([]*Target, len(b.targets))
			copy(targets, b.targets)
			b.targetsMux.RUnlock()

			// Forward the result to all targets
			for _, target := range targets {
				if target.filter == nil || target.filter(result) {
					select {
					case target.ch <- result:
					case <-b.ctx.Done():
						return
					}
				}
			}
		}
	}
}

// executeTransform executes the transform function with retry logic if configured
func (b *TransformBlock) executeTransform(msg interface{}) (interface{}, error) {
	if b.options.RetryPolicy == nil || b.options.RetryPolicy.MaxRetries <= 1 {
		// No retry policy or only one attempt allowed
		return b.transform(msg)
	}

	var lastErr error
	maxAttempts := b.options.RetryPolicy.MaxRetries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := b.transform(msg)
		if err == nil {
			return result, nil // Success
		}

		lastErr = err

		// If this was the last attempt, break
		if attempt == maxAttempts-1 {
			break
		}

		// Calculate backoff time
		if b.options.RetryPolicy.Backoff > 0 {
			backoff := time.Duration(attempt+1) * b.options.RetryPolicy.Backoff
			select {
			case <-time.After(backoff):
			case <-b.ctx.Done():
				return nil, b.ctx.Err()
			}
		}
	}

	return nil, lastErr
}

// Complete marks the block as completed and closes the input channel
// This signals all workers to finish processing
func (b *TransformBlock) Complete() {
	b.stopOnce.Do(func() {
		close(b.input)
	})
}

