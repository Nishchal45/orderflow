package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/services/order-service/internal/model"
	"github.com/Nishchal45/orderflow/services/order-service/internal/repository"
	"github.com/rs/zerolog"
)

type OrderHandler struct {
	repo     *repository.OrderRepository
	producer *kafka.Producer
	logger   zerolog.Logger
}

func NewOrderHandler(repo *repository.OrderRepository, producer *kafka.Producer, logger zerolog.Logger) *OrderHandler {
	return &OrderHandler{
		repo:     repo,
		producer: producer,
		logger:   logger,
	}
}

// CreateOrder handles POST /orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CustomerID == "" {
		h.respondError(w, http.StatusBadRequest, "customer_id is required")
		return
	}
	if len(req.Items) == 0 {
		h.respondError(w, http.StatusBadRequest, "at least one item is required")
		return
	}

	order, err := h.repo.Create(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create order")
		h.respondError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	h.logger.Info().Str("order_id", order.ID).Str("customer", order.CustomerID).Msg("order created")

	// Publish ORDER_CREATED event to Kafka
	evt, err := events.NewEvent(events.OrderCreated, order.ID, "order-service", order)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create event")
	} else {
		data, _ := evt.Marshal()
		if err := h.producer.Publish(r.Context(), events.TopicOrderCreated, []byte(order.ID), data); err != nil {
			h.logger.Error().Err(err).Msg("failed to publish ORDER_CREATED event")
		} else {
			h.logger.Info().Str("order_id", order.ID).Msg("published ORDER_CREATED event")
		}
	}

	h.respondJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "order id is required")
		return
	}

	order, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	orders, total, err := h.repo.List(r.Context(), page, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list orders")
		h.respondError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// UpdateOrderStatus handles PATCH /orders/{id}/status
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.repo.UpdateStatus(r.Context(), id, model.OrderStatus(req.Status))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *OrderHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    status,
			"message": message,
		},
	})
}
