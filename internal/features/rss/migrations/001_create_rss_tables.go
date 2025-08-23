package migrations

import (
	"the-ark/internal/core"
)

// Migration001CreateRSSTables creates the initial RSS tables
var Migration001CreateRSSTables = core.Migration{
	Version:     1,
	Name:        "create_rss_tables",
	Description: "Create initial RSS feed reader tables",
	UpSQL: `
		-- RSS feeds
		CREATE TABLE IF NOT EXISTS rss_feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			description TEXT,
			site_url TEXT,
			favicon_url TEXT,
			last_fetched TIMESTAMP,
			fetch_interval INTEGER DEFAULT 3600,
			enabled BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- Feed categories/tags
		CREATE TABLE IF NOT EXISTS rss_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			color TEXT DEFAULT '#3B82F6',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- Feed-category relationships
		CREATE TABLE IF NOT EXISTS rss_feed_categories (
			feed_id INTEGER REFERENCES rss_feeds(id) ON DELETE CASCADE,
			category_id INTEGER REFERENCES rss_categories(id) ON DELETE CASCADE,
			PRIMARY KEY (feed_id, category_id)
		);

		-- Articles from feeds
		CREATE TABLE IF NOT EXISTS rss_articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL REFERENCES rss_feeds(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			link TEXT NOT NULL,
			description TEXT,
			content TEXT,
			author TEXT,
			published_at TIMESTAMP,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			read_at TIMESTAMP,
			is_read BOOLEAN DEFAULT 0,
			is_starred BOOLEAN DEFAULT 0,
			guid TEXT NOT NULL,
			UNIQUE(feed_id, guid)
		);

		-- Article tags for filtering
		CREATE TABLE IF NOT EXISTS rss_article_tags (
			article_id INTEGER REFERENCES rss_articles(id) ON DELETE CASCADE,
			tag TEXT NOT NULL,
			PRIMARY KEY (article_id, tag)
		);

		-- Reading progress tracking
		CREATE TABLE IF NOT EXISTS rss_reading_progress (
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			article_id INTEGER NOT NULL REFERENCES rss_articles(id) ON DELETE CASCADE,
			read_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, article_id)
		);

		-- Create indexes for better performance
		CREATE INDEX IF NOT EXISTS idx_rss_feeds_url ON rss_feeds(url);
		CREATE INDEX IF NOT EXISTS idx_rss_feeds_enabled ON rss_feeds(enabled);
		CREATE INDEX IF NOT EXISTS idx_rss_feeds_last_fetched ON rss_feeds(last_fetched);
		CREATE INDEX IF NOT EXISTS idx_rss_articles_feed_id ON rss_articles(feed_id);
		CREATE INDEX IF NOT EXISTS idx_rss_articles_published_at ON rss_articles(published_at);
		CREATE INDEX IF NOT EXISTS idx_rss_articles_is_read ON rss_articles(is_read);
		CREATE INDEX IF NOT EXISTS idx_rss_articles_is_starred ON rss_articles(is_starred);
		CREATE INDEX IF NOT EXISTS idx_rss_articles_guid ON rss_articles(guid);
		CREATE INDEX IF NOT EXISTS idx_rss_article_tags_tag ON rss_article_tags(tag);
		CREATE INDEX IF NOT EXISTS idx_rss_reading_progress_user_article ON rss_reading_progress(user_id, article_id);
	`,
	DownSQL: `
		-- Drop indexes
		DROP INDEX IF EXISTS idx_rss_reading_progress_user_article;
		DROP INDEX IF EXISTS idx_rss_article_tags_tag;
		DROP INDEX IF EXISTS idx_rss_articles_guid;
		DROP INDEX IF EXISTS idx_rss_articles_is_starred;
		DROP INDEX IF EXISTS idx_rss_articles_is_read;
		DROP INDEX IF EXISTS idx_rss_articles_published_at;
		DROP INDEX IF EXISTS idx_rss_articles_feed_id;
		DROP INDEX IF EXISTS idx_rss_feeds_last_fetched;
		DROP INDEX IF EXISTS idx_rss_feeds_enabled;
		DROP INDEX IF EXISTS idx_rss_feeds_url;

		-- Drop tables
		DROP TABLE IF EXISTS rss_reading_progress;
		DROP TABLE IF EXISTS rss_article_tags;
		DROP TABLE IF EXISTS rss_articles;
		DROP TABLE IF EXISTS rss_feed_categories;
		DROP TABLE IF EXISTS rss_categories;
		DROP TABLE IF EXISTS rss_feeds;
	`,
}
