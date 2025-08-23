package services

import (
	"context"
	"testing"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/models"
	"time"
)

func TestFetcherService(t *testing.T) {
	// Create a test logger
	logger := core.NewLogger()

	// Create test config
	config := &models.FetcherConfig{
		UserAgent:            "The Ark RSS Reader Test/1.0",
		Timeout:              30 * time.Second,
		MaxConcurrentFetches: 5,
	}

	// Create fetcher service
	fetcher := NewFetcherService(logger, config)

	// Test with a real RSS feed
	ctx := context.Background()
	feedURL := "https://feeds.bbci.co.uk/news/rss.xml" // BBC News RSS feed

	parsedFeed, err := fetcher.FetchFeed(ctx, feedURL)
	if err != nil {
		t.Skipf("Skipping test - failed to fetch feed (this is expected in CI): %v", err)
	}

	// Verify feed structure
	if parsedFeed.Title == "" {
		t.Error("Expected feed title to be set")
	}

	if len(parsedFeed.Articles) == 0 {
		t.Error("Expected feed to have articles")
	}

	// Verify first article
	firstArticle := parsedFeed.Articles[0]
	if firstArticle.Title == "" {
		t.Error("Expected article title to be set")
	}

	if firstArticle.Link == "" {
		t.Error("Expected article link to be set")
	}

	if firstArticle.GUID == "" {
		t.Error("Expected article GUID to be set")
	}

	t.Logf("Successfully parsed feed: %s with %d articles", parsedFeed.Title, len(parsedFeed.Articles))
}

func TestSchedulerConfig(t *testing.T) {
	config := models.DefaultSchedulerConfig()

	if config.UpdateInterval != 1*time.Hour {
		t.Errorf("Expected update interval to be 1 hour, got %v", config.UpdateInterval)
	}

	if config.MaxWorkers != 5 {
		t.Errorf("Expected max workers to be 5, got %d", config.MaxWorkers)
	}

	if config.RetryAttempts != 3 {
		t.Errorf("Expected retry attempts to be 3, got %d", config.RetryAttempts)
	}

	if config.RetryDelay != 5*time.Minute {
		t.Errorf("Expected retry delay to be 5 minutes, got %v", config.RetryDelay)
	}
}
