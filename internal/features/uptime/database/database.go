package database

import (
	"database/sql"
	"fmt"
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

// GetUptimePercentage calculates the uptime percentage for a given time period
func (s *DatabaseService) GetUptimePercentage(websiteID int, hours int) (float64, int, int, error) {
	query := `
		SELECT 
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_checks
		FROM uptime_checks 
		WHERE website_id = ? 
		AND checked_at >= datetime('now', '-' || ? || ' hours')
	`

	var totalChecks, upChecks int
	err := s.db.QueryRow(query, websiteID, hours).Scan(&totalChecks, &upChecks)
	if err != nil {
		return 0, 0, 0, err
	}

	if totalChecks == 0 {
		return 100.0, 0, 0, nil // No checks means 100% uptime
	}

	percentage := float64(upChecks) / float64(totalChecks) * 100
	return percentage, upChecks, totalChecks - upChecks, nil
}

// GetUptimeHistory returns uptime checks for a website with pagination
func (s *DatabaseService) GetUptimeHistory(websiteID int, limit int) ([]models.WebsiteStatus, error) {
	query := `
		SELECT id, website_id, status, response_time, status_code, error_message, checked_at
		FROM uptime_checks
		WHERE website_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, websiteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []models.WebsiteStatus
	for rows.Next() {
		var status models.WebsiteStatus
		var checkedAt time.Time

		err := rows.Scan(
			&status.ID,
			&status.WebsiteID,
			&status.Status,
			&status.ResponseTime,
			&status.StatusCode,
			&status.Error,
			&checkedAt,
		)
		if err != nil {
			return nil, err
		}

		status.CheckedAt = checkedAt
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetIncidents returns incidents (downtime periods) for a website
func (s *DatabaseService) GetIncidents(websiteID int, limit int) ([]models.Incident, error) {
	query := `
		WITH status_changes AS (
			SELECT 
				status,
				checked_at,
				LAG(status) OVER (ORDER BY checked_at) as prev_status,
				LAG(checked_at) OVER (ORDER BY checked_at) as prev_checked_at
			FROM uptime_checks
			WHERE website_id = ?
			ORDER BY checked_at
		)
		SELECT 
			prev_checked_at as started_at,
			checked_at as resolved_at,
			status as final_status
		FROM status_changes
		WHERE prev_status = 'up' AND status = 'down'
		ORDER BY prev_checked_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, websiteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []models.Incident
	for rows.Next() {
		var incident models.Incident
		var startedAtStr, resolvedAtStr sql.NullString
		var finalStatus string

		err := rows.Scan(&startedAtStr, &resolvedAtStr, &finalStatus)
		if err != nil {
			return nil, err
		}

		// Parse started_at
		if startedAtStr.Valid {
			startedAt, err := parseTimeFlexible(startedAtStr.String)
			if err != nil {
				return nil, err
			}
			incident.StartedAt = startedAt
		}

		// Parse resolved_at
		if resolvedAtStr.Valid {
			resolvedAt, err := parseTimeFlexible(resolvedAtStr.String)
			if err != nil {
				return nil, err
			}
			incident.ResolvedAt = &resolvedAt
		}

		incident.Status = finalStatus
		incident.WebsiteID = websiteID

		// Calculate duration
		if incident.ResolvedAt != nil {
			incident.Duration = incident.ResolvedAt.Sub(incident.StartedAt)
		} else {
			incident.Duration = time.Since(incident.StartedAt)
		}

		incidents = append(incidents, incident)
	}

	return incidents, nil
}

// GetAverageResponseTime calculates the average response time for a given period
func (s *DatabaseService) GetAverageResponseTime(websiteID int, hours int) (float64, error) {
	query := `
		SELECT AVG(response_time)
		FROM uptime_checks 
		WHERE website_id = ? 
		AND checked_at >= datetime('now', '-' || ? || ' hours')
		AND status = 'up'
	`

	var avgResponseTime sql.NullFloat64
	err := s.db.QueryRow(query, websiteID, hours).Scan(&avgResponseTime)
	if err != nil {
		return 0, err
	}

	if avgResponseTime.Valid {
		return avgResponseTime.Float64, nil
	}
	return 0, nil
}

// GetWebsiteDetailData retrieves all data needed for the detailed website view
func (s *DatabaseService) GetWebsiteDetailData(websiteID int) (*models.WebsiteDetailData, error) {
	// Get website
	website, err := s.GetWebsiteByID(websiteID)
	if err != nil {
		return nil, err
	}

	// Get last status
	lastStatus, err := s.GetLastWebsiteStatus(websiteID)
	if err != nil {
		return nil, err
	}

	// Get uptime stats for different periods
	uptimeStats, err := s.getUptimeStats(websiteID)
	if err != nil {
		return nil, err
	}

	// Get incidents
	incidents, err := s.GetIncidents(websiteID, 10)
	if err != nil {
		return nil, err
	}

	// Get average response time
	avgResponse, err := s.GetAverageResponseTime(websiteID, 24*30) // 30 days
	if err != nil {
		return nil, err
	}

	return &models.WebsiteDetailData{
		Website:     *website,
		LastStatus:  lastStatus,
		UptimeStats: uptimeStats,
		Incidents:   incidents,
		AvgResponse: avgResponse,
	}, nil
}

// getUptimeStats calculates uptime statistics for different time periods
func (s *DatabaseService) getUptimeStats(websiteID int) ([]models.UptimeStats, error) {
	periods := []struct {
		hours int
		label string
	}{
		{24, "24h"},
		{24 * 7, "7d"},
		{24 * 30, "30d"},
		{24 * 365, "365d"},
	}

	var stats []models.UptimeStats
	for _, period := range periods {
		percentage, upChecks, downChecks, err := s.GetUptimePercentage(websiteID, period.hours)
		if err != nil {
			return nil, err
		}

		// Get incident count for this period
		incidents, err := s.GetIncidents(websiteID, 100) // Get more incidents to count
		if err != nil {
			return nil, err
		}

		// Count incidents in this period
		incidentCount := 0
		var totalDowntime time.Duration
		for _, incident := range incidents {
			if time.Since(incident.StartedAt) <= time.Duration(period.hours)*time.Hour {
				incidentCount++
				totalDowntime += incident.Duration
			}
		}

		stats = append(stats, models.UptimeStats{
			WebsiteID:     websiteID,
			Period:        period.label,
			Percentage:    percentage,
			UpChecks:      upChecks,
			DownChecks:    downChecks,
			TotalChecks:   upChecks + downChecks,
			IncidentCount: incidentCount,
			Downtime:      formatDuration(totalDowntime),
		})
	}

	return stats, nil
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.Hours())
	}
	return fmt.Sprintf("%.0fd", d.Hours()/24)
}

// parseTimeFlexible tries to parse a datetime string using multiple common formats
func parseTimeFlexible(timeStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,                      // 2025-08-03T18:04:25.926402+01:00
		"2006-01-02T15:04:05.999999999",       // 2025-08-03T18:04:25.926402
		"2006-01-02 15:04:05.999999999-07:00", // 2025-08-03 17:46:37.91092+01:00
		"2006-01-02 15:04:05.999999999",       // 2025-08-03 17:46:37.91092
		"2006-01-02 15:04:05",                 // 2025-08-03 17:46:37
		time.RFC3339,                          // 2025-08-03T18:04:25+01:00
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}
