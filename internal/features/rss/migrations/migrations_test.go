package migrations

import (
	"context"
	"database/sql"
	"testing"
	"the-ark/internal/core"

	_ "modernc.org/sqlite"
)

func TestRSSMigrations(t *testing.T) {
	// Create temporary database
	dbPath := ":memory:" // Use in-memory SQLite for testing
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create core database wrapper
	coreDB := core.NewDatabase(db, core.NewLogger())

	// Create migration manager
	manager := NewManager(coreDB, core.NewLogger())

	// Test that migrations can be applied
	ctx := context.Background()
	err = manager.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify migrations table was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}

	expectedMigrations := len(manager.Migrations())
	if count != expectedMigrations {
		t.Errorf("Expected %d migrations, got %d", expectedMigrations, count)
	}

	// Verify RSS tables were created
	tables := []string{"rss_feeds", "rss_categories", "rss_feed_categories", "rss_articles", "rss_article_tags", "rss_reading_progress"}
	for _, table := range tables {
		var tableCount int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&tableCount)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if tableCount != 1 {
			t.Errorf("Table %s was not created", table)
		}
	}

	// Test that migrations are idempotent (can be run multiple times)
	err = manager.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to re-apply migrations: %v", err)
	}

	// Verify no duplicate migrations
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}
	if count != expectedMigrations {
		t.Errorf("Expected %d migrations after re-apply, got %d", expectedMigrations, count)
	}
}

func TestMigrationRollback(t *testing.T) {
	// Create temporary database
	dbPath := ":memory:" // Use in-memory SQLite for testing
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create core database wrapper
	coreDB := core.NewDatabase(db, core.NewLogger())

	// Create migration manager
	manager := NewManager(coreDB, core.NewLogger())

	// Apply migrations
	ctx := context.Background()
	err = manager.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Test rollback
	err = manager.Rollback(ctx)
	if err != nil {
		t.Fatalf("Failed to rollback migrations: %v", err)
	}

	// Verify tables were removed
	tables := []string{"rss_feeds", "rss_categories", "rss_feed_categories", "rss_articles", "rss_article_tags", "rss_reading_progress"}
	for _, table := range tables {
		var tableCount int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&tableCount)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if tableCount != 0 {
			t.Errorf("Table %s was not removed during rollback", table)
		}
	}
}
