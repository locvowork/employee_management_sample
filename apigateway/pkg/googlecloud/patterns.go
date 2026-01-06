package googlecloud

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/datastore"
)

// Common Datastore errors for easier handling in services.
var (
	ErrNotFound      = errors.New("entity not found")
	ErrAlreadyExists = errors.New("entity already exists")
	ErrInvalidKey    = errors.New("invalid key")
)

// WrapDatastoreError converts Datastore-specific errors to domain errors.
func WrapDatastoreError(err error) error {
	if err == nil {
		return nil
	}
	if err == datastore.ErrNoSuchEntity {
		return ErrNotFound
	}
	// Could be expanded to check for other specific errors
	return err
}

// IsNotFoundError checks if an error is a not-found error.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, datastore.ErrNoSuchEntity)
}

// --- Retry Logic ---

// RetryConfig holds configuration for retry operations.
type RetryConfig struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// DefaultRetryConfig returns sensible defaults for retries.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
	}
}

// WithRetry executes a function with exponential backoff retry.
// Useful for handling transient Datastore errors.
func WithRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	wait := cfg.InitialWait

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		// Don't wait after the last attempt
		if attempt < cfg.MaxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
			// Exponential backoff
			wait *= 2
			if wait > cfg.MaxWait {
				wait = cfg.MaxWait
			}
		}
	}
	return lastErr
}

// --- Optimistic Locking ---

// VersionedEntity is an interface for entities that support optimistic locking.
type VersionedEntity interface {
	GetVersion() int64
	SetVersion(int64)
}

// VersionedTask demonstrates optimistic locking with a version field.
type VersionedTask struct {
	Task
	Version int64 `datastore:"version" json:"version"`
}

func (v *VersionedTask) GetVersion() int64    { return v.Version }
func (v *VersionedTask) SetVersion(ver int64) { v.Version = ver }

// UpdateWithOptimisticLock updates an entity only if its version matches.
// This prevents lost updates in concurrent scenarios.
func (c *Client) UpdateWithOptimisticLock(ctx context.Context, taskListID string, taskID int64, updateFn func(*VersionedTask) error) error {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	key := datastore.IDKey(KindTask, taskID, parentKey)

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var task VersionedTask
		if err := tx.Get(key, &task); err != nil {
			return WrapDatastoreError(err)
		}

		expectedVersion := task.Version

		// Apply the update
		if err := updateFn(&task); err != nil {
			return err
		}

		// Increment version
		task.Version = expectedVersion + 1

		_, err := tx.Put(key, &task)
		return err
	})

	return err
}

// --- Upsert Pattern ---

// UpsertTaskList creates or updates a task list.
func (c *Client) UpsertTaskList(ctx context.Context, list *TaskList) error {
	if list.ID == "" {
		return ErrInvalidKey
	}

	key := datastore.NameKey(KindTaskList, list.ID, nil)

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var existing TaskList
		err := tx.Get(key, &existing)

		if err == datastore.ErrNoSuchEntity {
			// Create new
			if list.CreatedAt.IsZero() {
				list.CreatedAt = time.Now()
			}
		} else if err != nil {
			return err
		} else {
			// Update existing - preserve creation time
			list.CreatedAt = existing.CreatedAt
		}

		_, err = tx.Put(key, list)
		return err
	})

	return err
}

// --- Soft Delete Pattern ---

// SoftDeletableTask demonstrates soft delete pattern.
type SoftDeletableTask struct {
	Task
	DeletedAt *time.Time `datastore:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// IsDeleted returns true if the entity is soft-deleted.
func (t *SoftDeletableTask) IsDeleted() bool {
	return t.DeletedAt != nil
}

// SoftDeleteTask marks a task as deleted without removing it.
func (c *Client) SoftDeleteTask(ctx context.Context, taskListID string, taskID int64) error {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	key := datastore.IDKey(KindTask, taskID, parentKey)

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var task SoftDeletableTask
		if err := tx.Get(key, &task); err != nil {
			return WrapDatastoreError(err)
		}

		now := time.Now()
		task.DeletedAt = &now

		_, err := tx.Put(key, &task)
		return err
	})

	return err
}

// ListActiveTasks returns only non-deleted tasks.
func (c *Client) ListActiveTasks(ctx context.Context, taskListID string) ([]SoftDeletableTask, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	// Filter for null deleted_at - Note: Datastore doesn't index null values well.
	// A better approach is to use a boolean "is_deleted" field.
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		Order("created_at")

	var tasks []SoftDeletableTask
	keys, err := c.ds.GetAll(ctx, query, &tasks)
	if err != nil {
		return nil, err
	}

	// Filter out deleted tasks in memory
	result := make([]SoftDeletableTask, 0, len(tasks))
	for i, task := range tasks {
		if !task.IsDeleted() {
			task.ID = keys[i].ID
			task.TaskListID = taskListID
			result = append(result, task)
		}
	}

	return result, nil
}
