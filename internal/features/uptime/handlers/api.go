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

func (h *APIHandler) CreateWebsite(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("CreateWebsite handler called")

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error("Failed to parse form", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	h.logger.Info("Form parsed successfully", "method", r.Method, "content_type", r.Header.Get("Content-Type"))

	name := r.FormValue("name")
	url := r.FormValue("url")
	checkIntervalStr := r.FormValue("check_interval")

	h.logger.Info("Form values", "name", name, "url", url, "check_interval", checkIntervalStr)

	if name == "" || url == "" {
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}

	checkInterval := 300 // Default to 5 minutes
	if checkIntervalStr != "" {
		if interval, err := strconv.Atoi(checkIntervalStr); err == nil {
			checkInterval = interval
		}
	}

	// Check if we're at the limit (8 sites)
	websites, err := h.server.GetActiveWebsites()
	if err != nil {
		h.logger.Error("Failed to get active websites", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if len(websites) >= 8 {
		http.Error(w, "Maximum of 8 sites allowed", http.StatusBadRequest)
		return
	}

	// Create the website
	website := models.Website{
		Name:          name,
		URL:           url,
		CheckInterval: checkInterval,
		IsActive:      true,
	}

	// Add to database (you'll need to implement this method)
	err = h.server.CreateWebsite(website)
	if err != nil {
		h.logger.Error("Failed to create website", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return success response with redirect instruction
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true, "message": "Website added successfully", "redirect": "/uptime"}`))
}

func (h *APIHandler) DeleteWebsite(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("DeleteWebsite handler called")

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Error("Failed to parse website ID", "error", err)
		http.Error(w, "Invalid website ID", http.StatusBadRequest)
		return
	}

	// Delete the website
	err = h.server.DeleteWebsite(id)
	if err != nil {
		h.logger.Error("Failed to delete website", "website_id", id, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true, "message": "Website deleted successfully"}`))
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
