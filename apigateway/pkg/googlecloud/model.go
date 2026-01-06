package googlecloud

import (
	"time"
)

// TaskList represents a group of tasks.
type TaskList struct {
	ID        string    `datastore:"-" json:"id"` // Key Name
	Name      string    `datastore:"name" json:"name"`
	CreatedAt time.Time `datastore:"created_at" json:"created_at"`
}

// Task represents a single unit of work.
type Task struct {
	ID          int64     `datastore:"-" json:"id"` // Key ID (Auto-generated int64)
	Description string    `datastore:"description" json:"description"`
	Done        bool      `datastore:"done" json:"done"`
	Priority    int       `datastore:"priority" json:"priority"`
	CreatedAt   time.Time `datastore:"created_at" json:"created_at"`

	// TaskListID is stored to easily reference the parent ID in JSON,
	// though in Datastore it's part of the Key (Ancestor).
	TaskListID string `datastore:"-" json:"task_list_id"`
}
