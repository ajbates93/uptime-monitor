package models

import (
	"time"
)

// SchedulerConfig holds configuration for the scheduler service
type SchedulerConfig struct {
	UpdateInterval time.Duration `json:"update_interval"`
	MaxWorkers     int           `json:"max_workers"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		UpdateInterval: 1 * time.Hour,   // Update every hour
		MaxWorkers:     5,               // 5 concurrent feed updates
		RetryAttempts:  3,               // Retry failed updates 3 times
		RetryDelay:     5 * time.Minute, // Wait 5 minutes between retries
	}
}
