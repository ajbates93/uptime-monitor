package server

import (
	"database/sql"
	"fmt"
	"the-ark/internal/server/models"
	"the-ark/internal/server/services/monitor"
)

// Database operations for websites
func (s *Server) initDatabase() error {
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

	_, err := s.db.Exec(createWebsitesTable)
	if err != nil {
		return fmt.Errorf("failed to create websites table: %w", err)
	}

	_, err = s.db.Exec(createUptimeChecksTable)
	if err != nil {
		return fmt.Errorf("failed to create uptime_checks table: %w", err)
	}

	_, err = s.db.Exec(createAlertHistoryTable)
	if err != nil {
		return fmt.Errorf("failed to create alert_history table: %w", err)
	}

	return nil
}

func (s *Server) seedDatabase() error {
	// Check if websites already exist
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM websites").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check websites count: %w", err)
	}

	// If websites already exist, don't seed
	if count > 0 {
		s.logger.Info("Database already seeded, skipping...")
		return nil
	}

	// Websites to seed
	websites := []struct {
		url  string
		name string
	}{
		{url: "https://alexbates.dev", name: "Alex Bates Website"},
		{url: "https://pocketworks.co.uk", name: "Pocketworks"},
		{url: "https://www.anthonygordonpileofshite.com", name: "Anthony Gordon Pile of Shite"},
	}

	// Insert websites
	for _, website := range websites {
		_, err := s.db.Exec("INSERT INTO websites (url, name) VALUES (?, ?)", website.url, website.name)
		if err != nil {
			return fmt.Errorf("failed to insert website %s: %w", website.url, err)
		}
		s.logger.Info("Seeded website", "url", website.url, "name", website.name)
	}

	s.logger.Info("Database seeded successfully", "websites_added", len(websites))
	return nil
}

// Add a new website to the database
func (s *Server) addWebsite(url, name string) error {
	// Check if website already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM websites WHERE url = ?", url).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if website exists: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("website with URL %s already exists", url)
	}

	// Insert new website
	_, err = s.db.Exec("INSERT INTO websites (url, name) VALUES (?, ?)", url, name)
	if err != nil {
		return fmt.Errorf("failed to insert website: %w", err)
	}

	s.logger.Info("Added new website", "url", url, "name", name)
	return nil
}

// Get all active websites
func (s *Server) GetActiveWebsites() ([]models.Website, error) {
	rows, err := s.db.Query("SELECT id, url, name, check_interval, is_active, created_at, updated_at FROM websites WHERE is_active = 1")
	if err != nil {
		return nil, fmt.Errorf("failed to query active websites: %w", err)
	}
	defer rows.Close()

	var websites []models.Website
	for rows.Next() {
		var website models.Website
		err := rows.Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.IsActive, &website.CreatedAt, &website.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website: %w", err)
		}
		websites = append(websites, website)
	}

	return websites, nil
}

// Get a website by ID
func (s *Server) GetWebsiteByID(websiteID int) (*models.Website, error) {
	var website models.Website
	err := s.db.QueryRow("SELECT id, url, name, check_interval, is_active, created_at, updated_at FROM websites WHERE id = ?", websiteID).
		Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.IsActive, &website.CreatedAt, &website.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get website by ID: %w", err)
	}
	return &website, nil
}

// Store uptime check result
func (s *Server) StoreUptimeCheck(websiteID int, statusCode int, responseTime int64, isUp bool, errorMsg string) error {

	_, err := s.db.Exec("INSERT INTO uptime_checks (website_id, status_code, response_time_ms, is_up, error_message) VALUES (?, ?, ?, ?, ?)",
		websiteID, statusCode, responseTime, isUp, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to store uptime check: %w", err)
	}

	return nil
}

// Get uptime history for a website
func (s *Server) getUptimeHistory(websiteID int, limit int) ([]models.WebsiteStatus, error) {
	query := "SELECT id, website_id, status_code, response_time_ms, is_up, error_message, checked_at FROM uptime_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT ?"
	rows, err := s.db.Query(query, websiteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query uptime history: %w", err)
	}
	defer rows.Close()

	var statuses []models.WebsiteStatus
	for rows.Next() {
		var status models.WebsiteStatus
		var isUp bool
		err := rows.Scan(&status.ID, &status.WebsiteID, &status.StatusCode, &status.ResponseTime, &isUp, &status.Error, &status.CheckedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status: %w", err)
		}
		if isUp {
			status.Status = "up"
		} else {
			status.Status = "down"
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Get the last status for a website
func (s *Server) GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error) {
	var status models.WebsiteStatus
	var isUp bool
	err := s.db.QueryRow("SELECT id, website_id, status_code, response_time_ms, is_up, error_message, checked_at FROM uptime_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT 1", websiteID).
		Scan(&status.ID, &status.WebsiteID, &status.StatusCode, &status.ResponseTime, &isUp, &status.Error, &status.CheckedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get last website status: %w", err)
	}
	if isUp {
		status.Status = "up"
	} else {
		status.Status = "down"
	}
	return &status, nil
}

// Record that an alert was sent
func (s *Server) RecordAlertSent(websiteID int, alertType string) error {
	_, err := s.db.Exec("INSERT INTO alert_history (website_id, alert_type) VALUES (?, ?)", websiteID, alertType)
	if err != nil {
		return fmt.Errorf("failed to record alert sent: %w", err)
	}
	return nil
}

// Check if we should send an alert (avoid spam)
func (s *Server) ShouldSendAlert(websiteID int, alertType string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM alert_history WHERE website_id = ? AND alert_type = ? AND sent_at > datetime('now', '-30 minutes')", websiteID, alertType).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check alert history: %w", err)
	}

	return count == 0, nil
}

// Get websites with their current status for the dashboard
func (s *Server) getWebsitesWithStatus() ([]map[string]interface{}, error) {
	query := `
		SELECT w.id, w.url, w.name, w.check_interval, w.is_active, w.created_at, w.updated_at,
			   uc.status_code, uc.response_time_ms, uc.is_up, uc.error_message, uc.checked_at
		FROM websites w
		LEFT JOIN (
			SELECT website_id, status_code, response_time_ms, is_up, error_message, checked_at
			FROM uptime_checks
			WHERE id IN (
				SELECT MAX(id) FROM uptime_checks GROUP BY website_id
			)
		) uc ON w.id = uc.website_id
		WHERE w.is_active = 1
		ORDER BY w.name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query websites with status: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var website models.Website
		var statusCode sql.NullInt64
		var responseTime sql.NullInt64
		var isUp sql.NullBool
		var errorMsg sql.NullString
		var checkedAt sql.NullTime

		err := rows.Scan(
			&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.IsActive, &website.CreatedAt, &website.UpdatedAt,
			&statusCode, &responseTime, &isUp, &errorMsg, &checkedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website with status: %w", err)
		}

		result := map[string]interface{}{
			"website": website,
		}

		if checkedAt.Valid {
			status := "unknown"
			if isUp.Valid {
				if isUp.Bool {
					status = "up"
				} else {
					status = "down"
				}
			}
			result["status"] = status
			result["checked_at"] = checkedAt.Time
		} else {
			result["status"] = "unknown"
			result["checked_at"] = nil
		}

		results = append(results, result)
	}

	return results, nil
}

// CheckWebsite performs a manual check of a specific website
func (s *Server) CheckWebsite(website models.Website) error {
	// Import the monitor package to use its check logic
	monitor := monitor.New(s.logger, s.mailer, monitor.MonitorConfig{
		AlertRecipient: s.config.AlertRecipient,
	})

	// Perform the check
	monitor.CheckWebsite(website, s)

	return nil
}
