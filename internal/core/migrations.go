package core

import (
	"context"
	"fmt"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Name        string
	Description string
	UpSQL       string
	DownSQL     string
	CreatedAt   time.Time
}

// MigrationService handles database migrations
type MigrationService struct {
	db     *Database
	logger *Logger
}

// NewMigrationService creates a new migration service
func NewMigrationService(db *Database, logger *Logger) *MigrationService {
	return &MigrationService{
		db:     db,
		logger: logger,
	}
}

// InitMigrations initializes the migrations table
func (m *MigrationService) InitMigrations(ctx context.Context) error {
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := m.db.ExecWithTimeout(ctx, createMigrationsTable)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	m.logger.Info("Migrations table initialized")
	return nil
}

// GetAppliedMigrations returns all applied migrations
func (m *MigrationService) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `SELECT version, name, description, applied_at FROM migrations ORDER BY version`

	rows, err := m.db.QueryWithTimeout(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.Version, &migration.Name, &migration.Description, &migration.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// IsMigrationApplied checks if a migration has been applied
func (m *MigrationService) IsMigrationApplied(ctx context.Context, version int) (bool, error) {
	query := `SELECT COUNT(*) FROM migrations WHERE version = ?`

	var count int
	err := m.db.QueryRowWithTimeout(ctx, query, version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}

	return count > 0, nil
}

// ApplyMigration applies a single migration
func (m *MigrationService) ApplyMigration(ctx context.Context, migration Migration) error {
	// Check if already applied
	applied, err := m.IsMigrationApplied(ctx, migration.Version)
	if err != nil {
		return err
	}
	if applied {
		m.logger.Info("Migration already applied", "version", migration.Version, "name", migration.Name)
		return nil
	}

	// Begin transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Execute migration SQL
	_, err = tx.ExecContext(ctx, migration.UpSQL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	// Record migration as applied
	insertQuery := `INSERT INTO migrations (version, name, description) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, insertQuery, migration.Version, migration.Name, migration.Description)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
	}

	m.logger.Info("Applied migration", "version", migration.Version, "name", migration.Name)
	return nil
}

// RollbackMigration rolls back a single migration
func (m *MigrationService) RollbackMigration(ctx context.Context, migration Migration) error {
	// Check if migration is applied
	applied, err := m.IsMigrationApplied(ctx, migration.Version)
	if err != nil {
		return err
	}
	if !applied {
		m.logger.Info("Migration not applied, cannot rollback", "version", migration.Version, "name", migration.Name)
		return nil
	}

	// Begin transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Execute rollback SQL
	_, err = tx.ExecContext(ctx, migration.DownSQL)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to rollback migration %d (%s): %w", migration.Version, migration.Name, err)
	}

	// Remove migration record
	deleteQuery := `DELETE FROM migrations WHERE version = ?`
	_, err = tx.ExecContext(ctx, deleteQuery, migration.Version)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %d: %w", migration.Version, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback %d: %w", migration.Version, err)
	}

	m.logger.Info("Rolled back migration", "version", migration.Version, "name", migration.Name)
	return nil
}

// GetMigrationStatus returns the status of all migrations
func (m *MigrationService) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	status := &MigrationStatus{
		AppliedCount: len(applied),
		Applied:      applied,
		LastApplied:  nil,
	}

	if len(applied) > 0 {
		status.LastApplied = &applied[len(applied)-1]
	}

	return status, nil
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	AppliedCount int         `json:"applied_count"`
	Applied      []Migration `json:"applied"`
	LastApplied  *Migration  `json:"last_applied,omitempty"`
}
