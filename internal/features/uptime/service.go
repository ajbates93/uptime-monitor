package uptime

import (
	"context"
	"database/sql"
	"the-ark/internal/features/uptime/database"
	"the-ark/internal/features/uptime/handlers"
	"the-ark/internal/features/uptime/models"
	uptimeservices "the-ark/internal/features/uptime/services"
	"the-ark/internal/server/services/mailer"

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
