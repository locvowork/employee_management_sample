# Pipeline

A TPL-Dataflow-style pipeline framework for Go, implemented using channels and goroutines. This package provides a way to create data processing pipelines with support for buffering, transformation, action execution, retry policies, and concurrent processing.

## Features

- **BaseBlock**: Base implementation with completion and fault handling
- **BufferBlock**: Buffers messages for consumption by linked blocks with concurrent processing
- **TransformBlock**: Transforms input messages using a transform function with retry support
- **ActionBlock**: Executes an action for each input message with retry support
- **Retry Policies**: Configurable retry behavior for TransformBlock and ActionBlock
- **Concurrency**: Configurable number of concurrent workers per block
- **Context Support**: Built-in support for context cancellation
- **Thread-safe**: All blocks are safe for concurrent use
- **No External Dependencies**: Pure Go implementation

## Installation

```bash
go get -u github.com/your-org/pipeline
```

## Usage

### Importing the Package

```go
import "github.com/your-org/pipeline"
```

### Creating a Simple Pipeline

```go
// Create a buffer block with a capacity of 10 messages
buffer := pipeline.NewBufferBlock(pipeline.WithBufferSize(10))

// Create a transform block that converts strings to uppercase
transform := pipeline.NewTransformBlock(func(input interface{}) (interface{}, error) {
    if str, ok := input.(string); ok {
        return strings.ToUpper(str), nil
    }
    return nil, fmt.Errorf("expected string, got %T", input)
})

// Create an action block that prints the result
action := pipeline.NewActionBlock(func(input interface{}) error {
    fmt.Println("Processed:", input)
    return nil
})

// Link the blocks together
pipeline.LinkTo(buffer, transform, nil)
pipeline.LinkTo(transform, action, nil)

// Post some messages to the buffer
buffer.Post("hello")
buffer.Post("world")

// Complete the pipeline and wait for all messages to be processed
buffer.Complete()
pipeline.WaitAll(buffer, transform, action)
```

### Using Retry Policies

```go
// Create a retry policy
policy := pipeline.RetryPolicy{
    MaxRetries: 3,
    Backoff:    100 * time.Millisecond,
}

// Create a transform block with retry policy
transform := pipeline.NewTransformBlock(
    func(input interface{}) (interface{}, error) {
        // Simulate a potentially failing operation
        if rand.Float64() < 0.7 {
            return nil, fmt.Errorf("temporary error")
        }
        return input.(string) + "-processed", nil
    },
    pipeline.WithRetryPolicy(policy),
)

// Create an action block with retry policy
action := pipeline.NewActionBlock(
    func(input interface{}) error {
        // Action that might fail
        return someOperation(input)
    },
    pipeline.WithRetryPolicy(policy),
)

pipeline.LinkTo(transform, action, nil)

// Post some messages
for i := 0; i < 5; i++ {
    transform.Post(fmt.Sprintf("message-%d", i))
}

transform.Complete()
pipeline.WaitAll(transform, action)
```

### Using Concurrency

```go
// Create a buffer with 10 concurrent workers
buffer := pipeline.NewBufferBlock(
    pipeline.WithBufferSize(100),
    pipeline.WithConcurrencyDegree(10),
)

// Create a transform block with 5 concurrent workers
transform := pipeline.NewTransformBlock(
    func(input interface{}) (interface{}, error) {
        // CPU-intensive transformation
        return expensiveCalculation(input), nil
    },
    pipeline.WithConcurrencyDegree(5),
)

// Create an action block with 3 concurrent workers
action := pipeline.NewActionBlock(
    func(input interface{}) error {
        // I/O-intensive action
        return saveToDatabase(input)
    },
    pipeline.WithConcurrencyDegree(3),
)

pipeline.LinkTo(buffer, transform, nil)
pipeline.LinkTo(transform, action, nil)

// Post many messages for parallel processing
for i := 0; i < 1000; i++ {
    buffer.Post(i)
}

buffer.Complete()
pipeline.WaitAll(buffer, transform, action)
```

