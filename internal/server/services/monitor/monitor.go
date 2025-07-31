package monitor

import (
	"context"
	"net/http"
	"the-ark/internal/server/models"
	"the-ark/internal/server/services/mailer"
	"time"

	"log/slog"
)

// WebsiteEntry is used for seeding the database
type WebsiteEntry struct {
	url  string
	name string
}

type Monitor struct {
	logger *slog.Logger
	mailer mailer.Mailer
	config MonitorConfig
}

type MonitorConfig struct {
	AlertRecipient string
}

// Database interface for monitoring operations
type Database interface {
	GetActiveWebsites() ([]models.Website, error)
	GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error)
	StoreUptimeCheck(websiteID int, statusCode int, responseTime int64, isUp bool, errorMsg string) error
	ShouldSendAlert(websiteID int, alertType string) (bool, error)
	RecordAlertSent(websiteID int, alertType string) error
}

func New(logger *slog.Logger, mailer mailer.Mailer, config MonitorConfig) *Monitor {
	return &Monitor{
		logger: logger,
		mailer: mailer,
		config: config,
	}
}

// Start monitoring in a goroutine
func (m *Monitor) Start(ctx context.Context, db Database) {
	go m.run(ctx, db)
}

// Run the monitoring loop
func (m *Monitor) run(ctx context.Context, db Database) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Do initial check
	m.checkAllWebsites(db)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Monitoring stopped")
			return
		case <-ticker.C:
			m.checkAllWebsites(db)
		}
	}
}

// Check all websites and store results
func (m *Monitor) checkAllWebsites(db Database) {
	websites, err := db.GetActiveWebsites()
	if err != nil {
		m.logger.Error("Failed to get active websites", "error", err)
		return
	}

	for _, website := range websites {
		m.CheckWebsite(website, db)
	}
}

// Check a single website
func (m *Monitor) CheckWebsite(website models.Website, db Database) {
	start := time.Now()
	resp, err := http.Get(website.URL)
	responseTime := time.Since(start).Milliseconds()

	var statusCode int
	var isUp bool
	var errorMsg string

	if err != nil {
		errorMsg = err.Error()
		isUp = false
		m.logger.Error("Website check failed", "url", website.URL, "error", err)
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		isUp = resp.StatusCode == http.StatusOK
	}

	// Store result in database
	err = db.StoreUptimeCheck(website.ID, statusCode, responseTime, isUp, errorMsg)
	if err != nil {
		m.logger.Error("Failed to store uptime check", "error", err)
		return
	}

	// Check for status changes and send alerts
	m.handleStatusChange(website, isUp, db)
}

// Handle status changes and send appropriate alerts
func (m *Monitor) handleStatusChange(website models.Website, currentIsUp bool, db Database) {
	// Get the previous status
	lastStatus, err := db.GetLastWebsiteStatus(website.ID)
	if err != nil {
		// If no previous status, this is the first check
		if currentIsUp {
			// Site is up on first check, no alert needed
			return
		} else {
			// Site is down on first check, send initial down alert
			m.logger.Info("First check - site is down, sending initial alert", "website", website.Name)
			m.sendDownAlert(website, db)
			return
		}
	}

	// Check if status changed
	previousIsUp := lastStatus.Status == "up"

	if previousIsUp && !currentIsUp {
		// Website went from up to down
		m.logger.Info("Website went down", "url", website.URL, "name", website.Name)
		m.sendDownAlert(website, db)
	} else if !previousIsUp && currentIsUp {
		// Website went from down to up
		m.logger.Info("Website recovered", "url", website.URL, "name", website.Name)
		m.sendRecoveryAlert(website, db)
	} else if !currentIsUp {
		// Website is still down, check if we should send a reminder alert
		m.sendDownAlert(website, db)
	}
}

// Send down alert for a website
func (m *Monitor) sendDownAlert(website models.Website, db Database) {
	shouldSend, err := db.ShouldSendAlert(website.ID, "down")
	if err != nil {
		m.logger.Error("Failed to check if should send down alert", "error", err)
		return
	}

	if !shouldSend {
		m.logger.Debug("Skipping down alert - sent recently", "website", website.Name)
		return
	}

	// Send email alert
	alertData := map[string]interface{}{
		"WebsiteName": website.Name,
		"WebsiteURL":  website.URL,
		"AlertType":   "down",
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	}

	err = m.mailer.Send(m.config.AlertRecipient, "website_status_alert.tmpl", alertData)
	if err != nil {
		m.logger.Error("Failed to send down alert email", "error", err)
		return
	}

	// Record that alert was sent
	err = db.RecordAlertSent(website.ID, "down")
	if err != nil {
		m.logger.Error("Failed to record down alert sent", "error", err)
	}

	m.logger.Info("Sent down alert", "website", website.Name, "recipient", m.config.AlertRecipient)
}

// Send recovery alert for a website
func (m *Monitor) sendRecoveryAlert(website models.Website, db Database) {
	shouldSend, err := db.ShouldSendAlert(website.ID, "recovery")
	if err != nil {
		m.logger.Error("Failed to check if should send recovery alert", "error", err)
		return
	}

	if !shouldSend {
		m.logger.Debug("Skipping recovery alert - sent recently", "website", website.Name)
		return
	}

	// Send email alert
	alertData := map[string]interface{}{
		"WebsiteName": website.Name,
		"WebsiteURL":  website.URL,
		"AlertType":   "recovery",
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	}

	err = m.mailer.Send(m.config.AlertRecipient, "website_status_alert.tmpl", alertData)
	if err != nil {
		m.logger.Error("Failed to send recovery alert email", "error", err)
		return
	}

	// Record that alert was sent
	err = db.RecordAlertSent(website.ID, "recovery")
	if err != nil {
		m.logger.Error("Failed to record recovery alert sent", "error", err)
	}

	m.logger.Info("Sent recovery alert", "website", website.Name, "recipient", m.config.AlertRecipient)
}
