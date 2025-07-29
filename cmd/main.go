package main

import (
	"log/slog"
	"os"
	"uptime-monitor/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create and start server
	srv := server.New(logger)

	if err := srv.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
