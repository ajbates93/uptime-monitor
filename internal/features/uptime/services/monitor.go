package monitor

import (
	"context"
	"net/http"
	"the-ark/internal/features/uptime/models"
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

	// Store the check result
	err = db.StoreUptimeCheck(website.ID, statusCode, responseTime, isUp, errorMsg)
	if err != nil {
		m.logger.Error("Failed to store uptime check", "website_id", website.ID, "error", err)
		return
	}

	// Check if we need to send an alert
	m.handleStatusChange(website, isUp, db)
}

// Handle status changes and send alerts if needed
func (m *Monitor) handleStatusChange(website models.Website, currentIsUp bool, db Database) {
	// Get the previous status
	lastStatus, err := db.GetLastWebsiteStatus(website.ID)
	if err != nil {
		m.logger.Error("Failed to get last website status", "website_id", website.ID, "error", err)
		return
	}

	// If this is the first check, don't send an alert
	if lastStatus == nil {
		return
	}

	previousIsUp := lastStatus.Status == "up"

	// If status changed from up to down, send down alert
	if previousIsUp && !currentIsUp {
		m.sendDownAlert(website, db)
	}

	// If status changed from down to up, send recovery alert
	if !previousIsUp && currentIsUp {
		m.sendRecoveryAlert(website, db)
	}
}

// Send alert when website goes down
func (m *Monitor) sendDownAlert(website models.Website, db Database) {
	// Check if we should send an alert (avoid spam)
	shouldSend, err := db.ShouldSendAlert(website.ID, "down")
	if err != nil {
		m.logger.Error("Failed to check if should send down alert", "website_id", website.ID, "error", err)
		return
	}

	if !shouldSend {
		return
	}

	// Send the alert
	alertData := map[string]interface{}{
		"WebsiteName": website.Name,
		"WebsiteURL":  website.URL,
		"AlertType":   "down",
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	}

	err = m.mailer.Send(m.config.AlertRecipient, "website_status_alert.tmpl", alertData)
	if err != nil {
		m.logger.Error("Failed to send down alert", "website_id", website.ID, "error", err)
		return
	}

	// Record that we sent the alert
	err = db.RecordAlertSent(website.ID, "down")
	if err != nil {
		m.logger.Error("Failed to record down alert sent", "website_id", website.ID, "error", err)
	}

	m.logger.Info("Sent down alert", "website_id", website.ID, "url", website.URL)
}

// Send alert when website recovers
func (m *Monitor) sendRecoveryAlert(website models.Website, db Database) {
	// Check if we should send an alert (avoid spam)
	shouldSend, err := db.ShouldSendAlert(website.ID, "recovery")
	if err != nil {
		m.logger.Error("Failed to check if should send recovery alert", "website_id", website.ID, "error", err)
		return
	}

	if !shouldSend {
		return
	}

	// Send the alert
	alertData := map[string]interface{}{
		"WebsiteName": website.Name,
		"WebsiteURL":  website.URL,
		"AlertType":   "recovery",
		"Timestamp":   time.Now().Format("2006-01-02 15:04:05"),
	}

	err = m.mailer.Send(m.config.AlertRecipient, "website_status_alert.tmpl", alertData)
	if err != nil {
		m.logger.Error("Failed to send recovery alert", "website_id", website.ID, "error", err)
		return
	}

	// Record that we sent the alert
	err = db.RecordAlertSent(website.ID, "recovery")
	if err != nil {
		m.logger.Error("Failed to record recovery alert sent", "website_id", website.ID, "error", err)
	}

	m.logger.Info("Sent recovery alert", "website_id", website.ID, "url", website.URL)
}
