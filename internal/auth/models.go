package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in The Ark
type User struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  Password  `json:"-"` // Hidden from JSON
	Activated bool      `json:"activated"`
	Version   int       `json:"-"` // For optimistic locking
}

// Anonymous user for unauthenticated requests
var AnonymousUser = &User{}

// IsAnonymous checks if the user is anonymous
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// Password represents a hashed password
type Password struct {
	plaintext *string // Pointer to distinguish nil from empty
	hash      []byte
}

// Set hashes and stores a plaintext password
func (p *Password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

// Matches checks if a plaintext password matches the hash
func (p *Password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

// Token represents an authentication token
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// Token scopes
const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

// generateToken creates a new token for a user
func generateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	// Generate 16 random bytes
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Encode as base32 (26 characters)
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// Permissions represents a collection of permission codes
type Permissions []string

// Include checks if a permission code is included
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}
