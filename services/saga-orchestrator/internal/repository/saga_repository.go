package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/model"
)

type SagaRepository struct {
	db *sql.DB
}

func NewSagaRepository(db *sql.DB) *SagaRepository {
	return &SagaRepository{db: db}
}

// CreateSaga starts a new saga for an order and creates all step records
func (r *SagaRepository) CreateSaga(ctx context.Context, orderID string) (*model.Saga, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Create the saga
	var saga model.Saga
	err = tx.QueryRowContext(ctx,
		`INSERT INTO sagas (order_id, status, current_step)
		 VALUES ($1, $2, $3)
		 RETURNING id, order_id, status, current_step, started_at`,
		orderID, model.SagaStarted, model.StepReserveInventory,
	).Scan(&saga.ID, &saga.OrderID, &saga.Status, &saga.CurrentStep, &saga.StartedAt)
	if err != nil {
		return nil, fmt.Errorf("insert saga: %w", err)
	}

	// Create the 3 steps (all start as PENDING)
	steps := []string{model.StepReserveInventory, model.StepProcessPayment, model.StepConfirmOrder}
	for _, stepName := range steps {
		var step model.SagaStep
		err = tx.QueryRowContext(ctx,
			`INSERT INTO saga_steps (saga_id, step_name, status)
			 VALUES ($1, $2, $3)
			 RETURNING id, saga_id, step_name, status`,
			saga.ID, stepName, model.StepPending,
		).Scan(&step.ID, &step.SagaID, &step.StepName, &step.Status)
		if err != nil {
			return nil, fmt.Errorf("insert step %s: %w", stepName, err)
		}
		saga.Steps = append(saga.Steps, step)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &saga, nil
}

// UpdateStepStatus marks a step as executing, completed, or failed
func (r *SagaRepository) UpdateStepStatus(ctx context.Context, sagaID, stepName string, status model.StepStatus, errMsg string) error {
	now := time.Now()
	var startedAt, completedAt *time.Time

	if status == model.StepExecuting {
		startedAt = &now
	}
	if status == model.StepCompleted || status == model.StepFailed || status == model.StepCompensated {
		completedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE saga_steps SET status = $1, error = $2, started_at = COALESCE($3, started_at), completed_at = $4
		 WHERE saga_id = $5 AND step_name = $6`,
		status, errMsg, startedAt, completedAt, sagaID, stepName,
	)
	if err != nil {
		return fmt.Errorf("update step %s: %w", stepName, err)
	}
	return nil
}

// UpdateSagaStatus marks the whole saga as completed, failed, etc.
func (r *SagaRepository) UpdateSagaStatus(ctx context.Context, sagaID string, status model.SagaStatus, currentStep string, failureReason string) error {
	var completedAt *time.Time
	if status == model.SagaCompleted || status == model.SagaFailed || status == model.SagaCompensated {
		now := time.Now()
		completedAt = &now
	}

	_, err := r.db.ExecContext(ctx,
		`UPDATE sagas SET status = $1, current_step = $2, failure_reason = $3, completed_at = $4
		 WHERE id = $5`,
		status, currentStep, failureReason, completedAt, sagaID,
	)
	if err != nil {
		return fmt.Errorf("update saga: %w", err)
	}
	return nil
}

// GetByOrderID returns the saga for a given order
func (r *SagaRepository) GetByOrderID(ctx context.Context, orderID string) (*model.Saga, error) {
	var saga model.Saga
	var completedAt sql.NullTime
	var failureReason sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, status, current_step, failure_reason, started_at, completed_at
		 FROM sagas WHERE order_id = $1`, orderID,
	).Scan(&saga.ID, &saga.OrderID, &saga.Status, &saga.CurrentStep, &failureReason, &saga.StartedAt, &completedAt)
	if err != nil {
		return nil, fmt.Errorf("query saga: %w", err)
	}

	if completedAt.Valid {
		saga.CompletedAt = &completedAt.Time
	}
	if failureReason.Valid {
		saga.FailureReason = failureReason.String
	}

	// Get steps
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, saga_id, step_name, status, COALESCE(error, ''), started_at, completed_at
		 FROM saga_steps WHERE saga_id = $1 ORDER BY created_at`, saga.ID)
	if err != nil {
		return nil, fmt.Errorf("query steps: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var step model.SagaStep
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&step.ID, &step.SagaID, &step.StepName, &step.Status, &step.Error, &startedAt, &completedAt); err != nil {
			return nil, fmt.Errorf("scan step: %w", err)
		}
		if startedAt.Valid {
			step.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			step.CompletedAt = &completedAt.Time
		}
		saga.Steps = append(saga.Steps, step)
	}

	return &saga, nil
}
