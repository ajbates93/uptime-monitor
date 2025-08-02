package database

import (
	"database/sql"
	"the-ark/internal/features/uptime/models"
	"time"
)

type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(db *sql.DB) *DatabaseService {
	return &DatabaseService{
		db: db,
	}
}

// GetActiveWebsites retrieves all active websites from the database
func (s *DatabaseService) GetActiveWebsites() ([]models.Website, error) {
	query := `
		SELECT id, name, url, check_interval, created_at
		FROM uptime_websites
		ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var websites []models.Website
	for rows.Next() {
		var website models.Website
		var createdAt time.Time

		err := rows.Scan(
			&website.ID,
			&website.Name,
			&website.URL,
			&website.CheckInterval,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		website.CreatedAt = createdAt
		website.UpdatedAt = createdAt // Use created_at for updated_at since it doesn't exist
		website.IsActive = true       // All websites in uptime_websites are considered active

		websites = append(websites, website)
	}

	return websites, nil
}

// GetWebsiteByID retrieves a specific website by ID
func (s *DatabaseService) GetWebsiteByID(websiteID int) (*models.Website, error) {
	query := `
		SELECT id, name, url, check_interval, created_at
		FROM uptime_websites
		WHERE id = ?
	`

	var website models.Website
	var createdAt time.Time

	err := s.db.QueryRow(query, websiteID).Scan(
		&website.ID,
		&website.Name,
		&website.URL,
		&website.CheckInterval,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}

	website.CreatedAt = createdAt
	website.UpdatedAt = createdAt // Use created_at for updated_at since it doesn't exist
	website.IsActive = true

	return &website, nil
}

// GetLastWebsiteStatus retrieves the most recent status for a website
func (s *DatabaseService) GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error) {
	query := `
		SELECT id, website_id, status, response_time, status_code, error_message, checked_at
		FROM uptime_checks
		WHERE website_id = ?
		ORDER BY checked_at DESC
		LIMIT 1
	`

	var status models.WebsiteStatus
	var checkedAt time.Time

	err := s.db.QueryRow(query, websiteID).Scan(
		&status.ID,
		&status.WebsiteID,
		&status.Status,
		&status.ResponseTime,
		&status.StatusCode,
		&status.Error,
		&checkedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No status found
		}
		return nil, err
	}

	status.CheckedAt = checkedAt
	return &status, nil
}

// StoreUptimeCheck stores a new uptime check result
func (s *DatabaseService) StoreUptimeCheck(websiteID int, statusCode int, responseTime int64, isUp bool, errorMsg string) error {
	status := "down"
	if isUp {
		status = "up"
	}

	query := `
		INSERT INTO uptime_checks (website_id, status, response_time, status_code, error_message, checked_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, websiteID, status, responseTime, statusCode, errorMsg, time.Now())
	return err
}

// ShouldSendAlert checks if an alert should be sent (prevents spam)
func (s *DatabaseService) ShouldSendAlert(websiteID int, alertType string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM alert_history 
		WHERE website_id = ? AND alert_type = ? AND sent_at > datetime('now', '-1 hour')
	`

	var count int
	err := s.db.QueryRow(query, websiteID, alertType).Scan(&count)
	if err != nil {
		return false, err
	}

	// Send alert if no alert was sent in the last hour
	return count == 0, nil
}

// RecordAlertSent records that an alert was sent
func (s *DatabaseService) RecordAlertSent(websiteID int, alertType string) error {
	query := `
		INSERT INTO alert_history (website_id, alert_type, sent_at)
		VALUES (?, ?, ?)
	`

	_, err := s.db.Exec(query, websiteID, alertType, time.Now())
	return err
}

// CheckWebsite performs a manual check of a website
func (s *DatabaseService) CheckWebsite(website models.Website) error {
	// This method is implemented in the monitor service
	// We just need to ensure the website exists
	_, err := s.GetWebsiteByID(website.ID)
	return err
}
