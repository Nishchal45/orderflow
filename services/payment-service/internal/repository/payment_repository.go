package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/Nishchal45/orderflow/services/payment-service/internal/model"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) ProcessPayment(ctx context.Context, req model.ProcessPaymentRequest) (*model.Payment, error) {
	// Simulate processing delay (100-500ms)
	delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
	time.Sleep(delay)

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	// Simulate failure if requested
	if req.SimulateFailure {
		var payment model.Payment
		err := r.db.QueryRowContext(ctx,
			`INSERT INTO payments (order_id, amount, currency, status, failure_reason)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id, order_id, amount, currency, status, failure_reason, created_at, updated_at`,
			req.OrderID, req.Amount, currency, model.StatusFailed, "payment declined (simulated)",
		).Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency,
			&payment.Status, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert failed payment: %w", err)
		}
		return &payment, fmt.Errorf("payment declined (simulated)")
	}

	// Process payment (success)
	var payment model.Payment
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO payments (order_id, amount, currency, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, order_id, amount, currency, status, COALESCE(failure_reason, '') as failure_reason, created_at, updated_at`,
		req.OrderID, req.Amount, currency, model.StatusCompleted,
	).Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert payment: %w", err)
	}

	return &payment, nil
}

func (r *PaymentRepository) Refund(ctx context.Context, orderID string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.QueryRowContext(ctx,
		`UPDATE payments SET status = $1, updated_at = NOW() WHERE order_id = $2 AND status = $3
		 RETURNING id, order_id, amount, currency, status, COALESCE(failure_reason, '') as failure_reason, created_at, updated_at`,
		model.StatusRefunded, orderID, model.StatusCompleted,
	).Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no completed payment found for order: %s", orderID)
	}
	if err != nil {
		return nil, fmt.Errorf("refund payment: %w", err)
	}

	return &payment, nil
}

func (r *PaymentRepository) GetByOrderID(ctx context.Context, orderID string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.QueryRowContext(ctx,
		`SELECT id, order_id, amount, currency, status, COALESCE(failure_reason, '') as failure_reason, created_at, updated_at
		 FROM payments WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`, orderID,
	).Scan(&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment not found for order: %s", orderID)
	}
	if err != nil {
		return nil, fmt.Errorf("query payment: %w", err)
	}
	return &payment, nil
}
