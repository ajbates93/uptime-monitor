package rss

import (
	"context"
	"fmt"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/handlers"
	"the-ark/internal/features/rss/migrations"
	"the-ark/internal/features/rss/models"
	"the-ark/internal/features/rss/services"
	"time"
)

// Feature represents the RSS feed reader feature
type Feature struct {
	*core.BaseFeature
	config           *Config
	migrationMgr     *migrations.Manager
	feedService      *services.FeedService
	articleService   *services.ArticleService
	fetcherService   *services.FetcherService
	schedulerService *services.SchedulerService
	handlers         *handlers.Handlers
}

// NewFeature creates a new RSS feature
func NewFeature(logger *core.Logger, db *core.Database, config *Config) *Feature {
	// Create migration manager
	migrationMgr := migrations.NewManager(db, logger)

	// Create services
	feedService := services.NewFeedService(db, logger)
	articleService := services.NewArticleService(db, logger)

	// Create fetcher service
	fetcherConfig := &models.FetcherConfig{
		UserAgent:            config.UserAgent,
		Timeout:              30 * time.Second,
		MaxConcurrentFetches: config.MaxConcurrentFetches,
	}
	fetcherService := services.NewFetcherService(logger, fetcherConfig)

	// Create scheduler service
	schedulerConfig := models.DefaultSchedulerConfig()
	schedulerConfig.UpdateInterval = time.Duration(config.FetchInterval) * time.Second
	schedulerService := services.NewSchedulerService(feedService, articleService, fetcherService, logger, schedulerConfig)

	// Create handlers
	handlers := handlers.NewHandlers(logger, feedService, articleService)

	feature := &Feature{
		BaseFeature:      core.NewBaseFeature("rss", "RSS Feed Reader", config.Enabled, logger, db, config),
		config:           config,
		migrationMgr:     migrationMgr,
		feedService:      feedService,
		articleService:   articleService,
		fetcherService:   fetcherService,
		schedulerService: schedulerService,
		handlers:         handlers,
	}

	return feature
}

// Init initializes the RSS feature
func (f *Feature) Init(ctx context.Context) error {
	if err := f.BaseFeature.Init(ctx); err != nil {
		return err
	}

	// Validate configuration
	if err := f.config.Validate(); err != nil {
		return err
	}

	// Run migrations
	if err := f.migrationMgr.Migrate(ctx); err != nil {
		return err
	}

	// Start scheduler if feature is enabled
	if f.config.Enabled {
		if err := f.schedulerService.Start(ctx); err != nil {
			return fmt.Errorf("failed to start RSS scheduler: %w", err)
		}
		f.Logger().Info("RSS scheduler started")
	}

	f.Logger().Info("RSS feature initialized successfully")
	return nil
}

// Routes returns the HTTP routes for the RSS feature
func (f *Feature) Routes() []core.Route {
	return []core.Route{
		// Feed management
		{Method: "GET", Path: "/rss/feeds", Handler: f.handlers.ListFeeds},
		{Method: "POST", Path: "/rss/feeds", Handler: f.handlers.CreateFeed},
		{Method: "GET", Path: "/rss/feeds/{id}", Handler: f.handlers.GetFeed},
		{Method: "PUT", Path: "/rss/feeds/{id}", Handler: f.handlers.UpdateFeed},
		{Method: "DELETE", Path: "/rss/feeds/{id}", Handler: f.handlers.DeleteFeed},
		{Method: "POST", Path: "/rss/feeds/{id}/refresh", Handler: f.handlers.RefreshFeed},

		// Article management
		{Method: "GET", Path: "/rss/articles", Handler: f.handlers.ListArticles},
		{Method: "GET", Path: "/rss/articles/{id}", Handler: f.handlers.GetArticle},
		{Method: "PUT", Path: "/rss/articles/{id}/read", Handler: f.handlers.MarkAsRead},
		{Method: "PUT", Path: "/rss/articles/{id}/star", Handler: f.handlers.ToggleStar},
		{Method: "GET", Path: "/rss/articles/{id}/content", Handler: f.handlers.GetArticleContent},

		// Category management
		{Method: "GET", Path: "/rss/categories", Handler: f.handlers.ListCategories},
		{Method: "POST", Path: "/rss/categories", Handler: f.handlers.CreateCategory},
		{Method: "PUT", Path: "/rss/categories/{id}", Handler: f.handlers.UpdateCategory},
		{Method: "DELETE", Path: "/rss/categories/{id}", Handler: f.handlers.DeleteCategory},

		// Statistics and dashboard
		{Method: "GET", Path: "/rss/stats", Handler: f.handlers.GetStats},
		{Method: "GET", Path: "/rss/dashboard", Handler: f.handlers.GetDashboard},

		// Web interface routes
		{Method: "GET", Path: "/rss", Handler: f.handlers.RSSDashboard},
		{Method: "GET", Path: "/rss/feeds/add", Handler: f.handlers.AddFeedPage},
		{Method: "GET", Path: "/rss/articles/{id}", Handler: f.handlers.ViewArticlePage},
	}
}

// Shutdown gracefully shuts down the RSS feature
func (f *Feature) Shutdown(ctx context.Context) error {
	f.Logger().Info("Shutting down RSS feature")

	// Stop scheduler if it's running
	if f.config.Enabled && f.schedulerService != nil {
		if err := f.schedulerService.Stop(ctx); err != nil {
			f.Logger().Error("Failed to stop RSS scheduler", "error", err)
		}
	}

	return f.BaseFeature.Shutdown(ctx)
}

// GetMigrationManager returns the migration manager for this feature
func (f *Feature) GetMigrationManager() *migrations.Manager {
	return f.migrationMgr
}

// GetFeedService returns the feed service
func (f *Feature) GetFeedService() *services.FeedService {
	return f.feedService
}

// GetArticleService returns the article service
func (f *Feature) GetArticleService() *services.ArticleService {
	return f.articleService
}

// GetFetcherService returns the fetcher service
func (f *Feature) GetFetcherService() *services.FetcherService {
	return f.fetcherService
}

// GetSchedulerService returns the scheduler service
func (f *Feature) GetSchedulerService() *services.SchedulerService {
	return f.schedulerService
}
