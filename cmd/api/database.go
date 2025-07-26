package main

import (
	"fmt"
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

	_, err := app.db.Exec(createWebsitesTable)
	if err != nil {
		return fmt.Errorf("failed to create websites table: %w", err)
	}

	_, err = app.db.Exec(createUptimeChecksTable)
	if err != nil {
		return fmt.Errorf("failed to create uptime_checks table: %w", err)
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
