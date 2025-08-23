package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/models"
	"time"
)

// ArticleService handles RSS article operations
type ArticleService struct {
	db     *core.Database
	logger *core.Logger
}

// NewArticleService creates a new article service
func NewArticleService(db *core.Database, logger *core.Logger) *ArticleService {
	return &ArticleService{
		db:     db,
		logger: logger,
	}
}

// CreateArticle creates a new article
func (s *ArticleService) CreateArticle(ctx context.Context, article *models.ArticleCreate) (*models.Article, error) {
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

	// Insert article
	query := `
		INSERT INTO rss_articles (feed_id, title, link, description, content, author, published_at, guid, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, fetched_at
	`

	now := time.Now()
	var id int
	var fetchedAt time.Time

	err = tx.QueryRowContext(ctx, query,
		article.FeedID,
		article.Title,
		article.Link,
		article.Description,
		article.Content,
		article.Author,
		article.PublishedAt,
		article.GUID,
		now,
	).Scan(&id, &fetchedAt)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	// Insert tags if provided
	if len(article.Tags) > 0 {
		for _, tag := range article.Tags {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO rss_article_tags (article_id, tag) VALUES (?, ?)",
				id, tag)
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("failed to insert tag %s: %w", tag, err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return created article
	createdArticle := &models.Article{
		ID:          id,
		FeedID:      article.FeedID,
		Title:       article.Title,
		Link:        article.Link,
		Description: article.Description,
		Content:     article.Content,
		Author:      article.Author,
		PublishedAt: article.PublishedAt,
		GUID:        article.GUID,
		FetchedAt:   fetchedAt,
		IsRead:      false,
		IsStarred:   false,
		Tags:        article.Tags,
	}

	s.logger.Info("Created RSS article", "id", id, "title", article.Title, "feed_id", article.FeedID)
	return createdArticle, nil
}

// GetArticle retrieves an article by ID
func (s *ArticleService) GetArticle(ctx context.Context, id int) (*models.Article, error) {
	query := `
		SELECT a.id, a.feed_id, a.title, a.link, a.description, a.content, a.author,
		       a.published_at, a.fetched_at, a.read_at, a.is_read, a.is_starred, a.guid
		FROM rss_articles a
		WHERE a.id = ?
	`

	var article models.Article
	var publishedAt, readAt sql.NullTime

	err := s.db.QueryRowWithTimeout(ctx, query, id).Scan(
		&article.ID,
		&article.FeedID,
		&article.Title,
		&article.Link,
		&article.Description,
		&article.Content,
		&article.Author,
		&publishedAt,
		&article.FetchedAt,
		&readAt,
		&article.IsRead,
		&article.IsStarred,
		&article.GUID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("article not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	if publishedAt.Valid {
		article.PublishedAt = &publishedAt.Time
	}
	if readAt.Valid {
		article.ReadAt = &readAt.Time
	}

	// Load tags
	tags, err := s.getArticleTags(ctx, id)
	if err != nil {
		s.logger.Error("Failed to load article tags", "article_id", id, "error", err)
	} else {
		article.Tags = tags
	}

	return &article, nil
}

// ListArticles retrieves articles with filtering and pagination
func (s *ArticleService) ListArticles(ctx context.Context, params *models.ArticleListParams) ([]models.Article, error) {
	// Build query dynamically
	query := `
		SELECT DISTINCT a.id, a.feed_id, a.title, a.link, a.description, a.content, a.author,
		       a.published_at, a.fetched_at, a.read_at, a.is_read, a.is_starred, a.guid
		FROM rss_articles a
		LEFT JOIN rss_feeds f ON a.feed_id = f.id
		LEFT JOIN rss_feed_categories fc ON f.id = fc.feed_id
		LEFT JOIN rss_article_tags at ON a.id = at.article_id
	`

	args := make([]interface{}, 0)
	whereClauses := []string{}

	// Add filters
	if params.FeedID != nil {
		whereClauses = append(whereClauses, "a.feed_id = ?")
		args = append(args, *params.FeedID)
	}

	if params.CategoryID != nil {
		whereClauses = append(whereClauses, "fc.category_id = ?")
		args = append(args, *params.CategoryID)
	}

	if params.IsRead != nil {
		whereClauses = append(whereClauses, "a.is_read = ?")
		args = append(args, *params.IsRead)
	}

	if params.IsStarred != nil {
		whereClauses = append(whereClauses, "a.is_starred = ?")
		args = append(args, *params.IsStarred)
	}

	if params.Search != "" {
		whereClauses = append(whereClauses, "(a.title LIKE ? OR a.description LIKE ? OR a.content LIKE ?)")
		searchTerm := "%" + params.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	if params.FromDate != nil {
		whereClauses = append(whereClauses, "a.published_at >= ?")
		args = append(args, *params.FromDate)
	}

	if params.ToDate != nil {
		whereClauses = append(whereClauses, "a.published_at <= ?")
		args = append(args, *params.ToDate)
	}

	if params.Tags != nil && len(params.Tags) > 0 {
		placeholders := make([]string, len(params.Tags))
		for i := range params.Tags {
			placeholders[i] = "?"
			args = append(args, params.Tags[i])
		}
		whereClauses = append(whereClauses, "at.tag IN ("+strings.Join(placeholders, ",")+")")
	}

	// Add WHERE clause if we have filters
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add ORDER BY
	if params.SortBy == "" {
		params.SortBy = "published_at"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}

	query += " ORDER BY " + params.SortBy + " " + params.SortOrder

	// Add LIMIT and OFFSET
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryWithTimeout(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles: %w", err)
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		var publishedAt, readAt sql.NullTime

		err := rows.Scan(
			&article.ID,
			&article.FeedID,
			&article.Title,
			&article.Link,
			&article.Description,
			&article.Content,
			&article.Author,
			&publishedAt,
			&article.FetchedAt,
			&readAt,
			&article.IsRead,
			&article.IsStarred,
			&article.GUID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		if publishedAt.Valid {
			article.PublishedAt = &publishedAt.Time
		}
		if readAt.Valid {
			article.ReadAt = &readAt.Time
		}

		articles = append(articles, article)
	}

	return articles, nil
}

// MarkAsRead marks an article as read
func (s *ArticleService) MarkAsRead(ctx context.Context, id int, userID int) error {
	now := time.Now()

	query := `
		UPDATE rss_articles SET is_read = 1, read_at = ? WHERE id = ?
	`

	_, err := s.db.ExecWithTimeout(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark article as read: %w", err)
	}

	// Also update reading progress
	_, err = s.db.ExecWithTimeout(ctx,
		"INSERT OR REPLACE INTO rss_reading_progress (user_id, article_id, read_at) VALUES (?, ?, ?)",
		userID, id, now)
	if err != nil {
		s.logger.Error("Failed to update reading progress", "user_id", userID, "article_id", id, "error", err)
	}

	s.logger.Info("Marked article as read", "id", id, "user_id", userID)
	return nil
}

// ToggleStar toggles the starred status of an article
func (s *ArticleService) ToggleStar(ctx context.Context, id int) error {
	query := `
		UPDATE rss_articles SET is_starred = CASE WHEN is_starred = 1 THEN 0 ELSE 1 END
		WHERE id = ?
	`

	_, err := s.db.ExecWithTimeout(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to toggle article star: %w", err)
	}

	s.logger.Info("Toggled article star", "id", id)
	return nil
}

// getArticleTags retrieves tags for a specific article
func (s *ArticleService) getArticleTags(ctx context.Context, articleID int) ([]string, error) {
	query := `SELECT tag FROM rss_article_tags WHERE article_id = ? ORDER BY tag`

	rows, err := s.db.QueryWithTimeout(ctx, query, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query article tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}
