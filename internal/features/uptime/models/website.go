package models

import "time"

type Website struct {
	ID            int       `json:"id"`
	URL           string    `json:"url"`
	Name          string    `json:"name"`
	CheckInterval int       `json:"check_interval"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WebsiteStatus struct {
	ID           int       `json:"id"`
	WebsiteID    int       `json:"website_id"`
	Status       string    `json:"status"`
	ResponseTime int64     `json:"response_time"`
	StatusCode   int       `json:"status_code"`
	Error        string    `json:"error,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// DashboardWebsite combines Website with its current status for the web interface
type DashboardWebsite struct {
	Website   Website
	Status    string
	CheckedAt *time.Time
}

// Incident represents a downtime period
type Incident struct {
	ID         int           `json:"id"`
	WebsiteID  int           `json:"website_id"`
	Status     string        `json:"status"`
	StartedAt  time.Time     `json:"started_at"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
	Duration   time.Duration `json:"duration"`
	RootCause  string        `json:"root_cause,omitempty"`
	Comments   string        `json:"comments,omitempty"`
}

// UptimeStats represents uptime statistics for a website
type UptimeStats struct {
	WebsiteID     int     `json:"website_id"`
	Period        string  `json:"period"`
	Percentage    float64 `json:"percentage"`
	UpChecks      int     `json:"up_checks"`
	DownChecks    int     `json:"down_checks"`
	TotalChecks   int     `json:"total_checks"`
	IncidentCount int     `json:"incident_count"`
	Downtime      string  `json:"downtime"`
}

// WebsiteDetailData contains all data needed for the detailed website view
type WebsiteDetailData struct {
	Website     Website        `json:"website"`
	LastStatus  *WebsiteStatus `json:"last_status"`
	UptimeStats []UptimeStats  `json:"uptime_stats"`
	Incidents   []Incident     `json:"incidents"`
	AvgResponse float64        `json:"avg_response"`
}
