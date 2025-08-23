package models

import (
	"time"
)

// Article represents an article from an RSS feed
type Article struct {
	ID          int        `json:"id"`
	FeedID      int        `json:"feed_id"`
	Title       string     `json:"title"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	PublishedAt *time.Time `json:"published_at"`
	FetchedAt   time.Time  `json:"fetched_at"`
	ReadAt      *time.Time `json:"read_at"`
	IsRead      bool       `json:"is_read"`
	IsStarred   bool       `json:"is_starred"`
	GUID        string     `json:"guid"`
	Tags        []string   `json:"tags,omitempty"`
	Feed        *Feed      `json:"feed,omitempty"`
}

// ArticleCreate represents the data needed to create a new article
type ArticleCreate struct {
	FeedID      int        `json:"feed_id" validate:"required"`
	Title       string     `json:"title" validate:"required"`
	Link        string     `json:"link" validate:"required,url"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	PublishedAt *time.Time `json:"published_at"`
	GUID        string     `json:"guid" validate:"required"`
	Tags        []string   `json:"tags"`
}

// ArticleUpdate represents the data needed to update an article
type ArticleUpdate struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Content     *string    `json:"content"`
	Author      *string    `json:"author"`
	PublishedAt *time.Time `json:"published_at"`
	IsRead      *bool      `json:"is_read"`
	IsStarred   *bool      `json:"is_starred"`
	Tags        []string   `json:"tags"`
}

// ArticleListParams represents parameters for listing articles
type ArticleListParams struct {
	FeedID     *int       `json:"feed_id"`
	CategoryID *int       `json:"category_id"`
	IsRead     *bool      `json:"is_read"`
	IsStarred  *bool      `json:"is_starred"`
	Search     string     `json:"search"`
	FromDate   *time.Time `json:"from_date"`
	ToDate     *time.Time `json:"to_date"`
	Tags       []string   `json:"tags"`
	Limit      int        `json:"limit" validate:"min=1,max=100"`
	Offset     int        `json:"offset" validate:"min=0"`
	SortBy     string     `json:"sort_by" validate:"oneof=published_at fetched_at title feed_title"`
	SortOrder  string     `json:"sort_order" validate:"oneof=asc desc"`
}

// ArticleStats represents article statistics
type ArticleStats struct {
	TotalArticles    int `json:"total_articles"`
	UnreadArticles   int `json:"unread_articles"`
	StarredArticles  int `json:"starred_articles"`
	TodayArticles    int `json:"today_articles"`
	ThisWeekArticles int `json:"this_week_articles"`
}
