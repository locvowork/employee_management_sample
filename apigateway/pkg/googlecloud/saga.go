package googlecloud

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
)

// =============================================================================
// SAGA PATTERN IMPLEMENTATION
// =============================================================================
//
// The Saga pattern manages distributed transactions across multiple services
// by breaking them into a sequence of local transactions. Each step has a
// compensating action (rollback) that undoes its effects if a later step fails.
//
// This implementation uses Datastore to:
// 1. Persist the saga state (for recovery after failures)
// 2. Track each step's status
// 3. Enable saga resumption after service restarts
// =============================================================================

// --- Saga Status Constants ---

const (
	KindSaga     = "Saga"
	KindSagaStep = "SagaStep"
)

// SagaStatus represents the current state of a saga
type SagaStatus string

const (
	SagaStatusPending     SagaStatus = "PENDING"
	SagaStatusRunning     SagaStatus = "RUNNING"
	SagaStatusCompleted   SagaStatus = "COMPLETED"
	SagaStatusFailed      SagaStatus = "FAILED"
	SagaStatusRollingBack SagaStatus = "ROLLING_BACK"
	SagaStatusRolledBack  SagaStatus = "ROLLED_BACK"
)

// StepStatus represents the current state of a saga step
type StepStatus string

const (
	StepStatusPending     StepStatus = "PENDING"
	StepStatusRunning     StepStatus = "RUNNING"
	StepStatusCompleted   StepStatus = "COMPLETED"
	StepStatusFailed      StepStatus = "FAILED"
	StepStatusSkipped     StepStatus = "SKIPPED"
	StepStatusCompensated StepStatus = "COMPENSATED"
)

// --- Saga Entities ---

// Saga represents a distributed transaction workflow
type Saga struct {
	ID          string     `datastore:"-" json:"id"`
	Name        string     `datastore:"name" json:"name"`
	Status      SagaStatus `datastore:"status" json:"status"`
	CurrentStep int        `datastore:"current_step" json:"current_step"`
	TotalSteps  int        `datastore:"total_steps" json:"total_steps"`
	Payload     string     `datastore:"payload,noindex" json:"payload"` // JSON payload
	Error       string     `datastore:"error,noindex" json:"error,omitempty"`
	CreatedAt   time.Time  `datastore:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `datastore:"updated_at" json:"updated_at"`
	CompletedAt *time.Time `datastore:"completed_at,omitempty" json:"completed_at,omitempty"`
}

// SagaStep represents a single step in a saga
type SagaStep struct {
	ID            int64      `datastore:"-" json:"id"`
	SagaID        string     `datastore:"-" json:"saga_id"` // Parent key
	StepIndex     int        `datastore:"step_index" json:"step_index"`
	Name          string     `datastore:"name" json:"name"`
	ServiceName   string     `datastore:"service_name" json:"service_name"`
	Status        StepStatus `datastore:"status" json:"status"`
	Input         string     `datastore:"input,noindex" json:"input"`   // JSON input
	Output        string     `datastore:"output,noindex" json:"output"` // JSON output
	Error         string     `datastore:"error,noindex" json:"error,omitempty"`
	StartedAt     *time.Time `datastore:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt   *time.Time `datastore:"completed_at,omitempty" json:"completed_at,omitempty"`
	CompensatedAt *time.Time `datastore:"compensated_at,omitempty" json:"compensated_at,omitempty"`
}

// --- Saga Repository Operations ---

