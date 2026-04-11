package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Nishchal45/orderflow/services/saga-orchestrator/internal/repository"
	"github.com/rs/zerolog"
)

type SagaHandler struct {
	repo   *repository.SagaRepository
	logger zerolog.Logger
}

func NewSagaHandler(repo *repository.SagaRepository, logger zerolog.Logger) *SagaHandler {
	return &SagaHandler{repo: repo, logger: logger}
}

func (h *SagaHandler) GetSagaByOrderID(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	if orderID == "" {
		http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
		return
	}

	saga, err := h.repo.GetByOrderID(r.Context(), orderID)
	if err != nil {
		http.Error(w, `{"error":"saga not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(saga)
}
