package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/services/inventory-service/internal/model"
	"github.com/Nishchal45/orderflow/services/inventory-service/internal/repository"
	"github.com/rs/zerolog"
)

type InventoryHandler struct {
	repo     *repository.InventoryRepository
	producer *kafka.Producer
	logger   zerolog.Logger
}

func NewInventoryHandler(repo *repository.InventoryRepository, producer *kafka.Producer, logger zerolog.Logger) *InventoryHandler {
	return &InventoryHandler{repo: repo, producer: producer, logger: logger}
}

func (h *InventoryHandler) ReserveInventory(w http.ResponseWriter, r *http.Request) {
	var req model.ReserveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SimulateFailure {
		h.logger.Warn().Str("order_id", req.OrderID).Msg("simulating inventory failure")
		h.publishEvent(r, events.InventoryFailed, events.TopicInventoryFailed, req.OrderID, map[string]string{"reason": "simulated failure"})
		respondError(w, http.StatusConflict, "insufficient stock (simulated)")
		return
	}

	reservations, err := h.repo.Reserve(r.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", req.OrderID).Msg("failed to reserve inventory")
		h.publishEvent(r, events.InventoryFailed, events.TopicInventoryFailed, req.OrderID, map[string]string{"reason": err.Error()})
		respondError(w, http.StatusConflict, err.Error())
		return
	}

	h.logger.Info().Str("order_id", req.OrderID).Int("items", len(reservations)).Msg("inventory reserved")
	h.publishEvent(r, events.InventoryReserved, events.TopicInventoryReserved, req.OrderID, reservations)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":      true,
		"message":      "inventory reserved",
		"reservations": reservations,
	})
}

func (h *InventoryHandler) ReleaseInventory(w http.ResponseWriter, r *http.Request) {
	var req model.ReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.repo.Release(r.Context(), req.OrderID); err != nil {
		h.logger.Error().Err(err).Str("order_id", req.OrderID).Msg("failed to release inventory")
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.logger.Info().Str("order_id", req.OrderID).Msg("inventory released")
	h.publishEvent(r, events.InventoryReleased, events.TopicInventoryReleased, req.OrderID, nil)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "inventory released",
	})
}

func (h *InventoryHandler) GetStock(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "product id required")
		return
	}

	product, err := h.repo.GetStock(r.Context(), productID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, product)
}

func (h *InventoryHandler) publishEvent(r *http.Request, eventType, topic, orderID string, payload interface{}) {
	evt, err := events.NewEvent(eventType, orderID, "inventory-service", payload)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create event")
		return
	}
	data, _ := evt.Marshal()
	if err := h.producer.Publish(r.Context(), topic, []byte(orderID), data); err != nil {
		h.logger.Error().Err(err).Str("topic", topic).Msg("failed to publish event")
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{"error": map[string]interface{}{"code": status, "message": message}})
}
