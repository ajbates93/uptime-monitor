package core

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Registry manages all features in The Ark portal
type Registry struct {
	features map[string]Feature
	mutex    sync.RWMutex
	logger   *Logger
}

// NewRegistry creates a new feature registry
func NewRegistry(logger *Logger) *Registry {
	return &Registry{
		features: make(map[string]Feature),
		logger:   logger,
	}
}

// Register adds a feature to the registry
func (r *Registry) Register(feature Feature) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := feature.Name()
	if _, exists := r.features[name]; exists {
		return fmt.Errorf("feature %s already registered", name)
	}

	r.features[name] = feature
	r.logger.Info("Registered feature", "name", name, "enabled", feature.Enabled())
	return nil
}

// Get retrieves a feature by name
func (r *Registry) Get(name string) (Feature, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	feature, exists := r.features[name]
	return feature, exists
}

// List returns all registered features
func (r *Registry) List() []Feature {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	features := make([]Feature, 0, len(r.features))
	for _, feature := range r.features {
		features = append(features, feature)
	}

	// Sort by name for consistent ordering
	sort.Slice(features, func(i, j int) bool {
		return features[i].Name() < features[j].Name()
	})

	return features
}

// ListEnabled returns only enabled features
func (r *Registry) ListEnabled() []Feature {
	allFeatures := r.List()
	enabledFeatures := make([]Feature, 0)

	for _, feature := range allFeatures {
		if feature.Enabled() {
			enabledFeatures = append(enabledFeatures, feature)
		}
	}

	return enabledFeatures
}

// InitAll initializes all enabled features
func (r *Registry) InitAll(ctx context.Context) error {
	features := r.ListEnabled()
	r.logger.Info("Initializing features", "count", len(features))

	for _, feature := range features {
		if err := feature.Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize feature %s: %w", feature.Name(), err)
		}
		r.logger.Info("Initialized feature", "name", feature.Name())
	}

	return nil
}

// ShutdownAll gracefully shuts down all features
func (r *Registry) ShutdownAll(ctx context.Context) error {
	features := r.ListEnabled()
	r.logger.Info("Shutting down features", "count", len(features))

	for _, feature := range features {
		if err := feature.Shutdown(ctx); err != nil {
			r.logger.Error("Failed to shutdown feature", "name", feature.Name(), "error", err)
			// Continue shutting down other features
		} else {
			r.logger.Info("Shutdown feature", "name", feature.Name())
		}
	}

	return nil
}

// GetAllRoutes returns all routes from enabled features
func (r *Registry) GetAllRoutes() []Route {
	features := r.ListEnabled()
	var allRoutes []Route

	for _, feature := range features {
		routes := feature.Routes()
		allRoutes = append(allRoutes, routes...)
	}

	return allRoutes
}

// GetFeatureStatus returns the status of all features
func (r *Registry) GetFeatureStatus() map[string]FeatureStatus {
	features := r.List()
	status := make(map[string]FeatureStatus)

	for _, feature := range features {
		status[feature.Name()] = FeatureStatus{
			Name:        feature.Name(),
			Description: feature.Description(),
			Enabled:     feature.Enabled(),
		}
	}

	return status
}

// FeatureStatus represents the status of a feature
type FeatureStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}
