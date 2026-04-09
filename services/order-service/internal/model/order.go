package model

import (
	"time"
)

type OrderStatus string

const (
	StatusCreated             OrderStatus = "CREATED"
	StatusInventoryReserving  OrderStatus = "INVENTORY_RESERVING"
	StatusPaymentPending      OrderStatus = "PAYMENT_PENDING"
	StatusConfirmed           OrderStatus = "CONFIRMED"
	StatusShipped             OrderStatus = "SHIPPED"
	StatusRollingBack         OrderStatus = "ROLLING_BACK"
	StatusCancelled           OrderStatus = "CANCELLED"
	StatusRejected            OrderStatus = "REJECTED"
)

type Order struct {
	ID                       string      `json:"id"`
	CustomerID               string      `json:"customer_id"`
	Status                   OrderStatus `json:"status"`
	TotalAmount              float64     `json:"total_amount"`
	Currency                 string      `json:"currency"`
	Items                    []OrderItem `json:"items"`
	SimulatePaymentFailure   bool        `json:"simulate_payment_failure"`
	SimulateInventoryFailure bool        `json:"simulate_inventory_failure"`
	CreatedAt                time.Time   `json:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type OrderEvent struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	EventType string    `json:"event_type"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateOrderRequest is what the API sends to create an order.
type CreateOrderRequest struct {
	CustomerID               string            `json:"customer_id"`
	Items                    []CreateOrderItem `json:"items"`
	SimulatePaymentFailure   bool              `json:"simulate_payment_failure"`
	SimulateInventoryFailure bool              `json:"simulate_inventory_failure"`
}

type CreateOrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// ValidTransitions defines which status transitions are allowed.
var ValidTransitions = map[OrderStatus][]OrderStatus{
	StatusCreated:            {StatusInventoryReserving},
	StatusInventoryReserving: {StatusPaymentPending, StatusRejected},
	StatusPaymentPending:     {StatusConfirmed, StatusRollingBack},
	StatusRollingBack:        {StatusCancelled},
	StatusConfirmed:          {StatusShipped},
}

// CanTransitionTo checks if moving from current status to the target is valid.
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	allowed, exists := ValidTransitions[s]
	if !exists {
		return false
	}
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}
