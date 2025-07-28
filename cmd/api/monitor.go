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
	}

	// Store result in database
	err = app.storeUptimeCheck(website.ID, statusCode, responseTime, isUp, errorMsg)
	if err != nil {
		app.logger.Error("Failed to store uptime check", "error", err)
		return
	}

	// Check for status changes and send alerts
	app.handleStatusChange(website, isUp)
}

// Handle status changes and send appropriate alerts
func (app *application) handleStatusChange(website Website, currentIsUp bool) {
	// Get the previous status
	lastStatus, err := app.getLastWebsiteStatus(website.ID)
	if err != nil {
		// If no previous status, this is the first check
		if currentIsUp {
			// Site is up on first check, no alert needed
			return
		} else {
			// Site is down on first check, send initial down alert
			app.logger.Info("First check - site is down, sending initial alert", "website", website.Name)
			app.sendDownAlert(website)
			return
		}
	}

	// Check if status changed
	previousIsUp := lastStatus.Status == "up"

	if previousIsUp && !currentIsUp {
		// Website went from up to down
		app.logger.Info("Website went down", "url", website.URL, "name", website.Name)
		app.sendDownAlert(website)
	} else if !previousIsUp && currentIsUp {
		// Website went from down to up
		app.logger.Info("Website recovered", "url", website.URL, "name", website.Name)
		app.sendRecoveryAlert(website)
	} else if !currentIsUp {
		// Website is still down, check if we should send a reminder alert
		app.sendDownAlert(website)
	}
}

// Send down alert for a website
func (app *application) sendDownAlert(website Website) {
	shouldSend, err := app.shouldSendAlert(website.ID, "down")
	if err != nil {
		app.logger.Error("Failed to check if should send down alert", "error", err)
		return
	}

	if !shouldSend {
		app.logger.Info("Skipping down alert - too recent", "website", website.Name)
		return
	}

	// Get all websites with current status for the email
	websites, err := app.getWebsitesWithStatus()
	if err != nil {
		app.logger.Error("Failed to get websites with status", "error", err)
		return
	}

	// Prepare email data
	emailData := map[string]interface{}{
		"websites":  websites,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"alertType": "down",
		"downSite":  website.Name,
	}

	// Send email
	app.logger.Info("Sending down alert email", "website", website.Name, "recipient", app.config.alerts.recipient)
	err = app.mailer.Send(app.config.alerts.recipient, "website_status_alert.tmpl", emailData)
	if err != nil {
		app.logger.Error("Failed to send down alert email", "error", err)
		return
	}

	// Record that we sent the alert
	err = app.recordAlertSent(website.ID, "down")
	if err != nil {
		app.logger.Error("Failed to record down alert sent", "error", err)
	}

	app.logger.Info("Down alert sent successfully", "website", website.Name)
}

// Send recovery alert for a website
func (app *application) sendRecoveryAlert(website Website) {
	shouldSend, err := app.shouldSendAlert(website.ID, "recovery")
	if err != nil {
		app.logger.Error("Failed to check if should send recovery alert", "error", err)
		return
	}

	if !shouldSend {
		app.logger.Info("Skipping recovery alert - too recent", "website", website.Name)
		return
	}

	// Get all websites with current status for the email
	websites, err := app.getWebsitesWithStatus()
	if err != nil {
		app.logger.Error("Failed to get websites with status", "error", err)
		return
	}

	// Prepare email data
	emailData := map[string]interface{}{
		"websites":      websites,
		"timestamp":     time.Now().Format("2006-01-02 15:04:05"),
		"alertType":     "recovery",
		"recoveredSite": website.Name,
	}

	// Send email
	err = app.mailer.Send(app.config.alerts.recipient, "website_status_alert.tmpl", emailData)
	if err != nil {
		app.logger.Error("Failed to send recovery alert email", "error", err)
		return
	}

	// Record that we sent the alert
	err = app.recordAlertSent(website.ID, "recovery")
	if err != nil {
		app.logger.Error("Failed to record recovery alert sent", "error", err)
	}

	app.logger.Info("Recovery alert sent", "website", website.Name)
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
