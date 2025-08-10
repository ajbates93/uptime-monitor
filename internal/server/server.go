package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"the-ark/internal/features/uptime"
	"the-ark/internal/server/handlers"
	"the-ark/internal/server/services/mailer"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "modernc.org/sqlite"

	"the-ark/internal/auth"
	"the-ark/internal/core"
)

type Server struct {
	config      *core.Config
	logger      *slog.Logger
	coreLogger  *core.Logger
	db          *sql.DB
	mailer      mailer.Mailer
	authService *auth.Service
	registry    *core.Registry
	server      *http.Server
}

func New(logger *slog.Logger) *Server {
	// Load configuration using the new core config system
	config, err := core.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize database
	dbPath := config.Database.Path
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	// Initialize mailer using uptime config
	uptimeConfig := config.Features.Uptime
	mailer := mailer.New(uptimeConfig.SMTP2GOAPIKey, uptimeConfig.SMTP2GOSender)

	// Initialize core components
	coreLogger := core.NewLogger()
	coreDB := core.NewDatabase(db, coreLogger)
	authService := auth.NewService(coreLogger, db, config)
	registry := core.NewRegistry(coreLogger)

	// Initialize uptime feature if enabled
	var uptimeFeature *uptime.Feature
	if config.IsFeatureEnabled("uptime") {
		uptimeConfig := uptime.Config{
			AlertRecipient: config.Features.Uptime.AlertRecipient,
		}
		uptimeFeature = uptime.NewFeature(logger, coreDB, mailer, uptimeConfig)
	}

	srv := &Server{
		config:      config,
		logger:      logger,
		coreLogger:  coreLogger,
		db:          db,
		mailer:      mailer,
		authService: authService,
		registry:    registry,
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

	// Register features if enabled
	if uptimeFeature != nil {
		if err := registry.Register(uptimeFeature); err != nil {
			logger.Error("Failed to register uptime feature", "error", err)
			os.Exit(1)
		}
	}

	// Setup routes
	srv.setupRoutes()

	return srv
}

func (s *Server) setupRoutes() {
	// Initialize portal handler
	portalHandler := handlers.NewPortalHandler(s.coreLogger, s.registry, s.authService)

	// Create router
	mux := chi.NewRouter()

	// Add middleware
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(auth.WebAuthMiddleware(s.authService)) // Add web auth middleware

	// Portal routes (main dashboard)
	mux.Get("/auth/login", portalHandler.LoginPageHandler)
	mux.Post("/auth/login", s.authService.LoginHandler)
	mux.Post("/auth/logout", s.authService.LogoutHandler)

	// Health check
	mux.Get("/health", portalHandler.HealthCheckHandler)

	// Static assets
	mux.Get("/assets/*", handlers.StaticHandler)

	// Protected routes (require authentication)
	mux.Group(func(r chi.Router) {
		r.Use(auth.RequireAuthentication)

		// Portal dashboard (protected)
		r.Get("/", portalHandler.DashboardHandler)

		// Feature routes - use the registry to get all feature routes
		routes := s.registry.GetAllRoutes()
		for _, route := range routes {
			r.Method(route.Method, route.Path, route.Handler)
		}

		// Legacy API routes (for backward compatibility)
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			})
		})

		// Future feature routes (placeholder)
		r.Route("/server", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Server monitoring coming soon", http.StatusNotImplemented)
			})
		})

		r.Route("/ssl", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "SSL certificate tracker coming soon", http.StatusNotImplemented)
			})
		})

		r.Route("/logs", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Log viewer coming soon", http.StatusNotImplemented)
			})
		})
	})

	// Create HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port),
		Handler: mux,
	}
}

func (s *Server) Start() error {
	// Initialize all features
	ctx := context.Background()
	if err := s.registry.InitAll(ctx); err != nil {
		s.logger.Error("Failed to initialize features", "error", err)
		return err
	}

	// Start HTTP server
	s.logger.Info("Starting server", "host", s.config.Server.Host, "port", s.config.Server.Port)

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Shutdown all features
	if err := s.registry.ShutdownAll(ctx); err != nil {
		s.logger.Error("Failed to shutdown features", "error", err)
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	return nil
}
