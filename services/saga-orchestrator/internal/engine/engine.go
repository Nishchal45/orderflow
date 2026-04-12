package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/model"
	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/repository"
	"github.com/rs/zerolog"
)

// SagaEngine orchestrates the distributed transaction.
// It's the "head chef" — coordinates inventory, payment, and order services.
type SagaEngine struct {
	repo     *repository.SagaRepository
	producer *kafka.Producer
	logger   zerolog.Logger
	client   *http.Client

	// Service URLs
	orderServiceURL     string
	inventoryServiceURL string
	paymentServiceURL   string
}

func NewSagaEngine(
	repo *repository.SagaRepository,
	producer *kafka.Producer,
	logger zerolog.Logger,
	orderURL, inventoryURL, paymentURL string,
) *SagaEngine {
	return &SagaEngine{
		repo:                repo,
		producer:            producer,
		logger:              logger,
		client:              &http.Client{Timeout: 10 * time.Second},
		orderServiceURL:     orderURL,
		inventoryServiceURL: inventoryURL,
		paymentServiceURL:   paymentURL,
	}
}

// ExecuteSaga runs the full saga for an order.
// Steps: Reserve Inventory → Process Payment → Confirm Order
// If any step fails, compensate (undo) all previous steps in reverse.
func (e *SagaEngine) ExecuteSaga(ctx context.Context, orderData model.OrderData) error {
	e.logger.Info().Str("order_id", orderData.ID).Msg("starting saga")

	// IDEMPOTENCY CHECK: If Kafka delivers the same ORDER_CREATED event twice,
	// we must not create a second saga. Check if one already exists.
	existing, _ := e.repo.GetByOrderID(ctx, orderData.ID)
	if existing != nil {
		e.logger.Warn().
			Str("order_id", orderData.ID).
			Str("existing_saga_id", existing.ID).
			Str("status", string(existing.Status)).
			Msg("saga already exists for this order — skipping duplicate event")
		return nil
	}

	// Create saga record in database
	saga, err := e.repo.CreateSaga(ctx, orderData.ID)
	if err != nil {
		return fmt.Errorf("create saga: %w", err)
	}

	e.logger.Info().Str("saga_id", saga.ID).Str("order_id", orderData.ID).Msg("saga created")

	// ===== STEP 1: Reserve Inventory =====
	if err := e.repo.UpdateStepStatus(ctx, saga.ID, model.StepReserveInventory, model.StepExecuting, ""); err != nil {
		e.logger.Error().Err(err).Msg("failed to update step status")
	}
	if err := e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaStarted, model.StepReserveInventory, ""); err != nil {
		e.logger.Error().Err(err).Msg("failed to update saga status")
	}

	err = e.reserveInventory(ctx, orderData)
	if err != nil {
		e.logger.Error().Err(err).Str("order_id", orderData.ID).Msg("inventory reservation failed")
		e.repo.UpdateStepStatus(ctx, saga.ID, model.StepReserveInventory, model.StepFailed, err.Error())
		e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaFailed, model.StepReserveInventory, err.Error())

		// No compensation needed — nothing was done yet
		e.updateOrderStatus(ctx, orderData.ID, "REJECTED")
		e.publishEvent(ctx, events.OrderCancelled, events.TopicOrderCancelled, orderData.ID, map[string]string{"reason": err.Error()})
		return err
	}
	e.repo.UpdateStepStatus(ctx, saga.ID, model.StepReserveInventory, model.StepCompleted, "")
	e.logger.Info().Str("order_id", orderData.ID).Msg("step 1 complete: inventory reserved")

	// Update order status
	e.updateOrderStatus(ctx, orderData.ID, "PAYMENT_PENDING")

	// ===== STEP 2: Process Payment =====
	if err := e.repo.UpdateStepStatus(ctx, saga.ID, model.StepProcessPayment, model.StepExecuting, ""); err != nil {
		e.logger.Error().Err(err).Msg("failed to update step status")
	}
	if err := e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaStarted, model.StepProcessPayment, ""); err != nil {
		e.logger.Error().Err(err).Msg("failed to update saga status")
	}

	err = e.processPayment(ctx, orderData)
	if err != nil {
		e.logger.Error().Err(err).Str("order_id", orderData.ID).Msg("payment failed — starting compensation")
		e.repo.UpdateStepStatus(ctx, saga.ID, model.StepProcessPayment, model.StepFailed, err.Error())
		e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaCompensating, model.StepProcessPayment, err.Error())

		// COMPENSATE: Undo Step 1 (release inventory)
		e.compensate(ctx, saga.ID, orderData)
		return err
	}
	e.repo.UpdateStepStatus(ctx, saga.ID, model.StepProcessPayment, model.StepCompleted, "")
	e.logger.Info().Str("order_id", orderData.ID).Msg("step 2 complete: payment processed")

	// ===== STEP 3: Confirm Order =====
	e.repo.UpdateStepStatus(ctx, saga.ID, model.StepConfirmOrder, model.StepExecuting, "")
	e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaStarted, model.StepConfirmOrder, "")

	e.updateOrderStatus(ctx, orderData.ID, "CONFIRMED")
	e.repo.UpdateStepStatus(ctx, saga.ID, model.StepConfirmOrder, model.StepCompleted, "")

	// ===== SAGA COMPLETE =====
	e.repo.UpdateSagaStatus(ctx, saga.ID, model.SagaCompleted, "DONE", "")
	e.publishEvent(ctx, events.OrderConfirmed, events.TopicOrderConfirmed, orderData.ID, orderData)

	e.logger.Info().
		Str("order_id", orderData.ID).
		Str("saga_id", saga.ID).
		Msg("saga completed successfully — order confirmed!")

	return nil
}

