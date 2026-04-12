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
	h.logger.Info().Str("path", r.URL.Path).Str("order_id", orderID).Msg("saga lookup request")

	if orderID == "" {
		// Fallback: extract from URL path manually (for reverse proxy)
		parts := splitPath(r.URL.Path)
		if len(parts) > 0 {
			orderID = parts[len(parts)-1]
		}
	}

	if orderID == "" {
		http.Error(w, `{"error":"order_id required"}`, http.StatusBadRequest)
		return
	}

	saga, err := h.repo.GetByOrderID(r.Context(), orderID)
	if err != nil {
		h.logger.Error().Err(err).Str("order_id", orderID).Msg("saga not found")
		http.Error(w, `{"error":"saga not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(saga)
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
