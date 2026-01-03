package dataflow

import (
	"time"
)

// Option configures the behavior of pipeline stages.
type Option func(*config)

type config struct {
	workers    int
	maxRetries int
	backoff    func(int) time.Duration
	bufferSize int
	// errorHandler handles errors. If it returns true, the pipeline continues (swallows error).
	// If false or nil, the pipeline might stop or the error is logged (implementation dependent).
	// For this library, if errorHandler is nil, we typically drop the error or stop?
	// Idiomatic: Map returns (value, error). If error, we might drop the item.
	errorHandler func(error) bool
}

// defaultConfig returns the default configuration.
func defaultConfig() *config {
	return &config{
		workers:    1,
		maxRetries: 0,
		bufferSize: 0,
	}
}

// WithWorkers sets the number of concurrent workers for a stage.
// Default is 1 (sequential).
func WithWorkers(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.workers = n
		}
	}
}

// WithBufferSize sets the buffer size for the output channel of a stage.
func WithBufferSize(n int) Option {
	return func(c *config) {
		if n >= 0 {
			c.bufferSize = n
		}
	}
}

// WithRetry enables retry logic for the stage operation.
func WithRetry(maxRetries int, backoff func(attempt int) time.Duration) Option {
	return func(c *config) {
		c.maxRetries = maxRetries
		c.backoff = backoff
	}
}

// WithErrorHandler sets a custom error handler.
// If the handler returns true, the error is considered handled and the pipeline continues (item skipped).
// If false, it might stop the pipeline or bubble up depending on the stage.
func WithErrorHandler(h func(error) bool) Option {
	return func(c *config) {
		c.errorHandler = h
	}
}

// ConstantBackoff returns a backoff function that always returns the same duration.
func ConstantBackoff(d time.Duration) func(int) time.Duration {
	return func(_ int) time.Duration {
		return d
	}
}

// ExponentialBackoff returns a backoff function that increases the duration exponentially.
// backoff = initial * 2^(attempt-1)
func ExponentialBackoff(initial time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		if attempt <= 1 {
			return initial
		}
		return initial * time.Duration(1<<(attempt-1))
	}
}