// compensate undoes previous steps in reverse order
func (e *SagaEngine) compensate(ctx context.Context, sagaID string, orderData model.OrderData) {
	e.logger.Warn().Str("order_id", orderData.ID).Msg("compensation: releasing inventory")

	// Undo Step 1: Release inventory
	e.repo.UpdateStepStatus(ctx, sagaID, model.StepReserveInventory, model.StepCompensating, "")
	err := e.releaseInventory(ctx, orderData.ID)
	if err != nil {
		e.logger.Error().Err(err).Msg("compensation failed: could not release inventory")
		e.repo.UpdateStepStatus(ctx, sagaID, model.StepReserveInventory, model.StepFailed, "compensation failed: "+err.Error())
	} else {
		e.repo.UpdateStepStatus(ctx, sagaID, model.StepReserveInventory, model.StepCompensated, "")
		e.logger.Info().Str("order_id", orderData.ID).Msg("compensation: inventory released")
	}

	// Cancel the order
	e.updateOrderStatus(ctx, orderData.ID, "CANCELLED")
	e.repo.UpdateSagaStatus(ctx, sagaID, model.SagaCompensated, "COMPENSATED", "payment failed")
	e.publishEvent(ctx, events.OrderCancelled, events.TopicOrderCancelled, orderData.ID, map[string]string{"reason": "payment failed"})

	e.logger.Warn().Str("order_id", orderData.ID).Msg("saga compensated — order cancelled")
}

// ===== HTTP calls to other services =====

func (e *SagaEngine) reserveInventory(ctx context.Context, orderData model.OrderData) error {
	items := make([]map[string]interface{}, len(orderData.Items))
	for i, item := range orderData.Items {
		items[i] = map[string]interface{}{
			"product_id": item.ProductID,
			"quantity":   item.Quantity,
		}
	}

	body := map[string]interface{}{
		"order_id":         orderData.ID,
		"items":            items,
		"simulate_failure": orderData.SimulateInventoryFailure,
	}

	return e.callService(ctx, "POST", e.inventoryServiceURL+"/api/v1/inventory/reserve", body)
}

func (e *SagaEngine) releaseInventory(ctx context.Context, orderID string) error {
	body := map[string]interface{}{"order_id": orderID}
	return e.callService(ctx, "POST", e.inventoryServiceURL+"/api/v1/inventory/release", body)
}

func (e *SagaEngine) processPayment(ctx context.Context, orderData model.OrderData) error {
	body := map[string]interface{}{
		"order_id":         orderData.ID,
		"amount":           orderData.TotalAmount,
		"currency":         orderData.Currency,
		"simulate_failure": orderData.SimulatePaymentFailure,
	}

	return e.callService(ctx, "POST", e.paymentServiceURL+"/api/v1/payments/process", body)
}

func (e *SagaEngine) updateOrderStatus(ctx context.Context, orderID, status string) {
	body := map[string]interface{}{"status": status}
	err := e.callService(ctx, "PATCH", e.orderServiceURL+"/api/v1/orders/"+orderID+"/status", body)
	if err != nil {
		e.logger.Error().Err(err).Str("order_id", orderID).Str("status", status).Msg("failed to update order status")
	}
}

func (e *SagaEngine) callService(ctx context.Context, method, url string, body interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (e *SagaEngine) publishEvent(ctx context.Context, eventType, topic, orderID string, payload interface{}) {
	evt, err := events.NewEvent(eventType, orderID, "saga-orchestrator", payload)
	if err != nil {
		e.logger.Error().Err(err).Msg("failed to create event")
		return
	}
	data, _ := evt.Marshal()
	if err := e.producer.Publish(ctx, topic, []byte(orderID), data); err != nil {
		e.logger.Error().Err(err).Str("topic", topic).Msg("failed to publish event")
	}
}
