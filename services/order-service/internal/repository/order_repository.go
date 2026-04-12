package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Nishchal45/orderflow/services/order-service/internal/model"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Calculate total
	var total float64
	for _, item := range req.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}

	// Insert order
	var order model.Order
	err = tx.QueryRowContext(ctx,
		`INSERT INTO orders (customer_id, status, total_amount, currency, simulate_payment_failure, simulate_inventory_failure)
		 VALUES ($1, $2, $3, 'USD', $4, $5)
		 RETURNING id, customer_id, status, total_amount, currency, simulate_payment_failure, simulate_inventory_failure, created_at, updated_at`,
		req.CustomerID, model.StatusCreated, total, req.SimulatePaymentFailure, req.SimulateInventoryFailure,
	).Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency,
		&order.SimulatePaymentFailure, &order.SimulateInventoryFailure, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert order: %w", err)
	}

	// Insert items
	for _, item := range req.Items {
		var orderItem model.OrderItem
		err = tx.QueryRowContext(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, unit_price)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id, order_id, product_id, quantity, unit_price`,
			order.ID, item.ProductID, item.Quantity, item.UnitPrice,
		).Scan(&orderItem.ID, &orderItem.OrderID, &orderItem.ProductID, &orderItem.Quantity, &orderItem.UnitPrice)
		if err != nil {
			return nil, fmt.Errorf("insert item: %w", err)
		}
		order.Items = append(order.Items, orderItem)
	}

	// Log the creation event
	payload, _ := json.Marshal(order)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO order_events (order_id, event_type, payload) VALUES ($1, $2, $3)`,
		order.ID, "ORDER_CREATED", payload,
	)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &order, nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*model.Order, error) {
	var order model.Order
	err := r.db.QueryRowContext(ctx,
		`SELECT id, customer_id, status, total_amount, currency, simulate_payment_failure, simulate_inventory_failure, created_at, updated_at
		 FROM orders WHERE id = $1`, id,
	).Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency,
		&order.SimulatePaymentFailure, &order.SimulateInventoryFailure, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query order: %w", err)
	}

	// Get items
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, order_id, product_id, quantity, unit_price FROM order_items WHERE order_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}

	return &order, nil
}

func (r *OrderRepository) List(ctx context.Context, page, limit int) ([]model.Order, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	// Get orders
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, customer_id, status, total_amount, currency, created_at, updated_at
		 FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query orders: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.Status, &o.TotalAmount, &o.Currency, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}

	return orders, total, nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, newStatus model.OrderStatus) (*model.Order, error) {
	// Use a transaction with row-level lock to prevent race conditions.
	// Without this, two concurrent requests could both read the same status,
	// both pass validation, and both write — causing invalid state transitions.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// SELECT ... FOR UPDATE locks this row until the transaction commits.
	// Any other request trying to update the same order will WAIT here.
	var order model.Order
	err = tx.QueryRowContext(ctx,
		`SELECT id, customer_id, status, total_amount, currency, simulate_payment_failure, simulate_inventory_failure, created_at, updated_at
		 FROM orders WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&order.ID, &order.CustomerID, &order.Status, &order.TotalAmount, &order.Currency,
		&order.SimulatePaymentFailure, &order.SimulateInventoryFailure, &order.CreatedAt, &order.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query order for update: %w", err)
	}

	// Validate state machine transition (now safe — row is locked)
	if !order.Status.CanTransitionTo(newStatus) {
		return nil, fmt.Errorf("invalid transition: %s → %s", order.Status, newStatus)
	}

	// Update status
	_, err = tx.ExecContext(ctx,
		`UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`,
		newStatus, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Log the event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO order_events (order_id, event_type, payload) VALUES ($1, $2, $3)`,
		id, fmt.Sprintf("STATUS_CHANGED_TO_%s", newStatus), fmt.Sprintf(`{"from":"%s","to":"%s"}`, order.Status, newStatus),
	)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	order.Status = newStatus
	return &order, nil
}
