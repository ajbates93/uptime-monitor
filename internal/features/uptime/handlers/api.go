package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"the-ark/internal/features/uptime/models"
	"the-ark/views/uptime"

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

// GetDashboard returns the dashboard HTML for HTMX
func (h *APIHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	websites, err := h.server.GetActiveWebsites()
	if err != nil {
		h.logger.Error("Failed to get active websites", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Convert to DashboardWebsite for the web interface
	dashboardWebsites := make([]models.DashboardWebsite, len(websites))
	for i, website := range websites {
		// Get the latest status for this website
		status, err := h.server.GetLastWebsiteStatus(website.ID)
		if err != nil {
			h.logger.Error("Failed to get website status", "website_id", website.ID, "error", err)
			// Continue with unknown status
			dashboardWebsites[i] = models.DashboardWebsite{
				Website:   website,
				Status:    "unknown",
				CheckedAt: nil,
			}
			continue
		}

		dashboardWebsites[i] = models.DashboardWebsite{
			Website:   website,
			Status:    status.Status,
			CheckedAt: &status.CheckedAt,
		}
	}

	// Render the new card format for HTMX
	for _, website := range dashboardWebsites {
		component := uptime.UptimeWebsiteCard(website)
		component.Render(r.Context(), w)
	}
}

func (h *APIHandler) CheckWebsite(w http.ResponseWriter, r *http.Request) {
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

	// Perform the check
	err = h.server.CheckWebsite(*website)
	if err != nil {
		h.logger.Error("Failed to check website", "website_id", id, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get the updated status after the check
	status, err := h.server.GetLastWebsiteStatus(website.ID)
	if err != nil {
		h.logger.Error("Failed to get updated website status", "website_id", id, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create dashboard website with updated status
	dashboardWebsite := models.DashboardWebsite{
		Website:   *website,
		Status:    status.Status,
		CheckedAt: &status.CheckedAt,
	}

	// Render the updated card
	component := uptime.UptimeWebsiteCard(dashboardWebsite)
	component.Render(r.Context(), w)
}
