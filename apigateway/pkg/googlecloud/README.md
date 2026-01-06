# Google Cloud Datastore Guide for Go Microservices

This document provides a comprehensive guide to using Google Cloud Datastore in Go-based microservice applications. It covers all essential patterns, best practices, and includes working sample code from the `pkg/googlecloud` package.

---

## Table of Contents

1.  [Introduction](#introduction)
2.  [Core Concepts](#core-concepts)
3.  [Client Setup](#client-setup)
4.  [Data Modeling](#data-modeling)
5.  [Basic CRUD Operations](#basic-crud-operations)
6.  [Querying](#querying)
7.  [Transactions](#transactions)
8.  [Batch Operations](#batch-operations)
9.  [Pagination with Cursors](#pagination-with-cursors)
10. [Advanced Patterns](#advanced-patterns)
11. [Error Handling](#error-handling)
12. [Testing with Emulator](#testing-with-emulator)
13. [Best Practices for Microservices](#best-practices-for-microservices)
14. [Saga Pattern for Distributed Workflows](#saga-pattern-for-distributed-workflows)

---

## Introduction

Google Cloud Datastore is a highly scalable, fully managed NoSQL database designed for automatic scaling, high availability, and durability. It's an excellent choice for microservice architectures due to its:

-   **Schemaless nature**: Entities of the same kind can have different properties.
-   **Automatic scaling**: Handles millions of reads/writes per second.
-   **Strong consistency**: Guarantees read-your-writes consistency.
-   **ACID Transactions**: Supports multi-entity transactions.

### When to Use Datastore

| Use Case | Recommendation |
|----------|----------------|
| High read/write throughput | ✅ Excellent |
| Complex JOINs | ❌ Not suitable |
| Hierarchical data | ✅ Ancestor queries |
| Full-text search | ❌ Use Firestore or Elasticsearch |
| Real-time sync | ❌ Use Firestore |

---

## Core Concepts

### Kinds

Kinds are like tables in relational databases. Each entity belongs to a kind.

```go
const (
    KindTaskList = "TaskList"
    KindTask     = "Task"
)
```

### Keys

Keys uniquely identify entities. They can be:
-   **Name Keys**: String-based IDs (user-defined)
-   **ID Keys**: Integer-based IDs (auto-generated)

```go
// Name key (string ID)
key := datastore.NameKey(KindTaskList, "my-list-id", nil)

// Incomplete key (auto-generate ID)
key := datastore.IncompleteKey(KindTask, parentKey)

// ID key (known integer ID)
key := datastore.IDKey(KindTask, 12345, parentKey)
```

### Ancestor Relationships (Entity Groups)

Entities can have parent-child relationships forming "entity groups". Ancestor queries are strongly consistent.

```go
parentKey := datastore.NameKey(KindTaskList, "list-1", nil)
taskKey := datastore.IncompleteKey(KindTask, parentKey)
```

---

## Client Setup

### Basic Client Initialization

See: [client.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/client.go)

```go
package googlecloud

import (
    "context"
    "cloud.google.com/go/datastore"
)

type Client struct {
    ds *datastore.Client
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
    ds, err := datastore.NewClient(ctx, projectID)
    if err != nil {
        return nil, err
    }
    return &Client{ds: ds}, nil
}

func (c *Client) Close() error {
    return c.ds.Close()
}
```

### Connection in Microservices

Initialize once at application startup and reuse:

```go
func main() {
    ctx := context.Background()
    client, err := googlecloud.NewClient(ctx, os.Getenv("GCP_PROJECT_ID"))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Pass client to handlers/services
    handler := NewMyHandler(client)
}
```

---

## Data Modeling

### Struct Tags

See: [model.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/model.go)

```go
type Task struct {
    ID          int64     `datastore:"-" json:"id"`          // Excluded from storage (key ID)
    Description string    `datastore:"description" json:"description"`
    Done        bool      `datastore:"done" json:"done"`
    Priority    int       `datastore:"priority" json:"priority"`
    CreatedAt   time.Time `datastore:"created_at" json:"created_at"`
    TaskListID  string    `datastore:"-" json:"task_list_id"` // Derived from parent key
}
```

### Tag Options

| Tag | Purpose |
|-----|---------|
| `datastore:"-"` | Exclude field from storage |
| `datastore:"name"` | Custom property name |
| `datastore:",noindex"` | Don't index this property |
| `datastore:",omitempty"` | Don't store if empty/zero |
| `datastore:",flatten"` | Flatten nested struct |

---

## Basic CRUD Operations

See: [operations.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/operations.go)

### Create

```go
func (c *Client) CreateTaskList(ctx context.Context, list *TaskList) error {
    key := datastore.NameKey(KindTaskList, list.ID, nil)
    _, err := c.ds.Put(ctx, key, list)
    return err
}
```

### Read

```go
func (c *Client) GetTaskList(ctx context.Context, id string) (*TaskList, error) {
    key := datastore.NameKey(KindTaskList, id, nil)
    var list TaskList
    if err := c.ds.Get(ctx, key, &list); err != nil {
        return nil, err
    }
    list.ID = id
    return &list, nil
}
```

### Update

```go
func (c *Client) UpdateTask(ctx context.Context, taskListID string, taskID int64, task *Task) error {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    key := datastore.IDKey(KindTask, taskID, parentKey)
    _, err := c.ds.Put(ctx, key, task)
    return err
}
```

### Delete

```go
func (c *Client) DeleteTask(ctx context.Context, taskListID string, taskID int64) error {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    key := datastore.IDKey(KindTask, taskID, parentKey)
    return c.ds.Delete(ctx, key)
}
```

---

## Querying

### Simple Query

```go
query := datastore.NewQuery(KindTask).
    Filter("done =", false).
    Order("created_at")

var tasks []Task
keys, err := c.ds.GetAll(ctx, query, &tasks)
```

### Ancestor Query (Strongly Consistent)

```go
parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
query := datastore.NewQuery(KindTask).
    Ancestor(parentKey).
    Order("created_at")
```

### Complex Filters

See: [operations.go#ListAllTasksComplex](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/operations.go#L79)

```go
query := datastore.NewQuery(KindTask).
    Filter("priority >=", minPriority).
    Filter("done =", done).
    Order("-priority").
    Order("created_at")
```

> **Note**: Composite filters on multiple properties require composite indexes defined in `index.yaml`.

### Keys-Only Query

Efficient when you only need keys/counts:

```go
query := datastore.NewQuery(KindTask).
    Ancestor(parentKey).
    KeysOnly()

keys, _ := c.ds.GetAll(ctx, query, nil)
count := len(keys)
```

### Projection Query

Retrieve only specific properties:

```go
query := datastore.NewQuery(KindTask).
    Ancestor(parentKey).
    Project("description", "done")
```

---

## Transactions

See: [advanced_operations.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/advanced_operations.go)

### Simple Transaction

```go
func (c *Client) CreateTaskInTransaction(ctx context.Context, taskListID string, task *Task) error {
    _, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
        // Verify parent exists
        parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
        var list TaskList
        if err := tx.Get(parentKey, &list); err != nil {
            return err
        }

        // Create task
        taskKey := datastore.IncompleteKey(KindTask, parentKey)
        _, err := tx.Put(taskKey, task)
        return err
    })
    return err
}
```

### Cross-Entity Transaction

```go
func (c *Client) TransferTask(ctx context.Context, taskID int64, fromListID, toListID string) error {
    _, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
        // Get from old location
        fromKey := datastore.IDKey(KindTask, taskID, 
            datastore.NameKey(KindTaskList, fromListID, nil))
        var task Task
        if err := tx.Get(fromKey, &task); err != nil {
            return err
        }

        // Delete from old location
        if err := tx.Delete(fromKey); err != nil {
            return err
        }

        // Create in new location
        toParentKey := datastore.NameKey(KindTaskList, toListID, nil)
        newKey := datastore.IncompleteKey(KindTask, toParentKey)
        _, err := tx.Put(newKey, &task)
        return err
    })
    return err
}
```

### Transaction Limitations

-   Max 500 entities per transaction
-   All entities must be in the same region
-   Queries within transactions must use ancestor

---

## Batch Operations

See: [advanced_operations.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/advanced_operations.go)

### Batch Put

```go
func (c *Client) BatchCreateTasks(ctx context.Context, taskListID string, tasks []*Task) error {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    
    keys := make([]*datastore.Key, len(tasks))
    for i := range tasks {
        keys[i] = datastore.IncompleteKey(KindTask, parentKey)
    }
    
    newKeys, err := c.ds.PutMulti(ctx, keys, tasks)
    if err != nil {
        return err
    }
    
    for i, key := range newKeys {
        tasks[i].ID = key.ID
    }
    return nil
}
```

### Batch Get

```go
func (c *Client) BatchGetTasks(ctx context.Context, taskListID string, taskIDs []int64) ([]Task, error) {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    
    keys := make([]*datastore.Key, len(taskIDs))
    for i, id := range taskIDs {
        keys[i] = datastore.IDKey(KindTask, id, parentKey)
    }
    
    tasks := make([]Task, len(taskIDs))
    err := c.ds.GetMulti(ctx, keys, tasks)
    return tasks, err
}
```

### Batch Delete

```go
func (c *Client) BatchDeleteTasks(ctx context.Context, taskListID string, taskIDs []int64) error {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    
    keys := make([]*datastore.Key, len(taskIDs))
    for i, id := range taskIDs {
        keys[i] = datastore.IDKey(KindTask, id, parentKey)
    }
    
    return c.ds.DeleteMulti(ctx, keys)
}
```

---

## Pagination with Cursors

See: [advanced_operations.go#ListTasksPaginated](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/advanced_operations.go#L120)

Cursors are more efficient than offset-based pagination for large datasets.

```go
type PageResult struct {
    Tasks      []Task
    NextCursor string
    HasMore    bool
}

func (c *Client) ListTasksPaginated(ctx context.Context, taskListID string, pageSize int, cursorStr string) (*PageResult, error) {
    parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
    query := datastore.NewQuery(KindTask).
        Ancestor(parentKey).
        Order("created_at").
        Limit(pageSize + 1)

    // Apply cursor if provided
    if cursorStr != "" {
        cursor, err := datastore.DecodeCursor(cursorStr)
        if err != nil {
            return nil, err
        }
        query = query.Start(cursor)
    }

    var tasks []Task
    it := c.ds.Run(ctx, query)

    for {
        var task Task
        key, err := it.Next(&task)
        if err == datastore.Done {
            break
        }
        if err != nil {
            return nil, err
        }
        task.ID = key.ID
        tasks = append(tasks, task)
    }

    result := &PageResult{
        Tasks:   tasks,
        HasMore: len(tasks) > pageSize,
    }

    if result.HasMore {
        result.Tasks = tasks[:pageSize]
        cursor, _ := it.Cursor()
        result.NextCursor = cursor.String()
    }

    return result, nil
}
```

---

## Advanced Patterns

See: [patterns.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/patterns.go)

### Retry with Exponential Backoff

```go
type RetryConfig struct {
    MaxAttempts int
    InitialWait time.Duration
    MaxWait     time.Duration
}

func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
    var lastErr error
    wait := cfg.InitialWait

    for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        if attempt < cfg.MaxAttempts-1 {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(wait):
            }
            wait *= 2
            if wait > cfg.MaxWait {
                wait = cfg.MaxWait
            }
        }
    }
    return lastErr
}
```

### Optimistic Locking

Prevent lost updates in concurrent scenarios:

```go
type VersionedTask struct {
    Task
    Version int64 `datastore:"version"`
}

func (c *Client) UpdateWithOptimisticLock(ctx context.Context, taskListID string, taskID int64, updateFn func(*VersionedTask) error) error {
    _, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
        parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
        key := datastore.IDKey(KindTask, taskID, parentKey)

        var task VersionedTask
        if err := tx.Get(key, &task); err != nil {
            return err
        }

        expectedVersion := task.Version
        if err := updateFn(&task); err != nil {
            return err
        }

        task.Version = expectedVersion + 1
        _, err := tx.Put(key, &task)
        return err
    })
    return err
}
```

### Upsert (Create or Update)

```go
func (c *Client) UpsertTaskList(ctx context.Context, list *TaskList) error {
    key := datastore.NameKey(KindTaskList, list.ID, nil)

    _, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
        var existing TaskList
        err := tx.Get(key, &existing)

        if err == datastore.ErrNoSuchEntity {
            list.CreatedAt = time.Now()
        } else if err != nil {
            return err
        } else {
            list.CreatedAt = existing.CreatedAt // Preserve
        }

        _, err = tx.Put(key, list)
        return err
    })
    return err
}
```

### Soft Delete

```go
type SoftDeletableTask struct {
    Task
    DeletedAt *time.Time `datastore:"deleted_at,omitempty"`
}

func (t *SoftDeletableTask) IsDeleted() bool {
    return t.DeletedAt != nil
}

func (c *Client) SoftDeleteTask(ctx context.Context, taskListID string, taskID int64) error {
    // ... transaction to set DeletedAt
}
```

---

## Error Handling

See: [patterns.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/patterns.go)

### Common Errors

| Error | Meaning | Action |
|-------|---------|--------|
| `datastore.ErrNoSuchEntity` | Entity not found | Return 404 |
| `datastore.ErrConcurrentTransaction` | Transaction conflict | Retry |
| `context.DeadlineExceeded` | Timeout | Retry with backoff |

### Error Wrapping

```go
var (
    ErrNotFound      = errors.New("entity not found")
    ErrAlreadyExists = errors.New("entity already exists")
)

func WrapDatastoreError(err error) error {
    if err == nil {
        return nil
    }
    if err == datastore.ErrNoSuchEntity {
        return ErrNotFound
    }
    return err
}

func IsNotFoundError(err error) bool {
    return errors.Is(err, ErrNotFound) || errors.Is(err, datastore.ErrNoSuchEntity)
}
```

---

## Testing with Emulator

### Starting the Emulator

```bash
# Install (once)
gcloud components install cloud-datastore-emulator

# Start
gcloud beta emulators datastore start --host-port=0.0.0.0:8081
```

### Environment Setup

```bash
export DATASTORE_EMULATOR_HOST=localhost:8081
export GCP_PROJECT_ID=demo-project
```

### Integration Test Example

```go
func TestTaskOperations(t *testing.T) {
    ctx := context.Background()
    client, err := googlecloud.NewClient(ctx, "test-project")
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    // Create a task list
    list := &googlecloud.TaskList{ID: "test-list", Name: "Test"}
    if err := client.CreateTaskList(ctx, list); err != nil {
        t.Fatalf("CreateTaskList failed: %v", err)
    }

    // Create a task
    task := &googlecloud.Task{Description: "Test task", Priority: 1}
    if err := client.CreateTask(ctx, "test-list", task); err != nil {
        t.Fatalf("CreateTask failed: %v", err)
    }

    // Verify
    tasks, err := client.ListTasksByList(ctx, "test-list")
    if err != nil {
        t.Fatalf("ListTasksByList failed: %v", err)
    }
    if len(tasks) != 1 {
        t.Errorf("Expected 1 task, got %d", len(tasks))
    }
}
```

---

## Best Practices for Microservices

### 1. Use a Single Client Instance

```go
// Good: Single client, passed to services
type App struct {
    dsClient *googlecloud.Client
}

// Bad: Creating new client per request
func handler(w http.ResponseWriter, r *http.Request) {
    client, _ := googlecloud.NewClient(r.Context(), projectID) // Don't do this!
    defer client.Close()
}
```

### 2. Context Propagation

Always pass request context for proper timeout/cancellation:

```go
func (h *Handler) CreateTask(c echo.Context) error {
    ctx := c.Request().Context()
    return h.client.CreateTask(ctx, listID, task)
}
```

### 3. Design Entity Groups Carefully

-   Entities in the same group have strongly consistent reads
-   Max 1 write per entity group per second
-   Use ancestor relationships for related data that's queried together

### 4. Composite Indexes

Create `index.yaml` for complex queries:

```yaml
indexes:
- kind: Task
  properties:
  - name: priority
    direction: desc
  - name: created_at
```

### 5. Use Batch Operations

```go
// Good: Single batch call
client.BatchCreateTasks(ctx, listID, tasks)

// Bad: Individual calls in a loop
for _, task := range tasks {
    client.CreateTask(ctx, listID, task)
}
```

### 6. Handle Multi-Errors

`GetMulti` and `PutMulti` may return partial failures:

```go
tasks := make([]Task, len(keys))
err := client.ds.GetMulti(ctx, keys, tasks)

if me, ok := err.(datastore.MultiError); ok {
    for i, e := range me {
        if e != nil {
            log.Printf("Failed to get task %d: %v", i, e)
        }
    }
}
```

### 7. Limit Query Results

Always set limits to prevent OOM:

```go
query := datastore.NewQuery(KindTask).
    Limit(100) // Always set a reasonable limit
```

---

## File Reference

| [client.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/client.go) | Client wrapper and initialization |
| [model.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/model.go) | Entity struct definitions |
| [operations.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/operations.go) | Basic CRUD and query operations |
| [advanced_operations.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/advanced_operations.go) | Transactions, batching, pagination |
| [patterns.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/patterns.go) | Error handling, retry, optimistic locking |
| [saga.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/saga.go) | Saga pattern implementation |
| [saga_example.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/saga_example.go) | Order processing saga example |

---

## Saga Pattern for Distributed Workflows

See: [saga.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/saga.go) and [saga_example.go](file:///home/locvh/_projects/_personal/antigravity/employee_management_sample/apigateway/pkg/googlecloud/saga_example.go)

The Saga pattern manages distributed transactions across multiple microservices by breaking them into a sequence of local transactions. Each step has a **compensating action** (rollback) that undoes its effects if a later step fails.

### Why Use Sagas?

| Traditional 2PC | Saga Pattern |
|-----------------|--------------|
| Locks resources across services | No distributed locks |
| Single point of failure | Resilient to failures |
| Tight coupling | Loose coupling |
| Synchronous | Can be async |

### Saga Architecture with Datastore

```
┌─────────────────────────────────────────────────────────────────┐
│                     SAGA ORCHESTRATOR                           │
│  (Coordinates steps, handles failures, manages state)          │
└─────────────────────────────────────────────────────────────────┘
          │                    │                    │
          ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Inventory Svc  │  │  Payment Svc    │  │  Shipping Svc   │
│                 │  │                 │  │                 │
│  reserve()      │  │  charge()       │  │  ship()         │
│  cancel()       │  │  refund()       │  │  cancel()       │
└─────────────────┘  └─────────────────┘  └─────────────────┘
          │                    │                    │
          └────────────────────┴────────────────────┘
                               │
                               ▼
                   ┌─────────────────────┐
                   │  Google Datastore   │
                   │  (Saga State Store) │
                   └─────────────────────┘
```

### Core Entities

```go
// Saga represents a distributed transaction workflow
type Saga struct {
    ID          string     `datastore:"-"`
    Name        string     `datastore:"name"`
    Status      SagaStatus `datastore:"status"`
    CurrentStep int        `datastore:"current_step"`
    TotalSteps  int        `datastore:"total_steps"`
    Payload     string     `datastore:"payload,noindex"` // JSON
    Error       string     `datastore:"error,noindex"`
    CreatedAt   time.Time  `datastore:"created_at"`
    UpdatedAt   time.Time  `datastore:"updated_at"`
}

// SagaStep represents a single step
type SagaStep struct {
    ID          int64      `datastore:"-"`
    StepIndex   int        `datastore:"step_index"`
    Name        string     `datastore:"name"`
    ServiceName string     `datastore:"service_name"`
    Status      StepStatus `datastore:"status"`
    Input       string     `datastore:"input,noindex"`
    Output      string     `datastore:"output,noindex"`
    Error       string     `datastore:"error,noindex"`
}
```

### Saga Status Flow

```
PENDING → RUNNING → COMPLETED
              │
              ▼ (on failure)
        ROLLING_BACK → ROLLED_BACK
              │
              ▼ (compensation fails)
           FAILED
```

### Defining Steps

```go
type SagaStepDefinition struct {
    Name        string
    ServiceName string
    Execute     func(ctx context.Context, input string) (output string, err error)
    Compensate  func(ctx context.Context, input, output string) error
}
```

### Creating an Orchestrator

```go
func NewOrderProcessingSaga(client *Client) *SagaOrchestrator {
    steps := []SagaStepDefinition{
        {
            Name:        "Reserve Inventory",
            ServiceName: "inventory-service",
            Execute:     ReserveInventoryStep,
            Compensate:  CancelInventoryReservation,
        },
        {
            Name:        "Process Payment",
            ServiceName: "payment-service",
            Execute:     ProcessPaymentStep,
            Compensate:  RefundPayment,
        },
        {
            Name:        "Create Shipment",
            ServiceName: "shipping-service",
            Execute:     CreateShipmentStep,
            Compensate:  CancelShipment,
        },
        {
            Name:        "Send Notification",
            ServiceName: "notification-service",
            Execute:     SendNotificationStep,
            Compensate:  nil, // No compensation needed
        },
    }
    return NewSagaOrchestrator(client, steps)
}
```

### Executing a Saga

```go
func ProcessOrder(ctx context.Context, client *Client, order OrderPayload) error {
    orchestrator := NewOrderProcessingSaga(client)
    
    payload, _ := json.Marshal(order)
    sagaID := fmt.Sprintf("ORDER-%s-%d", order.OrderID, time.Now().Unix())
    
    // 1. Start (creates saga + steps in Datastore)
    if err := orchestrator.Start(ctx, sagaID, "Order Processing", string(payload)); err != nil {
        return err
    }
    
    // 2. Execute (runs each step, compensates on failure)
    return orchestrator.Execute(ctx, sagaID)
}
```

### Rollback Flow

When a step fails, the orchestrator automatically compensates completed steps in reverse order:

```go
func (o *SagaOrchestrator) Rollback(ctx context.Context, sagaID string, fromStep int) error {
    // Update status to ROLLING_BACK
    o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRollingBack, fromStep, "")
    
    steps, _ := o.client.GetSagaSteps(ctx, sagaID)
    
    // Compensate in reverse order
    for i := fromStep; i >= 0; i-- {
        step := steps[i]
        if step.Status != StepStatusCompleted {
            continue
        }
        
        compensator := o.steps[i].Compensate
        if compensator != nil {
            if err := compensator(ctx, step.Input, step.Output); err != nil {
                // Compensation failed - mark saga as FAILED
                return err
            }
        }
        // Mark step as COMPENSATED
        o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusCompensated, "", "")
    }
    
    return o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRolledBack, 0, "")
}
```

### Recovery After Service Restart

Sagas persist their state in Datastore, enabling recovery:

```go
type SagaRecoveryWorker struct {
    client       *Client
    orchestrators map[string]*SagaOrchestrator
    interval     time.Duration
}

func (w *SagaRecoveryWorker) Run(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.recoverSagas(ctx)
        }
    }
}

func (w *SagaRecoveryWorker) recoverSagas(ctx context.Context) {
    // Find sagas that were interrupted
    sagas, _ := w.client.ListPendingSagas(ctx, 10)
    
    for _, saga := range sagas {
        orchestrator := w.orchestrators[saga.Name]
        orchestrator.Resume(ctx, saga.ID)
    }
}
```

### Best Practices for Sagas

1.  **Idempotent Operations**: Steps and compensations should be idempotent
2.  **Timeouts**: Set appropriate timeouts for each step
3.  **Monitoring**: Track saga status and alert on stuck/failed sagas
4.  **Retry Failed Compensations**: Don't give up on compensation failures
5.  **Audit Trail**: Log all state transitions for debugging
6.  **Dead Letter Queue**: Move permanently failed sagas for manual review

### Query Examples

```go
// Find interrupted sagas (for recovery)
query := datastore.NewQuery(KindSaga).
    Filter("status =", string(SagaStatusRunning)).
    Order("-updated_at").
    Limit(10)

// Find failed sagas (for manual intervention)
query := datastore.NewQuery(KindSaga).
    Filter("status =", string(SagaStatusFailed)).
    Order("-updated_at")

// Get saga with all steps
saga, _ := client.GetSaga(ctx, sagaID)
steps, _ := client.GetSagaSteps(ctx, sagaID)
```

### Choreography vs Orchestration

| Orchestration (This Implementation) | Choreography |
|-------------------------------------|--------------|
| Central coordinator | Event-driven |
| Easier to understand flow | More decoupled |
| Single point of failure | More complex debugging |
| Better for linear workflows | Better for parallel steps |

