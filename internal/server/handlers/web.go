package handlers

import (
	"net/http"
	"the-ark/internal/server/models"
	"the-ark/views/home"

	"log/slog"
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

	// Render the dashboard page
	component := home.Dashboard(dashboardWebsites)
	component.Render(r.Context(), w)
}
