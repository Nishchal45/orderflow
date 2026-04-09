package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Nishchal45/orderflow/pkg/events"
	"github.com/Nishchal45/orderflow/pkg/kafka"
	"github.com/Nishchal45/orderflow/services/payment-service/internal/model"
	"github.com/Nishchal45/orderflow/services/payment-service/internal/repository"
	"github.com/rs/zerolog"
)

type PaymentHandler struct {
	repo     *repository.PaymentRepository
	producer *kafka.Producer
	logger   zerolog.Logger
}

func NewPaymentHandler(repo *repository.PaymentRepository, producer *kafka.Producer, logger zerolog.Logger) *PaymentHandler {
	return &PaymentHandler{repo: repo, producer: producer, logger: logger}
}

func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var req model.ProcessPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	payment, err := h.repo.ProcessPayment(r.Context(), req)
	if err != nil {
		h.logger.Warn().Err(err).Str("order_id", req.OrderID).Msg("payment failed")
		h.publishEvent(r, events.PaymentFailed, events.TopicPaymentFailed, req.OrderID, map[string]string{"reason": err.Error()})
		// Still return the payment record (with FAILED status)
		if payment != nil {
			respondJSON(w, http.StatusPaymentRequired, payment)
		} else {
			respondError(w, http.StatusPaymentRequired, err.Error())
		}
		return
	}

	h.logger.Info().Str("order_id", req.OrderID).Str("payment_id", payment.ID).Float64("amount", payment.Amount).Msg("payment completed")
	h.publishEvent(r, events.PaymentCompleted, events.TopicPaymentCompleted, req.OrderID, payment)

	respondJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	var req model.RefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	payment, err := h.repo.Refund(r.Context(), req.OrderID)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", req.OrderID).Msg("refund failed")
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info().Str("order_id", req.OrderID).Str("payment_id", payment.ID).Msg("payment refunded")
	h.publishEvent(r, events.PaymentRefunded, events.TopicPaymentRefunded, req.OrderID, payment)

	respondJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		respondError(w, http.StatusBadRequest, "order_id required")
		return
	}

	payment, err := h.repo.GetByOrderID(r.Context(), orderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) publishEvent(r *http.Request, eventType, topic, orderID string, payload interface{}) {
	evt, err := events.NewEvent(eventType, orderID, "payment-service", payload)
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
