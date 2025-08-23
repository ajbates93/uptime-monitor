package models

import (
	"time"
)

// Feed represents an RSS feed
type Feed struct {
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Description   string     `json:"description"`
	SiteURL       string     `json:"site_url"`
	FaviconURL    string     `json:"favicon_url"`
	LastFetched   *time.Time `json:"last_fetched"`
	FetchInterval int        `json:"fetch_interval"` // seconds
	Enabled       bool       `json:"enabled"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	Categories    []Category `json:"categories,omitempty"`
}

// FeedCreate represents the data needed to create a new feed
type FeedCreate struct {
	Title         string `json:"title" validate:"required"`
	URL           string `json:"url" validate:"required,url"`
	Description   string `json:"description"`
	SiteURL       string `json:"site_url" validate:"omitempty,url"`
	FaviconURL    string `json:"favicon_url" validate:"omitempty,url"`
	FetchInterval int    `json:"fetch_interval" validate:"min=300,max=86400"` // 5 minutes to 24 hours
	CategoryIDs   []int  `json:"category_ids"`
}

// FeedUpdate represents the data needed to update a feed
type FeedUpdate struct {
	Title         *string    `json:"title" validate:"omitempty,min=1"`
	Description   *string    `json:"description"`
	SiteURL       *string    `json:"site_url" validate:"omitempty,url"`
	FaviconURL    *string    `json:"favicon_url" validate:"omitempty,url"`
	FetchInterval *int       `json:"fetch_interval" validate:"omitempty,min=300,max=86400"`
	Enabled       *bool      `json:"enabled"`
	LastFetched   *time.Time `json:"last_fetched"`
	CategoryIDs   []int      `json:"category_ids"`
}

// FeedStats represents statistics for a feed
type FeedStats struct {
	FeedID          int        `json:"feed_id"`
	TotalArticles   int        `json:"total_articles"`
	UnreadArticles  int        `json:"unread_articles"`
	StarredArticles int        `json:"starred_articles"`
	LastArticleDate *time.Time `json:"last_article_date"`
}
