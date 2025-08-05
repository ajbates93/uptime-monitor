package uptime

import (
	"context"
	"database/sql"
	"fmt"
	"the-ark/internal/features/uptime/database"
	"the-ark/internal/features/uptime/handlers"
	"the-ark/internal/features/uptime/models"
	uptimeservices "the-ark/internal/features/uptime/services"
	"the-ark/internal/server/services/mailer"
	"time"

	"log/slog"
)

type Service struct {
	logger     *slog.Logger
	db         *sql.DB
	monitor    *uptimeservices.Monitor
	apiHandler *handlers.APIHandler
	webHandler *handlers.WebHandler
}

type Config struct {
	AlertRecipient string
}

func NewService(logger *slog.Logger, db *sql.DB, mailer mailer.Mailer, config Config) *Service {
	dbService := database.NewDatabaseService(db)

	monitorConfig := uptimeservices.MonitorConfig{
		AlertRecipient: config.AlertRecipient,
	}
	monitor := uptimeservices.New(logger, mailer, monitorConfig)

	apiHandler := handlers.NewAPIHandler(logger, dbService)
	webHandler := handlers.NewWebHandler(logger, dbService)

	return &Service{
		logger:     logger,
		db:         db,
		monitor:    monitor,
		apiHandler: apiHandler,
		webHandler: webHandler,
	}
}

// Start starts the uptime monitoring service
func (s *Service) Start(ctx context.Context) {
	s.logger.Info("Starting uptime monitoring service")
	dbService := database.NewDatabaseService(s.db)
	s.monitor.Start(ctx, dbService)
}

// GetAPIHandler returns the API handler for routing
func (s *Service) GetAPIHandler() *handlers.APIHandler {
	return s.apiHandler
}

// GetWebHandler returns the web handler for routing
func (s *Service) GetWebHandler() *handlers.WebHandler {
	return s.webHandler
}

// GetActiveWebsites retrieves all active websites
func (s *Service) GetActiveWebsites() ([]models.Website, error) {
	dbService := database.NewDatabaseService(s.db)
	return dbService.GetActiveWebsites()
}

// GetWebsiteByID retrieves a specific website by ID
func (s *Service) GetWebsiteByID(websiteID int) (*models.Website, error) {
	dbService := database.NewDatabaseService(s.db)
	return dbService.GetWebsiteByID(websiteID)
}

// GetLastWebsiteStatus retrieves the most recent status for a website
func (s *Service) GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error) {
	dbService := database.NewDatabaseService(s.db)
	return dbService.GetLastWebsiteStatus(websiteID)
}

// CheckWebsite performs a manual check of a website
func (s *Service) CheckWebsite(website models.Website) error {
	dbService := database.NewDatabaseService(s.db)
	s.monitor.CheckWebsite(website, dbService)
	return nil // The monitor's CheckWebsite doesn't return anything, so we return nil
}

// GetWebsiteDetailData retrieves all data needed for the detailed website view
func (s *Service) GetWebsiteDetailData(websiteID int) (*models.WebsiteDetailData, error) {
	dbService := database.NewDatabaseService(s.db)

	// Get website
	website, err := dbService.GetWebsiteByID(websiteID)
	if err != nil {
		return nil, err
	}

	// Get last status
	lastStatus, err := dbService.GetLastWebsiteStatus(websiteID)
	if err != nil {
		return nil, err
	}

	// Get uptime stats for different periods
	uptimeStats, err := s.getUptimeStats(websiteID)
	if err != nil {
		return nil, err
	}

	// Get incidents
	incidents, err := dbService.GetIncidents(websiteID, 10)
	if err != nil {
		return nil, err
	}

	// Get average response time
	avgResponse, err := dbService.GetAverageResponseTime(websiteID, 24*30) // 30 days
	if err != nil {
		return nil, err
	}

	return &models.WebsiteDetailData{
		Website:     *website,
		LastStatus:  lastStatus,
		UptimeStats: uptimeStats,
		Incidents:   incidents,
		AvgResponse: avgResponse,
	}, nil
}

// getUptimeStats calculates uptime statistics for different time periods
func (s *Service) getUptimeStats(websiteID int) ([]models.UptimeStats, error) {
	dbService := database.NewDatabaseService(s.db)

	periods := []struct {
		hours int
		label string
	}{
		{24, "24h"},
		{24 * 7, "7d"},
		{24 * 30, "30d"},
		{24 * 365, "365d"},
	}

	var stats []models.UptimeStats
	for _, period := range periods {
		percentage, upChecks, downChecks, err := dbService.GetUptimePercentage(websiteID, period.hours)
		if err != nil {
			return nil, err
		}

		// Get incident count for this period
		incidents, err := dbService.GetIncidents(websiteID, 100) // Get more incidents to count
		if err != nil {
			return nil, err
		}

		// Count incidents in this period
		incidentCount := 0
		var totalDowntime time.Duration
		for _, incident := range incidents {
			if time.Since(incident.StartedAt) <= time.Duration(period.hours)*time.Hour {
				incidentCount++
				totalDowntime += incident.Duration
			}
		}

		stats = append(stats, models.UptimeStats{
			WebsiteID:     websiteID,
			Period:        period.label,
			Percentage:    percentage,
			UpChecks:      upChecks,
			DownChecks:    downChecks,
			TotalChecks:   upChecks + downChecks,
			IncidentCount: incidentCount,
			Downtime:      formatDuration(totalDowntime),
		})
	}

	return stats, nil
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.Hours())
	}
	return fmt.Sprintf("%.0fd", d.Hours()/24)
}
