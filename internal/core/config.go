package core

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config represents the main configuration for The Ark
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Auth     AuthConfig     `json:"auth"`
	Features FeatureConfig  `json:"features"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	Path string `json:"path"`
}

// AuthConfig contains authentication-related configuration
type AuthConfig struct {
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
	SessionSecret string `json:"session_secret"`
}

// FeatureConfig contains feature-specific configuration
type FeatureConfig struct {
	Uptime UptimeConfig           `json:"uptime"`
	Server ServerMonitoringConfig `json:"server"`
	SSL    SSLConfig              `json:"ssl"`
	Logs   LogViewerConfig        `json:"logs"`
}

// UptimeConfig contains uptime monitoring configuration
type UptimeConfig struct {
	Enabled        bool   `json:"enabled"`
	CheckInterval  int    `json:"check_interval"`
	SMTP2GOAPIKey  string `json:"smtp2go_api_key"`
	SMTP2GOSender  string `json:"smtp2go_sender"`
	AlertRecipient string `json:"alert_recipient"`
}

// ServerMonitoringConfig contains server monitoring configuration
type ServerMonitoringConfig struct {
	Enabled bool `json:"enabled"`
}

// SSLConfig contains SSL certificate tracking configuration
type SSLConfig struct {
	Enabled bool `json:"enabled"`
}

// LogViewerConfig contains log viewer configuration
type LogViewerConfig struct {
	Enabled bool `json:"enabled"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("ARK_PORT", 4000),
			Host: getEnvOrDefault("ARK_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Path: getEnvOrDefault("ARK_DB_PATH", "./ark.db"),
		},
		Auth: AuthConfig{
			AdminEmail:    getEnvOrDefault("ARK_ADMIN_EMAIL", "hello@alexbates.dev"),
			AdminPassword: getEnvOrDefault("ARK_ADMIN_PASSWORD", ""),
			SessionSecret: getEnvOrDefault("ARK_SESSION_SECRET", ""),
		},
		Features: FeatureConfig{
			Uptime: UptimeConfig{
				Enabled:        getEnvAsBool("ARK_ENABLE_UPTIME", true),
				CheckInterval:  getEnvAsInt("ARK_UPTIME_CHECK_INTERVAL", 300),
				SMTP2GOAPIKey:  getEnvOrDefault("ARK_SMTP2GO_API_KEY", ""),
				SMTP2GOSender:  getEnvOrDefault("ARK_SMTP2GO_SENDER", "The Ark <ark@alexbates.dev>"),
				AlertRecipient: getEnvOrDefault("ARK_ALERT_RECIPIENT", "ajbates93@gmail.com"),
			},
			Server: ServerMonitoringConfig{
				Enabled: getEnvAsBool("ARK_ENABLE_SERVER_MONITORING", false),
			},
			SSL: SSLConfig{
				Enabled: getEnvAsBool("ARK_ENABLE_SSL_TRACKER", false),
			},
			Logs: LogViewerConfig{
				Enabled: getEnvAsBool("ARK_ENABLE_LOG_VIEWER", false),
			},
		},
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	if c.Auth.AdminEmail == "" {
		return fmt.Errorf("admin email is required")
	}

	if c.Auth.AdminPassword == "" {
		return fmt.Errorf("admin password is required")
	}

	if c.Auth.SessionSecret == "" {
		return fmt.Errorf("session secret is required")
	}

	// Validate uptime config if enabled
	if c.Features.Uptime.Enabled {
		if c.Features.Uptime.SMTP2GOAPIKey == "" {
			return fmt.Errorf("SMTP2GO API key is required when uptime monitoring is enabled")
		}
	}

	return nil
}

// GetFeatureConfig returns configuration for a specific feature
func (c *Config) GetFeatureConfig(featureName string) interface{} {
	switch strings.ToLower(featureName) {
	case "uptime":
		return c.Features.Uptime
	case "server":
		return c.Features.Server
	case "ssl":
		return c.Features.SSL
	case "logs":
		return c.Features.Logs
	default:
		return nil
	}
}

// IsFeatureEnabled checks if a feature is enabled
func (c *Config) IsFeatureEnabled(featureName string) bool {
	switch strings.ToLower(featureName) {
	case "uptime":
		return c.Features.Uptime.Enabled
	case "server":
		return c.Features.Server.Enabled
	case "ssl":
		return c.Features.SSL.Enabled
	case "logs":
		return c.Features.Logs.Enabled
	default:
		return false
	}
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}
