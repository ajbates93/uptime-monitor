package main

import (
	"net/http"
	"uptime-monitor/internal/models"
)

func (app *application) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	websites, err := app.getActiveWebsites()
	if err != nil {
		app.logger.Error("Failed to get active websites", "error", err)
		app.serverErrorResponse(w, r, err)
		return
	}

	// Convert to DashboardWebsite for the web interface
	dashboardWebsites := make([]models.DashboardWebsite, len(websites))
	for i, website := range websites {
		// Get the latest status for this website
		status, err := app.getLastWebsiteStatus(website.ID)
		if err != nil {
			app.logger.Error("Failed to get website status", "website_id", website.ID, "error", err)
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
	component := dashboardPage(dashboardWebsites)
	component.Render(r.Context(), w)
}
