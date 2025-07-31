package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Database wraps sql.DB with additional functionality
type Database struct {
	*sql.DB
	logger *Logger
}

// NewDatabase creates a new database wrapper
func NewDatabase(db *sql.DB, logger *Logger) *Database {
	return &Database{
		DB:     db,
		logger: logger,
	}
}

// Transaction executes a function within a database transaction
func (db *Database) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, rollback and re-panic
			tx.Rollback()
			panic(p)
		} else if err != nil {
			// Something went wrong, rollback
			tx.Rollback()
		} else {
			// All good, commit
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// PingWithTimeout pings the database with a timeout
func (db *Database) PingWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return db.PingContext(ctx)
}

// QueryWithTimeout executes a query with a timeout
func (db *Database) QueryWithTimeout(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return db.QueryContext(queryCtx, query, args...)
}

// QueryRowWithTimeout executes a query row with a timeout
func (db *Database) QueryRowWithTimeout(ctx context.Context, query string, args ...interface{}) *sql.Row {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return db.QueryRowContext(queryCtx, query, args...)
}

// ExecWithTimeout executes a command with a timeout
func (db *Database) ExecWithTimeout(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return db.ExecContext(queryCtx, query, args...)
}

// Close closes the database connection
func (db *Database) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

// Stats returns database statistics
func (db *Database) Stats() sql.DBStats {
	return db.DB.Stats()
}

// LogStats logs database statistics
func (db *Database) LogStats() {
	stats := db.Stats()
	db.logger.Info("Database stats",
		"max_open_connections", stats.MaxOpenConnections,
		"open_connections", stats.OpenConnections,
		"in_use", stats.InUse,
		"idle", stats.Idle,
		"wait_count", stats.WaitCount,
		"wait_duration", stats.WaitDuration,
		"max_idle_closed", stats.MaxIdleClosed,
		"max_lifetime_closed", stats.MaxLifetimeClosed,
	)
}
