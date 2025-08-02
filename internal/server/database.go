package server

import (
	"database/sql"
	"fmt"
	"os"
	"the-ark/internal/server/models"
	"the-ark/internal/server/services/monitor"

	"golang.org/x/crypto/bcrypt"
)

// Database operations for websites
func (s *Server) initDatabase() error {
	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash BYTEA NOT NULL,
		activated BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create tokens table
	createTokensTable := `
	CREATE TABLE IF NOT EXISTS tokens (
		hash BYTEA PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expiry DATETIME NOT NULL,
		scope TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	);`

	// Create permissions table
	createPermissionsTable := `
	CREATE TABLE IF NOT EXISTS permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE
	);`

	// Create users_permissions table
	createUsersPermissionsTable := `
	CREATE TABLE IF NOT EXISTS users_permissions (
		user_id INTEGER NOT NULL,
		permission_id INTEGER NOT NULL,
		PRIMARY KEY (user_id, permission_id),
		FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
		FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
	);`

	// Create uptime_websites table (renamed from websites)
	createUptimeWebsitesTable := `
	CREATE TABLE IF NOT EXISTS uptime_websites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL UNIQUE,
		check_interval INTEGER DEFAULT 300,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create uptime_checks table (renamed from uptime_checks)
	createUptimeChecksTable := `
	CREATE TABLE IF NOT EXISTS uptime_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		website_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		response_time INTEGER,
		status_code INTEGER,
		error_message TEXT,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (website_id) REFERENCES uptime_websites (id) ON DELETE CASCADE
	);`

	// Create alert_history table to track when emails were sent
	createAlertHistoryTable := `
	CREATE TABLE IF NOT EXISTS alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		website_id INTEGER,
		alert_type TEXT NOT NULL,
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (website_id) REFERENCES uptime_websites (id) ON DELETE CASCADE
	);`

	// Execute all table creation statements
	tables := []struct {
		name string
		sql  string
	}{
		{"users", createUsersTable},
		{"tokens", createTokensTable},
		{"permissions", createPermissionsTable},
		{"users_permissions", createUsersPermissionsTable},
		{"uptime_websites", createUptimeWebsitesTable},
		{"uptime_checks", createUptimeChecksTable},
		{"alert_history", createAlertHistoryTable},
	}

	for _, table := range tables {
		_, err := s.db.Exec(table.sql)
		if err != nil {
			return fmt.Errorf("failed to create %s table: %w", table.name, err)
		}
	}

	return nil
}

func (s *Server) seedDatabase() error {
	// Check if admin user already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "hello@alexbates.dev").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check admin user: %w", err)
	}

	// If admin user doesn't exist, create it
	if count == 0 {
		// Get password from environment variable
		adminPassword := os.Getenv("ARK_ADMIN_PASSWORD")
		if adminPassword == "" {
			return fmt.Errorf("ARK_ADMIN_PASSWORD environment variable is required")
		}

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 12)
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		_, err = s.db.Exec("INSERT INTO users (name, email, password_hash, activated) VALUES (?, ?, ?, ?)",
			"Alex Bates", "hello@alexbates.dev", hashedPassword, true)
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		s.logger.Info("Created admin user", "email", "hello@alexbates.dev")
	}

	// Check if uptime websites already exist
	err = s.db.QueryRow("SELECT COUNT(*) FROM uptime_websites").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check uptime websites count: %w", err)
	}

	// If websites already exist, don't seed
	if count > 0 {
		s.logger.Info("Uptime websites already seeded, skipping...")
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
		_, err := s.db.Exec("INSERT INTO uptime_websites (url, name) VALUES (?, ?)", website.url, website.name)
		if err != nil {
			return fmt.Errorf("failed to insert website %s: %w", website.url, err)
		}
		s.logger.Info("Seeded uptime website", "url", website.url, "name", website.name)
	}

	s.logger.Info("Database seeded successfully", "websites_added", len(websites))
	return nil
}

// Add a new website to the database
func (s *Server) addWebsite(url, name string) error {
	// Check if website already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM uptime_websites WHERE url = ?", url).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check if website exists: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("website with URL %s already exists", url)
	}

	// Insert new website
	_, err = s.db.Exec("INSERT INTO uptime_websites (url, name) VALUES (?, ?)", url, name)
	if err != nil {
		return fmt.Errorf("failed to insert website: %w", err)
	}

	s.logger.Info("Added new website", "url", url, "name", name)
	return nil
}

// Get all active websites
func (s *Server) GetActiveWebsites() ([]models.Website, error) {
	rows, err := s.db.Query("SELECT id, url, name, check_interval, created_at FROM uptime_websites")
	if err != nil {
		return nil, fmt.Errorf("failed to query active websites: %w", err)
	}
	defer rows.Close()

	var websites []models.Website
	for rows.Next() {
		var website models.Website
		err := rows.Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website: %w", err)
		}
		website.IsActive = true               // All websites in the new schema are active
		website.UpdatedAt = website.CreatedAt // Use created_at as updated_at for now
		websites = append(websites, website)
	}

	return websites, nil
}

// Get a website by ID
func (s *Server) GetWebsiteByID(websiteID int) (*models.Website, error) {
	var website models.Website
	err := s.db.QueryRow("SELECT id, url, name, check_interval, created_at FROM uptime_websites WHERE id = ?", websiteID).
		Scan(&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get website by ID: %w", err)
	}
	website.IsActive = true               // All websites in the new schema are active
	website.UpdatedAt = website.CreatedAt // Use created_at as updated_at for now
	return &website, nil
}

// Store uptime check result
func (s *Server) StoreUptimeCheck(websiteID int, statusCode int, responseTime int64, isUp bool, errorMsg string) error {
	var status string
	if isUp {
		status = "up"
	} else {
		status = "down"
	}

	_, err := s.db.Exec("INSERT INTO uptime_checks (website_id, status, response_time, status_code, error_message) VALUES (?, ?, ?, ?, ?)",
		websiteID, status, responseTime, statusCode, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to store uptime check: %w", err)
	}

	return nil
}

// Get uptime history for a website
func (s *Server) getUptimeHistory(websiteID int, limit int) ([]models.WebsiteStatus, error) {
	query := "SELECT id, website_id, status_code, response_time, status, error_message, checked_at FROM uptime_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT ?"
	rows, err := s.db.Query(query, websiteID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query uptime history: %w", err)
	}
	defer rows.Close()

	var statuses []models.WebsiteStatus
	for rows.Next() {
		var status models.WebsiteStatus
		err := rows.Scan(&status.ID, &status.WebsiteID, &status.StatusCode, &status.ResponseTime, &status.Status, &status.Error, &status.CheckedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status: %w", err)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Get the last status for a website
func (s *Server) GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error) {
	var status models.WebsiteStatus
	err := s.db.QueryRow("SELECT id, website_id, status_code, response_time, status, error_message, checked_at FROM uptime_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT 1", websiteID).
		Scan(&status.ID, &status.WebsiteID, &status.StatusCode, &status.ResponseTime, &status.Status, &status.Error, &status.CheckedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get last website status: %w", err)
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
		SELECT w.id, w.url, w.name, w.check_interval, w.created_at,
			   uc.status_code, uc.response_time, uc.status, uc.error_message, uc.checked_at
		FROM uptime_websites w
		LEFT JOIN (
			SELECT website_id, status_code, response_time, status, error_message, checked_at
			FROM uptime_checks
			WHERE id IN (
				SELECT MAX(id) FROM uptime_checks GROUP BY website_id
			)
		) uc ON w.id = uc.website_id
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
		var status sql.NullString
		var errorMsg sql.NullString
		var checkedAt sql.NullTime

		err := rows.Scan(
			&website.ID, &website.URL, &website.Name, &website.CheckInterval, &website.CreatedAt,
			&statusCode, &responseTime, &status, &errorMsg, &checkedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan website with status: %w", err)
		}

		result := map[string]interface{}{
			"website": website,
		}

		website.IsActive = true               // All websites in new schema are active
		website.UpdatedAt = website.CreatedAt // Use created_at as updated_at for now

		if checkedAt.Valid {
			result["status"] = status.String
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
		AlertRecipient: s.config.Features.Uptime.AlertRecipient,
	})

	// Perform the check
	monitor.CheckWebsite(website, s)

	return nil
}
