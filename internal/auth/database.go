package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"the-ark/internal/core"
)

// Common errors
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrDuplicateEmail = errors.New("duplicate email")
)

// UserModel handles database operations for users
type UserModel struct {
	db     *core.Database
	logger *core.Logger
}

// NewUserModel creates a new user model
func NewUserModel(db *core.Database, logger *core.Logger) *UserModel {
	return &UserModel{
		db:     db,
		logger: logger,
	}
}

// Insert creates a new user
func (m *UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES (?, ?, ?, ?)
		RETURNING id, created_at
	`

	args := []interface{}{user.Name, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		switch {
		case err.Error() == `UNIQUE constraint failed: users.email`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

// GetByEmail retrieves a user by email
func (m *UserModel) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, created_at, name, email, password_hash, activated
		FROM users
		WHERE email = ?
	`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

// GetForToken retrieves a user by token
func (m *UserModel) GetForToken(tokenScope, tokenPlaintext string) (*User, error) {
	// Hash the token
	hash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = ? AND tokens.scope = ? AND tokens.expiry > ?
	`

	args := []interface{}{hash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

// Update updates a user
func (m *UserModel) Update(user *User) error {
	query := `
		UPDATE users
		SET name = ?, email = ?, password_hash = ?, activated = ?
		WHERE id = ?
	`

	args := []interface{}{user.Name, user.Email, user.Password.hash, user.Activated, user.ID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// TokenModel handles database operations for tokens
type TokenModel struct {
	db     *core.Database
	logger *core.Logger
}

// NewTokenModel creates a new token model
func NewTokenModel(db *core.Database, logger *core.Logger) *TokenModel {
	return &TokenModel{
		db:     db,
		logger: logger,
	}
}

// New creates a new token
func (m *TokenModel) New(userID int, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

// Insert stores a token in the database
func (m *TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES (?, ?, ?, ?)
	`

	args := []interface{}{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllForUser deletes all tokens for a user and scope
func (m *TokenModel) DeleteAllForUser(scope string, userID int) error {
	query := `
		DELETE FROM tokens
		WHERE scope = ? AND user_id = ?
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.db.ExecContext(ctx, query, scope, userID)
	return err
}

// PermissionModel handles database operations for permissions
type PermissionModel struct {
	db     *core.Database
	logger *core.Logger
}

// NewPermissionModel creates a new permission model
func NewPermissionModel(db *core.Database, logger *core.Logger) *PermissionModel {
	return &PermissionModel{
		db:     db,
		logger: logger,
	}
}

// GetAllForUser retrieves all permissions for a user
func (m *PermissionModel) GetAllForUser(userID int) (Permissions, error) {
	query := `
		SELECT permissions.code
		FROM permissions
		INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
		INNER JOIN users ON users_permissions.user_id = users.id
		WHERE users.id = ?
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// AddForUser adds permissions for a user
func (m *PermissionModel) AddForUser(userID int, codes ...string) error {
	query := `
		INSERT INTO users_permissions
		SELECT ?, permissions.id FROM permissions WHERE permissions.code = ?
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, code := range codes {
		_, err := m.db.ExecContext(ctx, query, userID, code)
		if err != nil {
			return err
		}
	}

	return nil
}
