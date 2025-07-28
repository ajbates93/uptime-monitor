package main

import (
	"database/sql"
	"log/slog"
	"os"
	"uptime-monitor/internal/mailer"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	port    int
	smtp2go struct {
		apiKey string
		sender string
	}
	alerts struct {
		recipient string
	}
}

type application struct {
	config config
	logger *slog.Logger
	db     *sql.DB
	mailer mailer.Mailer
}

func main() {
	// Load .env file if it exists
	godotenv.Load()

	var cfg config

	// Load configuration from environment variables
	cfg.port = 4000

	// SMTP2GO configuration
	cfg.smtp2go.apiKey = getEnvOrDefault("SMTP2GO_API_KEY", "")
	cfg.smtp2go.sender = getEnvOrDefault("SMTP2GO_SENDER", "Uptime Monitor <uptime@alexbates.dev>")

	// Alert configuration
	cfg.alerts.recipient = getEnvOrDefault("ALERT_RECIPIENT", "ajbates93@gmail.com")

	// Validate required environment variables
	if cfg.smtp2go.apiKey == "" {
		slog.Error("SMTP2GO_API_KEY environment variable is required")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialise database
	dbPath := getEnvOrDefault("DB_PATH", "./uptime_monitor.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
		db:     db,
		mailer: mailer.New(cfg.smtp2go.apiKey, cfg.smtp2go.sender),
	}

	// Initialise database tables
	if err := app.initDatabase(); err != nil {
		logger.Error("Failed to initialise database", "error", err)
		os.Exit(1)
	}

	// Seed database with initial websites
	if err := app.seedDatabase(); err != nil {
		logger.Error("Failed to seed database", "error", err)
		os.Exit(1)
	}

	// Add any new websites that might not be in the seed
	// if err := app.addWebsite("https://airshift.co.uk", "AirShift"); err != nil {
	// 	logger.Error("Failed to add new website", "error", err)
	// 	os.Exit(1)
	// }

	// Start monitoring service
	app.startMonitoring()

	err = app.serve()
	if err != nil {
		logger.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}

// getEnvOrDefault returns the environment variable value or a default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
