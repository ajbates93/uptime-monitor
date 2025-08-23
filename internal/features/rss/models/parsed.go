package models

import (
	"time"
)

// ParsedFeed represents a parsed RSS/Atom feed
type ParsedFeed struct {
	Title       string          `json:"title"`
	Link        string          `json:"link"`
	Description string          `json:"description"`
	Language    string          `json:"language"`
	Articles    []ParsedArticle `json:"articles"`
}

// ParsedArticle represents a parsed article from a feed
type ParsedArticle struct {
	Title       string     `json:"title"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	PublishedAt *time.Time `json:"published_at"`
	GUID        string     `json:"guid"`
}

// FetcherConfig holds configuration for the fetcher service
type FetcherConfig struct {
	UserAgent            string        `json:"user_agent"`
	Timeout              time.Duration `json:"timeout"`
	MaxConcurrentFetches int           `json:"max_concurrent_fetches"`
}
