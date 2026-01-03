package pipeline

import (
	"fmt"
	"sync"
	"time"
)

// ActionFunc defines the function signature for actions
type ActionFunc func(interface{}) error

// ActionBlock represents a block that executes an action for each input message
// It supports configurable retry policies and concurrency for parallel processing
type ActionBlock struct {
	*BaseBlock
	input      chan interface{}
	action     ActionFunc
	targets    []*Target
	targetsMux sync.RWMutex
	stopOnce   sync.Once
	options    BlockOptions
}

// NewActionBlock creates a new ActionBlock with the specified action function and options
// Default behavior: no retry, sequential processing (1 worker)
func NewActionBlock(action ActionFunc, opts ...Option) *ActionBlock {
	options := applyOptions(opts)
	
	b := &ActionBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}, options.BufferSize),
		action:    action,
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

// Post sends a message to the action block
func (b *ActionBlock) Post(message interface{}) bool {
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
func (b *ActionBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop for a single worker
func (b *ActionBlock) process() {
	defer b.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			b.Fault(fmt.Errorf("panic in ActionBlock: %v", r))
		}
	}()

	for {
		select {
		case <-b.ctx.Done():
			return

		case msg, ok := <-b.input:
			if !ok {
				return
			}

			// Execute the action function with retry if configured
			err := b.executeAction(msg)
			if err != nil {
				b.Fault(err)
				continue
			}

			// Get a copy of targets to avoid holding the lock while sending
			b.targetsMux.RLock()
			targets := make([]*Target, len(b.targets))
			copy(targets, b.targets)
			b.targetsMux.RUnlock()

			// Forward the message to all targets
			for _, target := range targets {
				if target.filter == nil || target.filter(msg) {
					select {
					case target.ch <- msg:
					case <-b.ctx.Done():
						return
					}
				}
			}
		}
	}
}

// executeAction executes the action function with retry logic if configured
func (b *ActionBlock) executeAction(msg interface{}) error {
	if b.options.RetryPolicy == nil || b.options.RetryPolicy.MaxRetries <= 1 {
		// No retry policy or only one attempt allowed
		return b.action(msg)
	}

	var lastErr error
	maxAttempts := b.options.RetryPolicy.MaxRetries

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := b.action(msg)
		if err == nil {
			return nil // Success
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
				return b.ctx.Err()
			}
		}
	}

	return lastErr
}

// Complete marks the block as completed and closes the input channel
// This signals all workers to finish processing
func (b *ActionBlock) Complete() {
	b.stopOnce.Do(func() {
		close(b.input)
	})
}

