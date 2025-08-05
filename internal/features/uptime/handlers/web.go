package handlers

import (
	"net/http"
	"strconv"
	"the-ark/internal/auth"
	"the-ark/internal/features/uptime/models"
	"the-ark/views/uptime"

	"log/slog"

	"github.com/go-chi/chi/v5"
)

type WebHandler struct {
	logger *slog.Logger
	server ServerInterface
}

func NewWebHandler(logger *slog.Logger, server ServerInterface) *WebHandler {
	return &WebHandler{
		logger: logger,
		server: server,
	}
}

func (h *WebHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := auth.GetUserFromContext(r)

	// Add defensive check for nil user
	if user == nil {
		h.logger.Error("User is nil in Dashboard handler")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

		// Add defensive check for nil status
		if status == nil {
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

	// Render the dashboard page with user
	component := uptime.Dashboard(user, dashboardWebsites)
	component.Render(r.Context(), w)
}

func (h *WebHandler) WebsiteDetail(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := auth.GetUserFromContext(r)

	// Extract website ID from URL path using Chi router
	websiteID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		h.logger.Error("Failed to extract website ID", "error", err)
		http.Error(w, "Invalid website ID", http.StatusBadRequest)
		return
	}

	// Get detailed website data
	detailData, err := h.server.GetWebsiteDetailData(websiteID)
	if err != nil {
		h.logger.Error("Failed to get website detail data", "website_id", websiteID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Render the website detail page
	component := uptime.WebsiteDetail(user, *detailData)
	component.Render(r.Context(), w)
}

func (h *WebHandler) AddSiteModal(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("AddSiteModal handler called")
	component := uptime.AddSiteModal()
	component.Render(r.Context(), w)
}
