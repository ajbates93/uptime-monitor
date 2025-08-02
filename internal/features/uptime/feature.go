package uptime

import (
	"context"
	"the-ark/internal/core"
	"the-ark/internal/server/services/mailer"

	"log/slog"
)

type Feature struct {
	*core.BaseFeature
	service *Service
}

func NewFeature(logger *slog.Logger, db *core.Database, mailer mailer.Mailer, config Config) *Feature {
	service := NewService(logger, db.DB, mailer, config)

	baseFeature := core.NewBaseFeature(
		"uptime",
		"Website uptime monitoring with alerts",
		true, // Always enabled for now
		core.NewLogger(),
		db,
		config,
	)

	return &Feature{
		BaseFeature: baseFeature,
		service:     service,
	}
}

// Init initializes the uptime feature
func (f *Feature) Init(ctx context.Context) error {
	if err := f.BaseFeature.Init(ctx); err != nil {
		return err
	}

	f.service.Start(ctx)
	f.Logger().Info("Uptime feature initialized")
	return nil
}

// Routes returns the HTTP routes for the uptime feature
func (f *Feature) Routes() []core.Route {
	apiHandler := f.service.GetAPIHandler()
	webHandler := f.service.GetWebHandler()

	return []core.Route{
		// Web routes
		{Method: "GET", Path: "/uptime", Handler: webHandler.Dashboard},

		// API routes
		{Method: "GET", Path: "/uptime/api/websites", Handler: apiHandler.ListWebsites},
		{Method: "GET", Path: "/uptime/api/websites/{id}", Handler: apiHandler.GetWebsite},
		{Method: "POST", Path: "/uptime/api/websites/{id}/check", Handler: apiHandler.CheckWebsite},
		{Method: "GET", Path: "/uptime/api/dashboard", Handler: apiHandler.GetDashboard},
	}
}

// Shutdown gracefully shuts down the uptime feature
func (f *Feature) Shutdown(ctx context.Context) error {
	f.Logger().Info("Shutting down uptime feature")
	return f.BaseFeature.Shutdown(ctx)
}
