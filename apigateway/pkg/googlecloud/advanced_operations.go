package googlecloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/datastore"
)

// --- Transactions ---

// CreateTaskInTransaction demonstrates creating a task within a transaction.
// Transactions ensure atomicity - either all operations succeed or none do.
func (c *Client) CreateTaskInTransaction(ctx context.Context, taskListID string, task *Task) error {
	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// First, verify the parent TaskList exists
		parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
		var list TaskList
		if err := tx.Get(parentKey, &list); err != nil {
			return fmt.Errorf("parent task list not found: %w", err)
		}

		// Create the task
		taskKey := datastore.IncompleteKey(KindTask, parentKey)
		_, err := tx.Put(taskKey, task)
		if err != nil {
			return err
		}
		// Note: In transaction, we can't get the ID immediately.
		// The caller should query for it or use a name key.
		task.TaskListID = taskListID
		return nil
	})
	return err
}

// TransferTask moves a task from one list to another atomically.
func (c *Client) TransferTask(ctx context.Context, taskID int64, fromListID, toListID string) error {
	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// Get the task from old list
		fromParentKey := datastore.NameKey(KindTaskList, fromListID, nil)
		oldKey := datastore.IDKey(KindTask, taskID, fromParentKey)

		var task Task
		if err := tx.Get(oldKey, &task); err != nil {
			return fmt.Errorf("task not found: %w", err)
		}

		// Delete from old location
		if err := tx.Delete(oldKey); err != nil {
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

// --- Batch Operations ---

// BatchCreateTasks creates multiple tasks in a single batch operation.
// Batch operations are more efficient than individual puts.
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
		tasks[i].TaskListID = taskListID
	}
	return nil
}

// BatchGetTasks retrieves multiple tasks by their IDs.
func (c *Client) BatchGetTasks(ctx context.Context, taskListID string, taskIDs []int64) ([]Task, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)

	keys := make([]*datastore.Key, len(taskIDs))
	for i, id := range taskIDs {
		keys[i] = datastore.IDKey(KindTask, id, parentKey)
	}

	tasks := make([]Task, len(taskIDs))
	if err := c.ds.GetMulti(ctx, keys, tasks); err != nil {
		return nil, err
	}

	for i, key := range keys {
		tasks[i].ID = key.ID
		tasks[i].TaskListID = taskListID
	}
	return tasks, nil
}

// BatchDeleteTasks deletes multiple tasks in a single operation.
func (c *Client) BatchDeleteTasks(ctx context.Context, taskListID string, taskIDs []int64) error {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)

	keys := make([]*datastore.Key, len(taskIDs))
	for i, id := range taskIDs {
		keys[i] = datastore.IDKey(KindTask, id, parentKey)
	}

	return c.ds.DeleteMulti(ctx, keys)
}

// --- Pagination with Cursors ---

// PageResult holds paginated results with a cursor for the next page.
type PageResult struct {
	Tasks      []Task
	NextCursor string
	HasMore    bool
}

// ListTasksPaginated retrieves tasks with pagination support using cursors.
// Cursors are more efficient than offset-based pagination for large datasets.
func (c *Client) ListTasksPaginated(ctx context.Context, taskListID string, pageSize int, cursorStr string) (*PageResult, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		Order("created_at").
		Limit(pageSize + 1) // Fetch one extra to check if there are more

	// Apply cursor if provided
	if cursorStr != "" {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Start(cursor)
	}

	var tasks []Task
	it := c.ds.Run(ctx, query)

	for {
		var task Task
		key, err := it.Next(&task)
		if err != nil {
			// iterator.Done signals end of results
			break
		}
		task.ID = key.ID
		task.TaskListID = taskListID
		tasks = append(tasks, task)
	}

	result := &PageResult{
		Tasks:   tasks,
		HasMore: len(tasks) > pageSize,
	}

	// Trim to requested page size and get cursor
	if result.HasMore {
		result.Tasks = tasks[:pageSize]
		cursor, err := it.Cursor()
		if err != nil {
			return nil, err
		}
		result.NextCursor = cursor.String()
	}

	return result, nil
}

// --- Key-Only Queries ---

// CountTasksByList counts tasks without loading their data using a keys-only query.
// This is more efficient when you only need the count.
func (c *Client) CountTasksByList(ctx context.Context, taskListID string) (int, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		KeysOnly()

	keys, err := c.ds.GetAll(ctx, query, nil)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// GetTaskIDs returns only the IDs of tasks (keys-only query).
func (c *Client) GetTaskIDs(ctx context.Context, taskListID string) ([]int64, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		KeysOnly()

	keys, err := c.ds.GetAll(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, len(keys))
	for i, key := range keys {
		ids[i] = key.ID
	}
	return ids, nil
}

// --- Projection Queries ---

// TaskSummary holds projected fields only.
type TaskSummary struct {
	ID          int64  `datastore:"-"`
	Description string `datastore:"description"`
	Done        bool   `datastore:"done"`
}

// ListTaskSummaries retrieves only specific fields using projection.
// Projection queries are more efficient when you don't need all fields.
func (c *Client) ListTaskSummaries(ctx context.Context, taskListID string) ([]TaskSummary, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		Project("description", "done")

	var summaries []TaskSummary
	keys, err := c.ds.GetAll(ctx, query, &summaries)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		summaries[i].ID = key.ID
	}
	return summaries, nil
}

// --- Delete with Query ---

// DeleteCompletedTasks deletes all tasks marked as done in a task list.
func (c *Client) DeleteCompletedTasks(ctx context.Context, taskListID string) (int, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).
		Ancestor(parentKey).
		Filter("done =", true).
		KeysOnly()

	keys, err := c.ds.GetAll(ctx, query, nil)
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	if err := c.ds.DeleteMulti(ctx, keys); err != nil {
		return 0, err
	}
	return len(keys), nil
}
