package auth

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"the-ark/internal/core"
)

// Service provides authentication functionality
type Service struct {
	users       *UserModel
	tokens      *TokenModel
	permissions *PermissionModel
	logger      *core.Logger
	config      *core.Config
}

// NewService creates a new authentication service
func NewService(logger *core.Logger, db *sql.DB, config *core.Config) *Service {
	// Convert sql.DB to core.Database
	coreDB := core.NewDatabase(db, logger)

	return &Service{
		users:       NewUserModel(coreDB, logger),
		tokens:      NewTokenModel(coreDB, logger),
		permissions: NewPermissionModel(coreDB, logger),
		logger:      logger,
		config:      config,
	}
}

// AuthenticateUser authenticates a user with email and password
func (s *Service) AuthenticateUser(email, password string) (*User, error) {
	// Get user by email
	user, err := s.users.GetByEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, ErrRecordNotFound):
			return nil, ErrInvalidCredentials
		default:
			return nil, err
		}
	}

	// Check if user is activated
	if !user.Activated {
		return nil, ErrUserNotActivated
	}

	// Check password
	match, err := user.Password.Matches(password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// CreateAuthenticationToken creates a new authentication token for a user
func (s *Service) CreateAuthenticationToken(user *User) (*Token, error) {
	// Delete any existing authentication tokens for this user
	err := s.tokens.DeleteAllForUser(ScopeAuthentication, user.ID)
	if err != nil {
		return nil, err
	}

	// Create new authentication token (24 hour expiry)
	token, err := s.tokens.New(user.ID, 24*time.Hour, ScopeAuthentication)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Created authentication token", "user_id", user.ID)
	return token, nil
}

// ValidateToken validates an authentication token
func (s *Service) ValidateToken(tokenPlaintext string) (*User, error) {
	user, err := s.users.GetForToken(ScopeAuthentication, tokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, ErrRecordNotFound):
			return nil, ErrInvalidToken
		default:
			return nil, err
		}
	}

	return user, nil
}

// GetUserPermissions retrieves all permissions for a user
func (s *Service) GetUserPermissions(userID int) (Permissions, error) {
	return s.permissions.GetAllForUser(userID)
}

// UserHasPermission checks if a user has a specific permission
func (s *Service) UserHasPermission(userID int, permissionCode string) (bool, error) {
	permissions, err := s.permissions.GetAllForUser(userID)
	if err != nil {
		return false, err
	}

	return permissions.Include(permissionCode), nil
}

// CreateUser creates a new user (for admin user creation)
func (s *Service) CreateUser(name, email, password string) (*User, error) {
	user := &User{
		Name:      name,
		Email:     email,
		Activated: true, // Admin user is pre-activated
	}

	// Set password
	err := user.Password.Set(password)
	if err != nil {
		return nil, err
	}

	// Insert user
	err = s.users.Insert(user)
	if err != nil {
		return nil, err
	}

	// Add admin permissions
	err = s.permissions.AddForUser(user.ID, "admin:all")
	if err != nil {
		return nil, err
	}

	s.logger.Info("Created user", "user_id", user.ID, "email", user.Email)
	return user, nil
}

// LogoutUser invalidates all authentication tokens for a user
func (s *Service) LogoutUser(userID int) error {
	err := s.tokens.DeleteAllForUser(ScopeAuthentication, userID)
	if err != nil {
		return err
	}

	s.logger.Info("User logged out", "user_id", userID)
	return nil
}

// LoginHandler handles web login requests
func (s *Service) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := s.AuthenticateUser(email, password)
	if err != nil {
		// Redirect back to login with error
		http.Redirect(w, r, "/auth/login?error=invalid_credentials", http.StatusSeeOther)
		return
	}

	// Create authentication token
	token, err := s.CreateAuthenticationToken(user)
	if err != nil {
		s.logger.Error("Failed to create authentication token", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set secure HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token.Plaintext,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set to false for development
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour.Seconds()),
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// LogoutHandler handles web logout requests
func (s *Service) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := GetUserFromContext(r)
	if !user.IsAnonymous() {
		// Invalidate tokens
		if err := s.LogoutUser(user.ID); err != nil {
			s.logger.Error("Failed to logout user", "error", err)
		}
	}

	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Set to false for development
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Delete cookie
	})

	// Redirect to login page
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// Common authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActivated   = errors.New("user not activated")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
