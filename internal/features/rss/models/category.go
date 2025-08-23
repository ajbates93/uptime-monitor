package models

import (
	"time"
)

// Category represents a category for organizing RSS feeds
type Category struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	FeedCount int       `json:"feed_count,omitempty"`
}

// CategoryCreate represents the data needed to create a new category
type CategoryCreate struct {
	Name  string `json:"name" validate:"required,min=1,max=50"`
	Color string `json:"color" validate:"required,hexcolor"`
}

// CategoryUpdate represents the data needed to update a category
type CategoryUpdate struct {
	Name  *string `json:"name" validate:"omitempty,min=1,max=50"`
	Color *string `json:"color" validate:"omitempty,hexcolor"`
}

// CategoryStats represents statistics for a category
type CategoryStats struct {
	CategoryID      int `json:"category_id"`
	FeedCount       int `json:"feed_count"`
	TotalArticles   int `json:"total_articles"`
	UnreadArticles  int `json:"unread_articles"`
	StarredArticles int `json:"starred_articles"`
}
