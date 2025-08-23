package migrations

import (
	"context"
	"fmt"
	"the-ark/internal/core"
)

// Manager handles RSS feature migrations
type Manager struct {
	migrationService *core.MigrationService
	logger           *core.Logger
}

// NewManager creates a new RSS migration manager
func NewManager(db *core.Database, logger *core.Logger) *Manager {
	migrationService := core.NewMigrationService(db, logger)
	return &Manager{
		migrationService: migrationService,
		logger:           logger,
	}
}

// Migrations returns all RSS migrations in order
func (m *Manager) Migrations() []core.Migration {
	return []core.Migration{
		Migration001CreateRSSTables,
	}
}

// Migrate applies all pending RSS migrations
func (m *Manager) Migrate(ctx context.Context) error {
	// Initialize migrations table if it doesn't exist
	if err := m.migrationService.InitMigrations(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	migrations := m.Migrations()
	m.logger.Info("Starting RSS migrations", "count", len(migrations))

	for _, migration := range migrations {
		if err := m.migrationService.ApplyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	m.logger.Info("RSS migrations completed successfully")
	return nil
}

// Rollback rolls back the last applied RSS migration
func (m *Manager) Rollback(ctx context.Context) error {
	// Initialize migrations table if it doesn't exist
	if err := m.migrationService.InitMigrations(ctx); err != nil {
		return fmt.Errorf("failed to initialize migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.migrationService.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Check if this migration is applied
	applied, err = m.migrationService.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find the last applied RSS migration
	var lastApplied *core.Migration
	for _, migration := range applied {
		for _, rssMigration := range m.Migrations() {
			if migration.Version == rssMigration.Version {
				lastApplied = &rssMigration
				break
			}
		}
	}

	if lastApplied == nil {
		return fmt.Errorf("no RSS migrations have been applied")
	}

	// Rollback the last applied RSS migration
	if err := m.migrationService.RollbackMigration(ctx, *lastApplied); err != nil {
		return fmt.Errorf("failed to rollback migration %d (%s): %w", lastApplied.Version, lastApplied.Name, err)
	}

	m.logger.Info("Rolled back RSS migration", "version", lastApplied.Version, "name", lastApplied.Name)
	return nil
}

// Status returns the current migration status
func (m *Manager) Status(ctx context.Context) (*core.MigrationStatus, error) {
	return m.migrationService.GetMigrationStatus(ctx)
}

// GetPendingMigrations returns migrations that haven't been applied yet
func (m *Manager) GetPendingMigrations(ctx context.Context) ([]core.Migration, error) {
	applied, err := m.migrationService.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map of applied migration versions
	appliedVersions := make(map[int]bool)
	for _, migration := range applied {
		appliedVersions[migration.Version] = true
	}

	// Find pending migrations
	var pending []core.Migration
	for _, migration := range m.Migrations() {
		if !appliedVersions[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}
