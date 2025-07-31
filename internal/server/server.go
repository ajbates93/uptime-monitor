package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"the-ark/internal/server/handlers"
	"the-ark/internal/server/services/mailer"
	"the-ark/internal/server/services/monitor"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	config  Config
	logger  *slog.Logger
	db      *sql.DB
	mailer  mailer.Mailer
	monitor *monitor.Monitor
	server  *http.Server
}

type Config struct {
	Port           int
	SMTP2GOAPIKey  string
	SMTP2GOSender  string
	AlertRecipient string
	DBPath         string
}

func New(logger *slog.Logger) *Server {
	config := loadConfig()

	// Initialize database
	dbPath := config.DBPath
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	// Initialize mailer
	mailer := mailer.New(config.SMTP2GOAPIKey, config.SMTP2GOSender)

	// Initialize monitor
	monitorConfig := monitor.MonitorConfig{
		AlertRecipient: config.AlertRecipient,
	}
	monitor := monitor.New(logger, mailer, monitorConfig)

	srv := &Server{
		config:  config,
		logger:  logger,
		db:      db,
		mailer:  mailer,
		monitor: monitor,
	}

	// Initialize database tables
	if err := srv.initDatabase(); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Seed database with initial websites
	if err := srv.seedDatabase(); err != nil {
		logger.Error("Failed to seed database", "error", err)
		os.Exit(1)
	}

	// Setup HTTP server
	srv.setupRoutes()

	return srv
}

func (s *Server) setupRoutes() {
	// Initialize handlers
	webHandler := handlers.NewWebHandler(s.logger, s)
	apiHandler := handlers.NewAPIHandler(s.logger, s)

	// Create router
	mux := chi.NewRouter()

	// Add middleware
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)

	// Web routes
	mux.Get("/", webHandler.Dashboard)

	// Static assets
	mux.Get("/assets/*", handlers.StaticHandler)

	// API routes
	mux.Group(func(r chi.Router) {
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/healthcheck", apiHandler.Healthcheck)
			r.Get("/dashboard", apiHandler.GetDashboard)
			r.Get("/websites", apiHandler.ListWebsites)
			r.Get("/websites/{id}", apiHandler.GetWebsite)
			r.Post("/websites/{id}/check", apiHandler.CheckWebsite)
		})
	})

	// Create HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}
}

func (s *Server) Start() error {
	// Start monitoring in background
	ctx := context.Background()
	s.monitor.Start(ctx, s)

	// Start HTTP server
	s.logger.Info("Starting server", "port", s.config.Port)

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}

func loadConfig() Config {
	config := Config{
		Port:           4000,
		SMTP2GOAPIKey:  getEnvOrDefault("SMTP2GO_API_KEY", ""),
		SMTP2GOSender:  getEnvOrDefault("SMTP2GO_SENDER", "Uptime Monitor <uptime@alexbates.dev>"),
		AlertRecipient: getEnvOrDefault("ALERT_RECIPIENT", "ajbates93@gmail.com"),
		DBPath:         getEnvOrDefault("DB_PATH", "./uptime_monitor.db"),
	}

	// Validate required environment variables
	if config.SMTP2GOAPIKey == "" {
		panic("SMTP2GO_API_KEY environment variable is required")
	}

	return config
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
