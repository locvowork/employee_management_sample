package pipeline

import (
	"fmt"
	"sync"
)

// BufferBlock represents a block that buffers messages for consumption by linked blocks
// It supports configurable concurrency for parallel processing of messages
type BufferBlock struct {
	*BaseBlock
	input      chan interface{}
	targets    []*Target
	targetsMux sync.RWMutex
	capacity   int
	stopOnce   sync.Once
}

// NewBufferBlock creates a new BufferBlock with the specified options
// Default behavior: unbuffered channel, sequential processing (1 worker)
func NewBufferBlock(opts ...Option) *BufferBlock {
	options := applyOptions(opts)
	
	b := &BufferBlock{
		BaseBlock: NewBaseBlock(),
		input:     make(chan interface{}, options.BufferSize),
		targets:   make([]*Target, 0),
		capacity:  options.BufferSize,
	}

	// Start multiple worker goroutines based on concurrency degree
	b.wg.Add(options.ConcurrencyDegree)
	for i := 0; i < options.ConcurrencyDegree; i++ {
		go b.process()
	}

	return b
}

// Post sends a message to the buffer block
func (b *BufferBlock) Post(message interface{}) bool {
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
func (b *BufferBlock) LinkTo(target *Target, filter func(interface{}) bool) {
	b.targetsMux.Lock()
	defer b.targetsMux.Unlock()

	b.targets = append(b.targets, target)

	// If there's a filter, set it on the target
	if filter != nil {
		target.SetFilter(filter)
	}
}

// process handles the message processing loop for a single worker
func (b *BufferBlock) process() {
	defer b.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			b.Fault(fmt.Errorf("panic in BufferBlock: %v", r))
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

// Complete marks the block as completed and closes the input channel
// This signals all workers to finish processing
func (b *BufferBlock) Complete() {
	b.stopOnce.Do(func() {
		close(b.input)
	})
}

