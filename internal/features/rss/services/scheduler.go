package services

import (
	"context"
	"fmt"
	"sync"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/models"
	"time"
)

// SchedulerService handles periodic RSS feed updates
type SchedulerService struct {
	feedService    *FeedService
	articleService *ArticleService
	fetcherService *FetcherService
	logger         *core.Logger
	config         *models.SchedulerConfig
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(
	feedService *FeedService,
	articleService *ArticleService,
	fetcherService *FetcherService,
	logger *core.Logger,
	config *models.SchedulerConfig,
) *SchedulerService {
	return &SchedulerService{
		feedService:    feedService,
		articleService: articleService,
		fetcherService: fetcherService,
		logger:         logger,
		config:         config,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *SchedulerService) Start(ctx context.Context) error {
	s.logger.Info("Starting RSS feed scheduler", "interval", s.config.UpdateInterval)

	// Start the main update loop
	s.wg.Add(1)
	go s.updateLoop(ctx)

	return nil
}

// Stop gracefully stops the scheduler
func (s *SchedulerService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping RSS feed scheduler")
	close(s.stopChan)
	s.wg.Wait()
	return nil
}

// updateLoop runs the main update loop
func (s *SchedulerService) updateLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.UpdateInterval)
	defer ticker.Stop()

	// Do initial update
	s.updateAllFeeds(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler context cancelled")
			return
		case <-s.stopChan:
			s.logger.Info("Scheduler stop signal received")
			return
		case <-ticker.C:
			s.updateAllFeeds(ctx)
		}
	}
}

// updateAllFeeds updates all enabled feeds
func (s *SchedulerService) updateAllFeeds(ctx context.Context) {
	s.logger.Info("Starting feed update cycle")

    // Get all enabled feeds
	feeds, err := s.feedService.ListFeeds(ctx, true)
	if err != nil {
		s.logger.Error("Failed to get feeds for update", "error", err)
		return
	}

	if len(feeds) == 0 {
		s.logger.Info("No feeds to update")
		return
	}

	s.logger.Info("Updating feeds", "count", len(feeds))

	// Create worker pool for concurrent updates
	feedChan := make(chan *models.Feed, len(feeds))
	var wg sync.WaitGroup

	// Start workers
    for i := 0; i < s.config.MaxWorkers; i++ {
		wg.Add(1)
		go s.feedWorker(ctx, feedChan, &wg)
	}

	// Send feeds to workers
	for i := range feeds {
		feedChan <- &feeds[i]
	}
	close(feedChan)

	// Wait for all workers to complete
	wg.Wait()

	s.logger.Info("Feed update cycle completed")
}

// feedWorker processes feeds from the channel
func (s *SchedulerService) feedWorker(ctx context.Context, feedChan <-chan *models.Feed, wg *sync.WaitGroup) {
	defer wg.Done()

	for feed := range feedChan {
		if err := s.updateFeed(ctx, feed); err != nil {
			s.logger.Error("Failed to update feed", "feed_id", feed.ID, "feed_title", feed.Title, "error", err)
		}
	}
}

// updateFeed updates a single feed
func (s *SchedulerService) updateFeed(ctx context.Context, feed *models.Feed) error {
	s.logger.Info("Updating feed", "feed_id", feed.ID, "feed_title", feed.Title, "url", feed.URL)

	// Check if feed needs updating
	if feed.LastFetched != nil {
		timeSinceLastFetch := time.Since(*feed.LastFetched)
		if timeSinceLastFetch < time.Duration(feed.FetchInterval)*time.Second {
			s.logger.Debug("Feed doesn't need updating yet", "feed_id", feed.ID, "time_since_last", timeSinceLastFetch)
			return nil
		}
	}

    // Fetch the feed
	parsedFeed, err := s.fetcherService.FetchFeed(ctx, feed.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Update feed metadata if available
	if parsedFeed.Title != "" && feed.Title == "" {
		update := &models.FeedUpdate{
			Title: &parsedFeed.Title,
		}
		if parsedFeed.Description != "" {
			update.Description = &parsedFeed.Description
		}
		if parsedFeed.Link != "" {
			update.SiteURL = &parsedFeed.Link
		}

		_, err = s.feedService.UpdateFeed(ctx, feed.ID, update)
		if err != nil {
			s.logger.Error("Failed to update feed metadata", "feed_id", feed.ID, "error", err)
		}
	}

	// Process articles
	articlesAdded := 0
    for _, parsedArticle := range parsedFeed.Articles {
		// Check if article already exists
        exists, err := s.articleService.ExistsByFeedAndGUID(ctx, feed.ID, parsedArticle.GUID)
		if err != nil {
			s.logger.Error("Failed to check if article exists", "feed_id", feed.ID, "guid", parsedArticle.GUID, "error", err)
			continue
		}

		if exists {
			continue
		}

		// Create new article
		article := &models.ArticleCreate{
			FeedID:      feed.ID,
			Title:       parsedArticle.Title,
			Link:        parsedArticle.Link,
			Description: parsedArticle.Description,
			Content:     parsedArticle.Content,
			Author:      parsedArticle.Author,
			PublishedAt: parsedArticle.PublishedAt,
			GUID:        parsedArticle.GUID,
		}

		_, err = s.articleService.CreateArticle(ctx, article)
		if err != nil {
			s.logger.Error("Failed to create article", "feed_id", feed.ID, "guid", parsedArticle.GUID, "error", err)
			continue
		}

		articlesAdded++
	}

	// Update feed's last fetched time
	now := time.Now()
	update := &models.FeedUpdate{
		LastFetched: &now,
	}
	_, err = s.feedService.UpdateFeed(ctx, feed.ID, update)
	if err != nil {
		s.logger.Error("Failed to update feed last fetched time", "feed_id", feed.ID, "error", err)
	}

	s.logger.Info("Feed update completed", "feed_id", feed.ID, "articles_added", articlesAdded)
	return nil
}

// RefreshFeedByID fetches and processes a single feed by ID immediately
func (s *SchedulerService) RefreshFeedByID(ctx context.Context, feedID int) error {
    feed, err := s.feedService.GetFeed(ctx, feedID)
    if err != nil {
        return fmt.Errorf("failed to get feed %d: %w", feedID, err)
    }
    return s.updateFeed(ctx, feed)
}

// RefreshAll triggers an immediate update cycle for all enabled feeds
func (s *SchedulerService) RefreshAll(ctx context.Context) {
    s.updateAllFeeds(ctx)
}

// Removed local articleExists; using ArticleService.ExistsByFeedAndGUID instead
