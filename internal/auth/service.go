package auth

import (
	"errors"
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
func NewService(db *core.Database, logger *core.Logger, config *core.Config) *Service {
	return &Service{
		users:       NewUserModel(db, logger),
		tokens:      NewTokenModel(db, logger),
		permissions: NewPermissionModel(db, logger),
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

// Common authentication errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotActivated   = errors.New("user not activated")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
