package model

import "time"

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "PENDING"
	StatusCompleted PaymentStatus = "COMPLETED"
	StatusFailed    PaymentStatus = "FAILED"
	StatusRefunded  PaymentStatus = "REFUNDED"
)

type Payment struct {
	ID            string        `json:"id"`
	OrderID       string        `json:"order_id"`
	Amount        float64       `json:"amount"`
	Currency      string        `json:"currency"`
	Status        PaymentStatus `json:"status"`
	FailureReason string        `json:"failure_reason,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

type ProcessPaymentRequest struct {
	OrderID         string  `json:"order_id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	SimulateFailure bool    `json:"simulate_failure"`
}

type RefundRequest struct {
	OrderID string `json:"order_id"`
}
