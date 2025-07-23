package main

import (
	"net/http"
	"time"
)

// WebsiteEntry is used for seeding the database
type WebsiteEntry struct {
	url  string
	name string
}

// Check all websites and store results
func (app *application) checkAllWebsites() {
	websites, err := app.getActiveWebsites()
	if err != nil {
		app.logger.Error("Failed to get active websites", "error", err)
		return
	}

	for _, website := range websites {
		app.checkWebsite(website)
	}
}

// Check a single website
func (app *application) checkWebsite(website Website) {
	start := time.Now()
	resp, err := http.Get(website.URL)
	responseTime := time.Since(start).Milliseconds()

	var statusCode int
	var isUp bool
	var errorMsg string

	if err != nil {
		errorMsg = err.Error()
		isUp = false
		app.logger.Error("Website check failed", "url", website.URL, "error", err)
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		isUp = resp.StatusCode == http.StatusOK
		app.logger.Info("Website check completed",
			"url", website.URL,
			"status", statusCode,
			"response_time_ms", responseTime,
			"is_up", isUp)
	}

	// Store result in database
	err = app.storeUptimeCheck(website.ID, statusCode, responseTime, isUp, errorMsg)
	if err != nil {
		app.logger.Error("Failed to store uptime check", "error", err)
	}
}

// Start the monitoring service
func (app *application) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				app.checkAllWebsites()
			case <-done:
				return
			}
		}
	}()

	app.logger.Info("Monitoring service started")
}
