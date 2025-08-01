package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"the-ark/internal/core"
)

// Context key for user
type contextKey string

const userContextKey = contextKey("user")

// Middleware provides authentication middleware
type Middleware struct {
	service *Service
	logger  *core.Logger
}

// NewMiddleware creates new authentication middleware
func NewMiddleware(service *Service, logger *core.Logger) *Middleware {
	return &Middleware{
		service: service,
		logger:  logger,
	}
}

// Authenticate middleware adds user to request context
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add Vary header for caching
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		// If no Authorization header, set anonymous user
		if authorizationHeader == "" {
			r = contextSetUser(r, AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Parse Bearer token
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			m.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		// Validate token
		user, err := m.service.ValidateToken(token)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidToken):
				m.invalidAuthenticationTokenResponse(w, r)
			default:
				m.logger.Error("Token validation error", "error", err)
				m.serverErrorResponse(w, r)
			}
			return
		}

		// Set user in request context
		r = contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// RequireAuthenticatedUser middleware requires an authenticated user
func (m *Middleware) RequireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		if user.IsAnonymous() {
			m.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireActivatedUser middleware requires an activated user
func (m *Middleware) RequireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		if user.IsAnonymous() {
			m.authenticationRequiredResponse(w, r)
			return
		}

		if !user.Activated {
			m.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return m.RequireAuthenticatedUser(fn)
}

// RequirePermission middleware requires a specific permission
func (m *Middleware) RequirePermission(permissionCode string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := contextGetUser(r)

		hasPermission, err := m.service.UserHasPermission(user.ID, permissionCode)
		if err != nil {
			m.logger.Error("Permission check error", "error", err)
			m.serverErrorResponse(w, r)
			return
		}

		if !hasPermission {
			m.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	return m.RequireActivatedUser(fn)
}

// Context management
func contextSetUser(r *http.Request, user *User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

func contextGetUser(r *http.Request) *User {
	user, ok := r.Context().Value(userContextKey).(*User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}

// Response helpers
func (m *Middleware) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	core.WriteErrorResponse(w, http.StatusUnauthorized, core.NewAppError(
		core.ErrCodeUnauthorized, "Invalid authentication token", nil))
}

func (m *Middleware) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	core.WriteErrorResponse(w, http.StatusUnauthorized, core.NewAppError(
		core.ErrCodeUnauthorized, "Authentication required", nil))
}

func (m *Middleware) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	core.WriteErrorResponse(w, http.StatusForbidden, core.NewAppError(
		core.ErrCodeForbidden, "Account not activated", nil))
}

func (m *Middleware) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	core.WriteErrorResponse(w, http.StatusForbidden, core.NewAppError(
		core.ErrCodeForbidden, "Permission denied", nil))
}

func (m *Middleware) serverErrorResponse(w http.ResponseWriter, r *http.Request) {
	core.WriteErrorResponse(w, http.StatusInternalServerError, core.NewAppError(
		core.ErrCodeInternal, "Internal server error", nil))
}

// RequireAuthentication middleware for Chi router
func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUserFromContext(r)

		if user.IsAnonymous() {
			// Redirect to login page for web requests
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WebAuthMiddleware adds user to request context from cookies
func WebAuthMiddleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get auth token from cookie
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				// No cookie, set anonymous user
				r = contextSetUser(r, AnonymousUser)
				next.ServeHTTP(w, r)
				return
			}

			// Validate token
			user, err := service.ValidateToken(cookie.Value)
			if err != nil {
				// Invalid token, set anonymous user
				r = contextSetUser(r, AnonymousUser)
				next.ServeHTTP(w, r)
				return
			}

			// Set user in request context
			r = contextSetUser(r, user)
			next.ServeHTTP(w, r)
		})
	}
}
