package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"the-ark/internal/core"
)

// Handler provides authentication HTTP handlers
type Handler struct {
	service *Service
	logger  *core.Logger
}

// NewHandler creates a new authentication handler
func NewHandler(service *Service, logger *core.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User  *User  `json:"user"`
	Token *Token `json:"token"`
}

// LoginHandler handles user login
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		core.WriteErrorResponse(w, http.StatusMethodNotAllowed, core.NewAppError(
			core.ErrCodeValidation, "Method not allowed", nil))
		return
	}

	// Parse request
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteErrorResponse(w, http.StatusBadRequest, core.NewAppError(
			core.ErrCodeValidation, "Invalid request body", err))
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		core.WriteErrorResponse(w, http.StatusBadRequest, core.NewAppError(
			core.ErrCodeValidation, "Email and password are required", nil))
		return
	}

	// Authenticate user
	user, err := h.service.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			core.WriteErrorResponse(w, http.StatusUnauthorized, core.NewAppError(
				core.ErrCodeUnauthorized, "Invalid credentials", err))
		case errors.Is(err, ErrUserNotActivated):
			core.WriteErrorResponse(w, http.StatusForbidden, core.NewAppError(
				core.ErrCodeForbidden, "Account not activated", err))
		default:
			h.logger.Error("Authentication error", "error", err)
			core.WriteErrorResponse(w, http.StatusInternalServerError, core.NewAppError(
				core.ErrCodeInternal, "Authentication failed", err))
		}
		return
	}

	// Create authentication token
	token, err := h.service.CreateAuthenticationToken(user)
	if err != nil {
		h.logger.Error("Token creation error", "error", err)
		core.WriteErrorResponse(w, http.StatusInternalServerError, core.NewAppError(
			core.ErrCodeInternal, "Failed to create authentication token", err))
		return
	}

	// Set secure HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token.Plaintext,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set to false for development without HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Return success response
	response := LoginResponse{
		User:  user,
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})

	h.logger.Info("User logged in", "user_id", user.ID, "email", user.Email)
}

// LogoutHandler handles user logout
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		core.WriteErrorResponse(w, http.StatusMethodNotAllowed, core.NewAppError(
			core.ErrCodeValidation, "Method not allowed", nil))
		return
	}

	// Get user from context
	user := GetUserFromContext(r)
	if user.IsAnonymous() {
		core.WriteErrorResponse(w, http.StatusUnauthorized, core.NewAppError(
			core.ErrCodeUnauthorized, "Not authenticated", nil))
		return
	}

	// Logout user (invalidate tokens)
	err := h.service.LogoutUser(user.ID)
	if err != nil {
		h.logger.Error("Logout error", "error", err)
		core.WriteErrorResponse(w, http.StatusInternalServerError, core.NewAppError(
			core.ErrCodeInternal, "Logout failed", err))
		return
	}

	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(-1 * time.Hour), // Expire immediately
		MaxAge:   -1,
	})

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})

	h.logger.Info("User logged out", "user_id", user.ID, "email", user.Email)
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(r *http.Request) *User {
	user, ok := r.Context().Value(userContextKey).(*User)
	if !ok {
		return AnonymousUser
	}
	return user
}

// SessionMiddleware handles session-based authentication using cookies
func (h *Handler) SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for auth cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			// No cookie, continue with anonymous user
			r = h.contextSetUser(r, AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Validate token
		user, err := h.service.ValidateToken(cookie.Value)
		if err != nil {
			// Invalid token, clear cookie and continue with anonymous user
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				Expires:  time.Now().Add(-1 * time.Hour),
				MaxAge:   -1,
			})
			r = h.contextSetUser(r, AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Valid token, set user in context
		r = h.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// Context management (reusing from middleware.go)
func (h *Handler) contextSetUser(r *http.Request, user *User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}
