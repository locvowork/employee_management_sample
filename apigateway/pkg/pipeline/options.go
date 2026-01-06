package pipeline

import (
	"time"
)

// BlockOptions configures the behavior of pipeline blocks
type BlockOptions struct {
	// RetryPolicy defines the retry behavior for operations that can fail
	RetryPolicy *RetryPolicy

	// ConcurrencyDegree specifies the number of concurrent workers processing messages
	// Default is 1 (sequential processing)
	ConcurrencyDegree int

	// BufferSize specifies the capacity of the input channel
	// Default varies by block type
	BufferSize int
}

// RetryPolicy defines the retry policy for operations
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts (including the initial attempt)
	// Default is 1 (no retries)
	MaxRetries int

	// Backoff is the initial backoff duration between retries
	// The actual backoff time is calculated as: Backoff * (attempt + 1)
	// Default is 0 (no backoff)
	Backoff time.Duration
}

// Option is a function that configures BlockOptions
type Option func(*BlockOptions)

// DefaultBlockOptions returns the default block options
func DefaultBlockOptions() BlockOptions {
	return BlockOptions{
		RetryPolicy:       nil, // No retry by default
		ConcurrencyDegree: 1,   // Sequential processing by default
		BufferSize:        0,   // Unbuffered by default
	}
}

// WithRetryPolicy configures a retry policy for the block
func WithRetryPolicy(policy RetryPolicy) Option {
	return func(o *BlockOptions) {
		o.RetryPolicy = &policy
	}
}

// WithConcurrencyDegree sets the number of concurrent workers
func WithConcurrencyDegree(degree int) Option {
	return func(o *BlockOptions) {
		if degree > 0 {
			o.ConcurrencyDegree = degree
		}
	}
}

// WithBufferSize sets the buffer size for the input channel
func WithBufferSize(size int) Option {
	return func(o *BlockOptions) {
		if size > 0 {
			o.BufferSize = size
		}
	}
}

// applyOptions applies the given options to the default options
func applyOptions(opts []Option) BlockOptions {
	options := DefaultBlockOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// Target represents a target that can receive messages from a source block
type Target struct {
	ch     chan<- interface{}
	filter func(interface{}) bool
}

// NewTarget creates a new target with the specified channel
func NewTarget(ch chan<- interface{}) *Target {
	return &Target{
		ch: ch,
	}
}

// SetFilter sets the filter function for the target
func (t *Target) SetFilter(filter func(interface{}) bool) {
	t.filter = filter
}