// CreateSaga creates a new saga with its steps
func (c *Client) CreateSaga(ctx context.Context, saga *Saga, steps []SagaStep) error {
	now := time.Now()
	saga.CreatedAt = now
	saga.UpdatedAt = now
	saga.Status = SagaStatusPending
	saga.CurrentStep = 0
	saga.TotalSteps = len(steps)

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		// Create saga
		sagaKey := datastore.NameKey(KindSaga, saga.ID, nil)
		if _, err := tx.Put(sagaKey, saga); err != nil {
			return err
		}

		// Create steps as children of saga
		for i := range steps {
			steps[i].StepIndex = i
			steps[i].Status = StepStatusPending
			steps[i].SagaID = saga.ID
			stepKey := datastore.IncompleteKey(KindSagaStep, sagaKey)
			if _, err := tx.Put(stepKey, &steps[i]); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// GetSaga retrieves a saga by ID
func (c *Client) GetSaga(ctx context.Context, sagaID string) (*Saga, error) {
	key := datastore.NameKey(KindSaga, sagaID, nil)
	var saga Saga
	if err := c.ds.Get(ctx, key, &saga); err != nil {
		return nil, WrapDatastoreError(err)
	}
	saga.ID = sagaID
	return &saga, nil
}

// GetSagaSteps retrieves all steps for a saga
func (c *Client) GetSagaSteps(ctx context.Context, sagaID string) ([]SagaStep, error) {
	sagaKey := datastore.NameKey(KindSaga, sagaID, nil)
	query := datastore.NewQuery(KindSagaStep).
		Ancestor(sagaKey).
		Order("step_index")

	var steps []SagaStep
	keys, err := c.ds.GetAll(ctx, query, &steps)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		steps[i].ID = key.ID
		steps[i].SagaID = sagaID
	}
	return steps, nil
}

// UpdateSagaStatus updates the saga's status and current step
func (c *Client) UpdateSagaStatus(ctx context.Context, sagaID string, status SagaStatus, currentStep int, errMsg string) error {
	sagaKey := datastore.NameKey(KindSaga, sagaID, nil)

	_, err := c.ds.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		var saga Saga
		if err := tx.Get(sagaKey, &saga); err != nil {
			return err
		}

		saga.Status = status
		saga.CurrentStep = currentStep
		saga.UpdatedAt = time.Now()
		saga.Error = errMsg

		if status == SagaStatusCompleted || status == SagaStatusRolledBack || status == SagaStatusFailed {
			now := time.Now()
			saga.CompletedAt = &now
		}

		_, err := tx.Put(sagaKey, &saga)
		return err
	})
	return err
}

// UpdateStepStatus updates a specific step's status
func (c *Client) UpdateStepStatus(ctx context.Context, sagaID string, stepIndex int, status StepStatus, output, errMsg string) error {
	sagaKey := datastore.NameKey(KindSaga, sagaID, nil)

	// Find the step by index
	query := datastore.NewQuery(KindSagaStep).
		Ancestor(sagaKey).
		Filter("step_index =", stepIndex).
		Limit(1)

	var steps []SagaStep
	keys, err := c.ds.GetAll(ctx, query, &steps)
	if err != nil || len(keys) == 0 {
		return fmt.Errorf("step not found: index %d", stepIndex)
	}

	step := steps[0]
	step.Status = status
	step.Output = output
	step.Error = errMsg
	now := time.Now()

	switch status {
	case StepStatusRunning:
		step.StartedAt = &now
	case StepStatusCompleted, StepStatusFailed:
		step.CompletedAt = &now
	case StepStatusCompensated:
		step.CompensatedAt = &now
	}

	_, err = c.ds.Put(ctx, keys[0], &step)
	return err
}

// ListPendingSagas finds sagas that need to be resumed (e.g., after service restart)
func (c *Client) ListPendingSagas(ctx context.Context, limit int) ([]Saga, error) {
	query := datastore.NewQuery(KindSaga).
		Filter("status =", string(SagaStatusRunning)).
		Order("-updated_at").
		Limit(limit)

	var sagas []Saga
	keys, err := c.ds.GetAll(ctx, query, &sagas)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		sagas[i].ID = key.Name
	}
	return sagas, nil
}

// ListFailedSagas finds sagas that failed and might need manual intervention
func (c *Client) ListFailedSagas(ctx context.Context, limit int) ([]Saga, error) {
	query := datastore.NewQuery(KindSaga).
		Filter("status =", string(SagaStatusFailed)).
		Order("-updated_at").
		Limit(limit)

	var sagas []Saga
	keys, err := c.ds.GetAll(ctx, query, &sagas)
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		sagas[i].ID = key.Name
	}
	return sagas, nil
}

// --- Saga Orchestrator ---

// StepExecutor is a function that executes a saga step
// Returns output (JSON string) and error
type StepExecutor func(ctx context.Context, input string) (output string, err error)

// StepCompensator is a function that compensates (undoes) a saga step
type StepCompensator func(ctx context.Context, input, output string) error

// SagaStepDefinition defines a step with its executor and compensator
type SagaStepDefinition struct {
	Name        string
	ServiceName string
	Execute     StepExecutor
	Compensate  StepCompensator
}

