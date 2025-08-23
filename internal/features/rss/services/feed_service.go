package services

import (
	"context"
	"database/sql"
	"fmt"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/models"
	"time"
)

// FeedService handles RSS feed operations
type FeedService struct {
	db     *core.Database
	logger *core.Logger
}

// NewFeedService creates a new feed service
func NewFeedService(db *core.Database, logger *core.Logger) *FeedService {
	return &FeedService{
		db:     db,
		logger: logger,
	}
}

// CreateFeed creates a new RSS feed
func (s *FeedService) CreateFeed(ctx context.Context, feed *models.FeedCreate) (*models.Feed, error) {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Insert feed
	query := `
		INSERT INTO rss_feeds (title, url, description, site_url, favicon_url, fetch_interval, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var id int
	var createdAt, updatedAt time.Time

	err = tx.QueryRowContext(ctx, query,
		feed.Title,
		feed.URL,
		feed.Description,
		feed.SiteURL,
		feed.FaviconURL,
		feed.FetchInterval,
		now,
		now,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	// Insert category relationships if provided
	if len(feed.CategoryIDs) > 0 {
		for _, categoryID := range feed.CategoryIDs {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO rss_feed_categories (feed_id, category_id) VALUES (?, ?)",
				id, categoryID)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to link feed to category %d: %w", categoryID, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return created feed
	createdFeed := &models.Feed{
		ID:            id,
		Title:         feed.Title,
		URL:           feed.URL,
		Description:   feed.Description,
		SiteURL:       feed.SiteURL,
		FaviconURL:    feed.FaviconURL,
		FetchInterval: feed.FetchInterval,
		Enabled:       true,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}

	s.logger.Info("Created RSS feed", "id", id, "title", feed.Title, "url", feed.URL)
	return createdFeed, nil
}

// GetFeed retrieves a feed by ID
func (s *FeedService) GetFeed(ctx context.Context, id int) (*models.Feed, error) {
	query := `
		SELECT f.id, f.title, f.url, f.description, f.site_url, f.favicon_url,
		       f.last_fetched, f.fetch_interval, f.enabled, f.created_at, f.updated_at
		FROM rss_feeds f
		WHERE f.id = ?
	`

	var feed models.Feed
	var lastFetched sql.NullTime

	err := s.db.QueryRowWithTimeout(ctx, query, id).Scan(
		&feed.ID,
		&feed.Title,
		&feed.URL,
		&feed.Description,
		&feed.SiteURL,
		&feed.FaviconURL,
		&lastFetched,
		&feed.FetchInterval,
		&feed.Enabled,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("feed not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}

	if lastFetched.Valid {
		feed.LastFetched = &lastFetched.Time
	}

	// Load categories
	categories, err := s.getFeedCategories(ctx, id)
	if err != nil {
		s.logger.Error("Failed to load feed categories", "feed_id", id, "error", err)
	} else {
		feed.Categories = categories
	}

	return &feed, nil
}

// ListFeeds retrieves all feeds with optional filtering
func (s *FeedService) ListFeeds(ctx context.Context, enabledOnly bool) ([]models.Feed, error) {
	query := `
		SELECT f.id, f.title, f.url, f.description, f.site_url, f.favicon_url,
		       f.last_fetched, f.fetch_interval, f.enabled, f.created_at, f.updated_at
		FROM rss_feeds f
	`
	args := []interface{}{}

	if enabledOnly {
		query += " WHERE f.enabled = 1"
	}

	query += " ORDER BY f.title"

	rows, err := s.db.QueryWithTimeout(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		var lastFetched sql.NullTime

		err := rows.Scan(
			&feed.ID,
			&feed.Title,
			&feed.URL,
			&feed.Description,
			&feed.SiteURL,
			&feed.FaviconURL,
			&lastFetched,
			&feed.FetchInterval,
			&feed.Enabled,
			&feed.CreatedAt,
			&feed.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}

		if lastFetched.Valid {
			feed.LastFetched = &lastFetched.Time
		}

		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// UpdateFeed updates an existing feed
func (s *FeedService) UpdateFeed(ctx context.Context, id int, update *models.FeedUpdate) (*models.Feed, error) {
	// Get current feed
	currentFeed, err := s.GetFeed(ctx, id)
	if err != nil {
		return nil, err
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Build update query dynamically
	query := "UPDATE rss_feeds SET updated_at = ?"
	args := []interface{}{time.Now()}

	if update.Title != nil {
		query += ", title = ?"
		args = append(args, *update.Title)
		currentFeed.Title = *update.Title
	}

	if update.Description != nil {
		query += ", description = ?"
		args = append(args, *update.Description)
		currentFeed.Description = *update.Description
	}

	if update.SiteURL != nil {
		query += ", site_url = ?"
		args = append(args, *update.SiteURL)
		currentFeed.SiteURL = *update.SiteURL
	}

	if update.FaviconURL != nil {
		query += ", favicon_url = ?"
		args = append(args, *update.FaviconURL)
		currentFeed.FaviconURL = *update.FaviconURL
	}

	if update.FetchInterval != nil {
		query += ", fetch_interval = ?"
		args = append(args, *update.FetchInterval)
		currentFeed.FetchInterval = *update.FetchInterval
	}

	if update.Enabled != nil {
		query += ", enabled = ?"
		args = append(args, *update.Enabled)
		currentFeed.Enabled = *update.Enabled
	}

	if update.LastFetched != nil {
		query += ", last_fetched = ?"
		args = append(args, *update.LastFetched)
		currentFeed.LastFetched = update.LastFetched
	}

	query += " WHERE id = ?"
	args = append(args, id)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update feed: %w", err)
	}

	// Update categories if provided
	if update.CategoryIDs != nil {
		// Remove existing category relationships
		_, err = tx.ExecContext(ctx, "DELETE FROM rss_feed_categories WHERE feed_id = ?", id)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to remove existing category relationships: %w", err)
		}

		// Add new category relationships
		for _, categoryID := range update.CategoryIDs {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO rss_feed_categories (feed_id, category_id) VALUES (?, ?)",
				id, categoryID)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to link feed to category %d: %w", categoryID, err)
			}
		}

		// Reload categories
		categories, err := s.getFeedCategories(ctx, id)
		if err != nil {
			s.logger.Error("Failed to reload feed categories", "feed_id", id, "error", err)
		} else {
			currentFeed.Categories = categories
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Updated RSS feed", "id", id)
	return currentFeed, nil
}

// DeleteFeed deletes a feed and all its articles
func (s *FeedService) DeleteFeed(ctx context.Context, id int) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Delete feed (cascading deletes will handle related records)
	_, err = tx.ExecContext(ctx, "DELETE FROM rss_feeds WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Deleted RSS feed", "id", id)
	return nil
}

// getFeedCategories retrieves categories for a specific feed
func (s *FeedService) getFeedCategories(ctx context.Context, feedID int) ([]models.Category, error) {
	query := `
		SELECT c.id, c.name, c.color, c.created_at
		FROM rss_categories c
		JOIN rss_feed_categories fc ON c.id = fc.category_id
		WHERE fc.feed_id = ?
		ORDER BY c.name
	`

	rows, err := s.db.QueryWithTimeout(ctx, query, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(&category.ID, &category.Name, &category.Color, &category.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}
