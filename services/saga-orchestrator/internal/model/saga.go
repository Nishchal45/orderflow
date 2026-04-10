package model

import "time"

type SagaStatus string

const (
	SagaStarted      SagaStatus = "STARTED"
	SagaCompleted    SagaStatus = "COMPLETED"
	SagaFailed       SagaStatus = "FAILED"
	SagaCompensating SagaStatus = "COMPENSATING"
	SagaCompensated  SagaStatus = "COMPENSATED"
)

type StepStatus string

const (
	StepPending      StepStatus = "PENDING"
	StepExecuting    StepStatus = "EXECUTING"
	StepCompleted    StepStatus = "COMPLETED"
	StepFailed       StepStatus = "FAILED"
	StepCompensating StepStatus = "COMPENSATING"
	StepCompensated  StepStatus = "COMPENSATED"
)

type Saga struct {
	ID            string     `json:"id"`
	OrderID       string     `json:"order_id"`
	Status        SagaStatus `json:"status"`
	CurrentStep   string     `json:"current_step"`
	FailureReason string     `json:"failure_reason,omitempty"`
	Steps         []SagaStep `json:"steps"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

type SagaStep struct {
	ID          string     `json:"id"`
	SagaID      string     `json:"saga_id"`
	StepName    string     `json:"step_name"`
	Status      StepStatus `json:"status"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// The 3 saga steps in order
const (
	StepReserveInventory = "RESERVE_INVENTORY"
	StepProcessPayment   = "PROCESS_PAYMENT"
	StepConfirmOrder     = "CONFIRM_ORDER"
)

// OrderData is what we get from the ORDER_CREATED Kafka event
type OrderData struct {
	ID                       string      `json:"id"`
	CustomerID               string      `json:"customer_id"`
	TotalAmount              float64     `json:"total_amount"`
	Currency                 string      `json:"currency"`
	Items                    []OrderItem `json:"items"`
	SimulatePaymentFailure   bool        `json:"simulate_payment_failure"`
	SimulateInventoryFailure bool        `json:"simulate_inventory_failure"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}