// SagaOrchestrator manages saga execution
type SagaOrchestrator struct {
	client *Client
	steps  []SagaStepDefinition
}

// NewSagaOrchestrator creates a new orchestrator with defined steps
func NewSagaOrchestrator(client *Client, steps []SagaStepDefinition) *SagaOrchestrator {
	return &SagaOrchestrator{
		client: client,
		steps:  steps,
	}
}

// Start initiates a new saga
func (o *SagaOrchestrator) Start(ctx context.Context, sagaID, name, payload string) error {
	// Create step entities
	stepEntities := make([]SagaStep, len(o.steps))
	for i, def := range o.steps {
		stepEntities[i] = SagaStep{
			Name:        def.Name,
			ServiceName: def.ServiceName,
			Input:       payload, // Each step gets the payload; could be customized
		}
	}

	saga := &Saga{
		ID:      sagaID,
		Name:    name,
		Payload: payload,
	}

	return o.client.CreateSaga(ctx, saga, stepEntities)
}

// Execute runs the saga forward
func (o *SagaOrchestrator) Execute(ctx context.Context, sagaID string) error {
	// Update saga to running
	if err := o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRunning, 0, ""); err != nil {
		return err
	}

	saga, err := o.client.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	steps, err := o.client.GetSagaSteps(ctx, sagaID)
	if err != nil {
		return err
	}

	// Execute each step
	for i, step := range steps {
		if step.Status == StepStatusCompleted {
			continue // Already done (resuming)
		}

		// Update step to running
		if err := o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusRunning, "", ""); err != nil {
			return err
		}

		// Update saga current step
		if err := o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRunning, i, ""); err != nil {
			return err
		}

		// Execute the step
		output, execErr := o.steps[i].Execute(ctx, step.Input)
		if execErr != nil {
			// Step failed - mark it and start rollback
			if err := o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusFailed, "", execErr.Error()); err != nil {
				return err
			}
			// Initiate rollback
			return o.Rollback(ctx, sagaID, i-1)
		}

		// Step succeeded
		if err := o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusCompleted, output, ""); err != nil {
			return err
		}

		// Update input for next step with output from current step
		if i+1 < len(steps) {
			steps[i+1].Input = output
		}
	}

	// All steps completed
	return o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusCompleted, saga.TotalSteps, "")
}

// Rollback compensates all completed steps in reverse order
func (o *SagaOrchestrator) Rollback(ctx context.Context, sagaID string, fromStep int) error {
	if err := o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRollingBack, fromStep, ""); err != nil {
		return err
	}

	steps, err := o.client.GetSagaSteps(ctx, sagaID)
	if err != nil {
		return err
	}

	// Compensate in reverse order
	for i := fromStep; i >= 0; i-- {
		step := steps[i]
		if step.Status != StepStatusCompleted {
			continue // Can only compensate completed steps
		}

		compensator := o.steps[i].Compensate
		if compensator == nil {
			// Mark as skipped if no compensator defined
			if err := o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusSkipped, "", ""); err != nil {
				return err
			}
			continue
		}

		// Execute compensation
		if compErr := compensator(ctx, step.Input, step.Output); compErr != nil {
			// Compensation failed - this is serious
			errMsg := fmt.Sprintf("compensation failed for step %d: %v", i, compErr)
			if err := o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusFailed, i, errMsg); err != nil {
				return err
			}
			return fmt.Errorf(errMsg)
		}

		// Mark step as compensated
		if err := o.client.UpdateStepStatus(ctx, sagaID, i, StepStatusCompensated, "", ""); err != nil {
			return err
		}
	}

	// All compensations done
	return o.client.UpdateSagaStatus(ctx, sagaID, SagaStatusRolledBack, 0, "")
}

// Resume continues a saga that was interrupted (e.g., after service restart)
func (o *SagaOrchestrator) Resume(ctx context.Context, sagaID string) error {
	saga, err := o.client.GetSaga(ctx, sagaID)
	if err != nil {
		return err
	}

	switch saga.Status {
	case SagaStatusRunning:
		return o.Execute(ctx, sagaID)
	case SagaStatusRollingBack:
		return o.Rollback(ctx, sagaID, saga.CurrentStep)
	default:
		return fmt.Errorf("saga %s is in status %s, cannot resume", sagaID, saga.Status)
	}
}
