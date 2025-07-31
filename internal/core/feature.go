package core

import (
	"context"
	"net/http"
)

// Feature represents a modular feature in The Ark portal
type Feature interface {
	// Name returns the unique name of the feature
	Name() string

	// Description returns a human-readable description
	Description() string

	// Enabled returns whether this feature is enabled
	Enabled() bool

	// Init initializes the feature
	Init(ctx context.Context) error

	// Routes returns the HTTP routes for this feature
	Routes() []Route

	// Shutdown gracefully shuts down the feature
	Shutdown(ctx context.Context) error
}

// Route represents an HTTP route for a feature
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// BaseFeature provides common functionality for all features
type BaseFeature struct {
	name        string
	description string
	enabled     bool
	logger      *Logger
	db          *Database
	config      interface{}
}

// NewBaseFeature creates a new base feature
func NewBaseFeature(name, description string, enabled bool, logger *Logger, db *Database, config interface{}) *BaseFeature {
	return &BaseFeature{
		name:        name,
		description: description,
		enabled:     enabled,
		logger:      logger,
		db:          db,
		config:      config,
	}
}

// Name returns the feature name
func (f *BaseFeature) Name() string {
	return f.name
}

// Description returns the feature description
func (f *BaseFeature) Description() string {
	return f.description
}

// Enabled returns whether the feature is enabled
func (f *BaseFeature) Enabled() bool {
	return f.enabled
}

// Logger returns the feature-specific logger
func (f *BaseFeature) Logger() *Logger {
	return f.logger.ForFeature(f.name)
}

// DB returns the database connection
func (f *BaseFeature) DB() *Database {
	return f.db
}

// Config returns the feature configuration
func (f *BaseFeature) Config() interface{} {
	return f.config
}

// Default implementations for optional methods
func (f *BaseFeature) Init(ctx context.Context) error {
	f.Logger().Info("Initializing feature", "name", f.name)
	return nil
}

func (f *BaseFeature) Routes() []Route {
	return []Route{}
}

func (f *BaseFeature) Shutdown(ctx context.Context) error {
	f.Logger().Info("Shutting down feature", "name", f.name)
	return nil
}
