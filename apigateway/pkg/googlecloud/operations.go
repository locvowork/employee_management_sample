package googlecloud

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
)

const (
	KindTaskList = "TaskList"
	KindTask     = "Task"
)

// CreateTaskList creates a new task list with a string ID.
func (c *Client) CreateTaskList(ctx context.Context, list *TaskList) error {
	if list.ID == "" {
		return fmt.Errorf("task list ID cannot be empty")
	}
	if list.CreatedAt.IsZero() {
		list.CreatedAt = time.Now()
	}

	key := datastore.NameKey(KindTaskList, list.ID, nil)
	_, err := c.ds.Put(ctx, key, list)
	return err
}

// GetTaskList retrieves a task list by ID.
func (c *Client) GetTaskList(ctx context.Context, id string) (*TaskList, error) {
	key := datastore.NameKey(KindTaskList, id, nil)
	var list TaskList
	if err := c.ds.Get(ctx, key, &list); err != nil {
		return nil, err
	}
	list.ID = id
	return &list, nil
}

// CreateTask creates a new task under a specific task list (ancestor query support).
func (c *Client) CreateTask(ctx context.Context, taskListID string, task *Task) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	// IncompleteKey will auto-generate an int64 ID
	key := datastore.IncompleteKey(KindTask, parentKey)

	newKey, err := c.ds.Put(ctx, key, task)
	if err != nil {
		return err
	}
	task.ID = newKey.ID
	task.TaskListID = taskListID
	return nil
}

// ListTasksByList retrieves all tasks belonging to a specific task list using an ancestor query.
func (c *Client) ListTasksByList(ctx context.Context, taskListID string) ([]Task, error) {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	query := datastore.NewQuery(KindTask).Ancestor(parentKey).Order("created_at")

	var tasks []Task
	keys, err := c.ds.GetAll(ctx, query, &tasks)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		tasks[i].ID = key.ID
		tasks[i].TaskListID = taskListID
	}

	return tasks, nil
}

// ListAllTasksComplex demo: filter by priority and status across all task lists.
func (c *Client) ListAllTasksComplex(ctx context.Context, minPriority int, done bool) ([]Task, error) {
	query := datastore.NewQuery(KindTask).
		Filter("priority >=", minPriority).
		Filter("done =", done).
		Order("-priority").
		Order("created_at")

	var tasks []Task
	keys, err := c.ds.GetAll(ctx, query, &tasks)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		tasks[i].ID = key.ID
		// Note: Finding the parent ID would require parsing the key.Parent() if needed.
		if key.Parent != nil {
			tasks[i].TaskListID = key.Parent.Name
		}
	}

	return tasks, nil
}

// UpdateTask updates an existing task.
func (c *Client) UpdateTask(ctx context.Context, taskListID string, taskID int64, task *Task) error {
	parentKey := datastore.NameKey(KindTaskList, taskListID, nil)
	key := datastore.IDKey(KindTask, taskID, parentKey)

	_, err := c.ds.Put(ctx, key, task)
	return err
}
