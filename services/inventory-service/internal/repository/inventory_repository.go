package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nishchal45/orderflow/services/inventory-service/internal/model"
)

type InventoryRepository struct {
	db *sql.DB
}

func NewInventoryRepository(db *sql.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) Reserve(ctx context.Context, req model.ReserveRequest) ([]model.Reservation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var reservations []model.Reservation

	for _, item := range req.Items {
		// Lock the product row and check availability
		var stock, reserved int
		err := tx.QueryRowContext(ctx,
			`SELECT stock, reserved FROM products WHERE id = $1 FOR UPDATE`,
			item.ProductID,
		).Scan(&stock, &reserved)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found: %s", item.ProductID)
		}
		if err != nil {
			return nil, fmt.Errorf("query product: %w", err)
		}

		available := stock - reserved
		if available < item.Quantity {
			return nil, fmt.Errorf("insufficient stock for %s: available=%d, requested=%d", item.ProductID, available, item.Quantity)
		}

		// Increment reserved count
		_, err = tx.ExecContext(ctx,
			`UPDATE products SET reserved = reserved + $1, updated_at = NOW() WHERE id = $2`,
			item.Quantity, item.ProductID,
		)
		if err != nil {
			return nil, fmt.Errorf("update reserved: %w", err)
		}

		// Create reservation record
		var res model.Reservation
		err = tx.QueryRowContext(ctx,
			`INSERT INTO reservations (order_id, product_id, quantity, status) VALUES ($1, $2, $3, 'RESERVED') RETURNING id, order_id, product_id, quantity, status, created_at`,
			req.OrderID, item.ProductID, item.Quantity,
		).Scan(&res.ID, &res.OrderID, &res.ProductID, &res.Quantity, &res.Status, &res.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert reservation: %w", err)
		}
		reservations = append(reservations, res)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return reservations, nil
}

func (r *InventoryRepository) Release(ctx context.Context, orderID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Read all reservations into memory first
	rows, err := tx.QueryContext(ctx,
		`SELECT product_id, quantity FROM reservations WHERE order_id = $1 AND status = 'RESERVED'`, orderID)
	if err != nil {
		return fmt.Errorf("query reservations: %w", err)
	}

	type item struct {
		productID string
		quantity  int
	}
	var items []item
	for rows.Next() {
		var i item
		if err := rows.Scan(&i.productID, &i.quantity); err != nil {
			rows.Close()
			return fmt.Errorf("scan reservation: %w", err)
		}
		items = append(items, i)
	}
	rows.Close() // Close rows BEFORE executing updates

	// Step 2: Now update each product's reserved count
	for _, i := range items {
		_, err = tx.ExecContext(ctx,
			`UPDATE products SET reserved = GREATEST(reserved - $1, 0), updated_at = NOW() WHERE id = $2`,
			i.quantity, i.productID,
		)
		if err != nil {
			return fmt.Errorf("update reserved for %s: %w", i.productID, err)
		}
	}

	// Step 3: Mark reservations as released
	_, err = tx.ExecContext(ctx,
		`UPDATE reservations SET status = 'RELEASED' WHERE order_id = $1 AND status = 'RESERVED'`, orderID)
	if err != nil {
		return fmt.Errorf("update reservations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (r *InventoryRepository) GetStock(ctx context.Context, productID string) (*model.Product, error) {
	var p model.Product
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, stock, reserved, stock - reserved as available, updated_at FROM products WHERE id = $1`,
		productID,
	).Scan(&p.ID, &p.Name, &p.Stock, &p.Reserved, &p.Available, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("product not found: %s", productID)
	}
	if err != nil {
		return nil, fmt.Errorf("query product: %w", err)
	}
	return &p, nil
}
