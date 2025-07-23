package main

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
