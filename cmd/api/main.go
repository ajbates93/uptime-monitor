package main

import (
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	port int
}

type application struct {
	config config
	logger *slog.Logger
	db     *sql.DB
}

func main() {

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialise database
	db, err := sql.Open("sqlite3", "./uptime_monitor.db")
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
		config: config{
			port: 4000,
		},
		logger: logger,
		db:     db,
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

	// Start monitoring service
	app.startMonitoring()

	err = app.serve()
	if err != nil {
		logger.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}
