package main

import (
	"fmt"
	"time"
)

// Database operations for websites
func (app *application) initDatabase() error {
	// Create websites table
	createWebsitesTable := `
	CREATE TABLE IF NOT EXISTS websites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL UNIQUE,
		name TEXT,
		check_interval INTEGER DEFAULT 30,
		is_active BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create uptime_checks table
	createUptimeChecksTable := `
	CREATE TABLE IF NOT EXISTS uptime_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		website_id INTEGER,
		status_code INTEGER,
		response_time_ms INTEGER,
		is_up BOOLEAN,
		error_message TEXT,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (website_id) REFERENCES websites (id)
	);`

	// Create alert_history table to track when emails were sent
	createAlertHistoryTable := `
	CREATE TABLE IF NOT EXISTS alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		website_id INTEGER,
		alert_type TEXT NOT NULL,
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (website_id) REFERENCES websites (id)
	);`

	_, err := app.db.Exec(createWebsitesTable)
	if err != nil {
		return fmt.Errorf("failed to create websites table: %w", err)
	}

	_, err = app.db.Exec(createUptimeChecksTable)
	if err != nil {
		return fmt.Errorf("failed to create uptime_checks table: %w", err)
	}

	_, err = app.db.Exec(createAlertHistoryTable)
	if err != nil {
		return fmt.Errorf("failed to create alert_history table: %w", err)
	}

	return nil
}

func (app *application) seedDatabase() error {
	// Check if websites already exist
	var count int
	err := app.db.QueryRow("SELECT COUNT(*) FROM websites").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check websites count: %w", err)
	}

	// If websites already exist, don't seed
	if count > 0 {
		app.logger.Info("Database already seeded, skipping...")
		return nil
	}

	// Websites to seed
	websites := []WebsiteEntry{
		{url: "https://alexbates.dev", name: "Alex Bates Website"},
		{url: "https://pocketworks.co.uk", name: "Pocketworks"},
		{url: "https://www.anthonygordonpileofshite.com", name: "Anthony Gordon Pile of Shite"},
	}

	// Insert websites
	for _, website := range websites {
		_, err := app.db.Exec("INSERT INTO websites (url, name) VALUES (?, ?)", website.url, website.name)
		if err != nil {
			return fmt.Errorf("failed to insert website %s: %w", website.url, err)
		}
		app.logger.Info("Seeded website", "url", website.url, "name", website.name)
	}

	app.logger.Info("Database seeded successfully", "websites_added", len(websites))
	return nil
}

// Add a new website to the database
func (app *application) addWebsite(url, name string) error {
	// Check if website already exists
	var count int
	err := app.db.QueryRow("SELECT COUNT(*) FROM websites WHERE url = ?", url).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if website exists: %w", err)
	}

	if count > 0 {
		app.logger.Info("Website already exists", "url", url)
		return nil
	}

	// Insert new website
	_, err = app.db.Exec("INSERT INTO websites (url, name) VALUES (?, ?)", url, name)
	if err != nil {
		return fmt.Errorf("failed to insert website %s: %w", url, err)
	}

	app.logger.Info("Added new website", "url", url, "name", name)
	return nil
}

// Get all active websites
func (app *application) getActiveWebsites() ([]Website, error) {
	rows, err := app.db.Query("SELECT id, url, name, check_interval, is_active, created_at, updated_at FROM websites WHERE is_active = 1")
	if err != nil {
		return nil, fmt.Errorf("failed to query websites: %w", err)
	}
	defer rows.Close()

	var websites []Website
	for rows.Next() {
		var website Website
		err := rows.Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.IsActive, &website.CreatedAt, &website.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website: %w", err)
		}
		websites = append(websites, website)
	}

	return websites, nil
}

// Get single active website
func (app *application) getWebsiteByID(websiteID int) (*Website, error) {
	rows, err := app.db.Query("SELECT * FROM websites WHERE id = ? && is_active = 1", websiteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get website: %w", err)
	}

	defer rows.Close()

	var website Website
	err = rows.Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.IsActive, &website.CreatedAt, &website.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to scan website: %w", err)
	}

	return &website, nil
}

// Store uptime check result
func (app *application) storeUptimeCheck(websiteID int, statusCode int, responseTime int64, isUp bool, errorMsg string) error {
	_, err := app.db.Exec(`
		INSERT INTO uptime_checks (website_id, status_code, response_time_ms, is_up, error_message)
		VALUES (?, ?, ?, ?, ?)`,
		websiteID, statusCode, responseTime, isUp, errorMsg)

	if err != nil {
		return fmt.Errorf("failed to store uptime check: %w", err)
	}

	return nil
}

// Get uptime history for a website
func (app *application) getUptimeHistory(websiteID int, limit int) ([]WebsiteStatus, error) {
	query := `
		SELECT id, website_id, 
		       CASE WHEN is_up THEN 'up' ELSE 'down' END as status,
		       response_time_ms, status_code, error_message, checked_at
		FROM uptime_checks 
		WHERE website_id = ? 
		ORDER BY checked_at DESC 
		LIMIT ?`

	rows, err := app.db.Query(query, websiteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query uptime history: %w", err)
	}
	defer rows.Close()

	var history []WebsiteStatus
	for rows.Next() {
		var status WebsiteStatus
		err := rows.Scan(&status.ID, &status.WebsiteID, &status.Status, &status.ResponseTime, &status.StatusCode, &status.Error, &status.CheckedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan uptime status: %w", err)
		}
		history = append(history, status)
	}

	return history, nil
}

// Get the last status for a website
func (app *application) getLastWebsiteStatus(websiteID int) (*WebsiteStatus, error) {
	query := `
		SELECT id, website_id, 
		       CASE WHEN is_up THEN 'up' ELSE 'down' END as status,
		       response_time_ms, status_code, error_message, checked_at
		FROM uptime_checks 
		WHERE website_id = ? 
		ORDER BY checked_at DESC 
		LIMIT 1`

	var status WebsiteStatus
	err := app.db.QueryRow(query, websiteID).Scan(&status.ID, &status.WebsiteID, &status.Status, &status.ResponseTime, &status.StatusCode, &status.Error, &status.CheckedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get last website status: %w", err)
	}

	return &status, nil
}

// Record that an alert was sent
func (app *application) recordAlertSent(websiteID int, alertType string) error {
	_, err := app.db.Exec("INSERT INTO alert_history (website_id, alert_type) VALUES (?, ?)", websiteID, alertType)
	if err != nil {
		return fmt.Errorf("failed to record alert sent: %w", err)
	}
	return nil
}

// Check if we should send an alert based on timing rules
func (app *application) shouldSendAlert(websiteID int, alertType string) (bool, error) {
	var count int

	// For "down" alerts, check if we sent one in the last hour
	if alertType == "down" {
		err := app.db.QueryRow(`
			SELECT COUNT(*) 
			FROM alert_history 
			WHERE website_id = ? AND alert_type = 'down' 
			AND sent_at > datetime('now', '-1 hour')`, websiteID).Scan(&count)

		if err != nil && err.Error() != "sql: no rows in result set" {
			return false, fmt.Errorf("failed to check alert history: %w", err)
		}

		// Send if no recent down alert
		return count == 0, nil
	}

	// For "recovery" alerts, check if we sent one in the last 24 hours
	if alertType == "recovery" {
		err := app.db.QueryRow(`
			SELECT COUNT(*) 
			FROM alert_history 
			WHERE website_id = ? AND alert_type = 'recovery' 
			AND sent_at > datetime('now', '-24 hours')`, websiteID).Scan(&count)

		if err != nil && err.Error() != "sql: no rows in result set" {
			return false, fmt.Errorf("failed to check alert history: %w", err)
		}

		// Send if no recent recovery alert
		return count == 0, nil
	}

	return false, nil
}

// Get all websites with their current status for email templates
func (app *application) getWebsitesWithStatus() ([]map[string]interface{}, error) {
	query := `
		SELECT w.id, w.name, w.url,
		       CASE WHEN uc.is_up THEN 'up' ELSE 'down' END as status,
		       uc.checked_at
		FROM websites w
		LEFT JOIN (
			SELECT uc1.website_id, uc1.is_up, uc1.checked_at
			FROM uptime_checks uc1
			WHERE uc1.checked_at = (
				SELECT MAX(uc2.checked_at)
				FROM uptime_checks uc2
				WHERE uc2.website_id = uc1.website_id
			)
		) uc ON w.id = uc.website_id
		WHERE w.is_active = 1
		ORDER BY w.name`

	rows, err := app.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query websites with status: %w", err)
	}
	defer rows.Close()

	var websites []map[string]interface{}
	for rows.Next() {
		var id int
		var name, url, status string
		var checkedAt *time.Time

		err := rows.Scan(&id, &name, &url, &status, &checkedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website status: %w", err)
		}

		// Default to "unknown" if no status found
		if status == "" {
			status = "unknown"
		}

		website := map[string]interface{}{
			"ID":     id,
			"Name":   name,
			"URL":    url,
			"Status": status,
		}

		if checkedAt != nil {
			website["CheckedAt"] = checkedAt.Format("2006-01-02 15:04:05")
		}

		websites = append(websites, website)
	}

	return websites, nil
}
