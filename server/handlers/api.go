package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"log/slog"

	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	logger *slog.Logger
	server ServerInterface
}

func NewAPIHandler(logger *slog.Logger, server ServerInterface) *APIHandler {
	return &APIHandler{
		logger: logger,
		server: server,
	}
}

func (h *APIHandler) Healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *APIHandler) ListWebsites(w http.ResponseWriter, r *http.Request) {
	websites, err := h.server.GetActiveWebsites()
	if err != nil {
		h.logger.Error("Failed to get active websites", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"websites": websites})
}

func (h *APIHandler) GetWebsite(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Error("Failed to get ID from query string", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	website, err := h.server.GetWebsiteByID(id)
	if err != nil {
		h.logger.Error("Failed to get website by ID", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"website": website})
}
