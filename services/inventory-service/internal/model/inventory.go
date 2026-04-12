package model

import "time"

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Stock     int       `json:"stock"`
	Reserved  int       `json:"reserved"`
	Available int       `json:"available"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Reservation struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ReserveRequest struct {
	OrderID         string        `json:"order_id"`
	Items           []ReserveItem `json:"items"`
	SimulateFailure bool          `json:"simulate_failure"`
}

type ReserveItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ReleaseRequest struct {
	OrderID string `json:"order_id"`
}
