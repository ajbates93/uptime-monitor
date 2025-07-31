package handlers

import (
	"encoding/json"
	"net/http"
	"the-ark/internal/auth"
	"the-ark/internal/core"
	"the-ark/views/portal"
)

// PortalHandler handles the main portal dashboard
type PortalHandler struct {
	logger      *core.Logger
	registry    *core.Registry
	authService *auth.Service
}

// NewPortalHandler creates a new portal handler
func NewPortalHandler(logger *core.Logger, registry *core.Registry, authService *auth.Service) *PortalHandler {
	return &PortalHandler{
		logger:      logger,
		registry:    registry,
		authService: authService,
	}
}

// DashboardHandler serves the main portal dashboard
func (h *PortalHandler) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := auth.GetUserFromContext(r)

	// Get feature status
	featureStatus := h.registry.GetFeatureStatus()

	// Render dashboard
	component := portal.Dashboard(user, featureStatus)
	component.Render(r.Context(), w)
}

// LoginPageHandler serves the login page
func (h *PortalHandler) LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	// If user is already authenticated, redirect to dashboard
	user := auth.GetUserFromContext(r)
	if !user.IsAnonymous() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Render login page
	component := portal.LoginPage()
	component.Render(r.Context(), w)
}

// HealthCheckHandler provides a health check endpoint
func (h *PortalHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "the-ark",
		"version": "1.0.0",
	})
}