### Combining Retry and Concurrency

```go
// Create a transform block with both retry policy and concurrency
transform := pipeline.NewTransformBlock(
    func(input interface{}) (interface{}, error) {
        // Operation that might fail and is CPU-intensive
        return processWithRetry(input)
    },
    pipeline.WithRetryPolicy(pipeline.RetryPolicy{
        MaxRetries: 3,
        Backoff:    50 * time.Millisecond,
    }),
    pipeline.WithConcurrencyDegree(8),
    pipeline.WithBufferSize(50),
)
```

## Block Types

### BaseBlock

The foundation for all blocks, providing:

- Completion handling
- Fault handling
- Context support
- Thread-safe operations

### BufferBlock

- Buffers messages for consumption by linked blocks
- Supports backpressure by dropping messages when full
- Can be linked to multiple targets
- Supports concurrent processing with multiple workers

**Options:**

- `WithBufferSize(size int)`: Sets the buffer capacity
- `WithConcurrencyDegree(degree int)`: Sets the number of concurrent workers

### TransformBlock

- Applies a transform function to each input message
- Forwards the transformed result to linked blocks
- Supports filtering of output messages
- Supports retry policies for the transform function
- Supports concurrent processing with multiple workers

**Options:**

- `WithRetryPolicy(policy RetryPolicy)`: Configures retry behavior
- `WithConcurrencyDegree(degree int)`: Sets the number of concurrent workers
- `WithBufferSize(size int)`: Sets the buffer capacity

### ActionBlock

- Executes an action for each input message
- Forwards the input to linked blocks after successful execution
- Supports error handling and fault propagation
- Supports retry policies for the action function
- Supports concurrent processing with multiple workers

**Options:**

- `WithRetryPolicy(policy RetryPolicy)`: Configures retry behavior
- `WithConcurrencyDegree(degree int)`: Sets the number of concurrent workers
- `WithBufferSize(size int)`: Sets the buffer capacity

## RetryPolicy

The `RetryPolicy` struct configures retry behavior:

```go
type RetryPolicy struct {
    MaxRetries int           // Maximum number of retry attempts (including initial)
    Backoff    time.Duration // Initial backoff duration between retries
}
```

- **MaxRetries**: Total attempts (initial + retries). Default is 1 (no retries)
- **Backoff**: Initial wait time between retries. Actual backoff is `Backoff * (attempt + 1)`

## Best Practices

1. **Error Handling**: Always handle errors returned by `Wait()` or `Error()` methods.
2. **Resource Cleanup**: Call `Complete()` on blocks when they're no longer needed to release resources.
3. **Backpressure**: Use appropriate buffer sizes to balance memory usage and throughput.
4. **Context Cancellation**: Use the provided context to support graceful shutdown.
5. **Concurrency**: Configure concurrency degree based on the nature of your workload:
   - CPU-bound tasks: Set concurrency to number of CPU cores
   - I/O-bound tasks: Set higher concurrency (e.g., 10-100)
6. **Retry Policies**: Use retry policies for operations that can fail temporarily (network calls, database operations)
7. **Thread Safety**: All blocks are safe for concurrent use, but be mindful of shared state in your transform/action functions.

## Performance Considerations

- **Concurrency Degree**: Tune based on workload characteristics

  - CPU-intensive: `runtime.NumCPU()` or lower
  - I/O-intensive: Higher values (10-100+)
  - Mixed: Experiment with different values

- **Buffer Sizes**: Balance memory vs. throughput

  - Small buffers: Lower memory, more backpressure
  - Large buffers: Higher memory, smoother processing

- **Retry Backoff**: Consider exponential backoff for distributed systems
  - Linear backoff: `Backoff * attempt`
  - Exponential backoff: `Backoff * (2 ^ attempt)`

## License

MIT
