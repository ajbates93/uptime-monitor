package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"
    "time"
    "the-ark/internal/auth"
    "the-ark/internal/core"
    "the-ark/internal/features/rss/models"
    "the-ark/internal/features/rss/services"
    viewrss "the-ark/views/rss"

    "github.com/go-chi/chi/v5"
)

// Handlers contains all RSS feature HTTP handlers
type Handlers struct {
	logger         *core.Logger
	feedService    *services.FeedService
	articleService *services.ArticleService
    scheduler      *services.SchedulerService
}

// NewHandlers creates a new handlers instance
func NewHandlers(logger *core.Logger, feedService *services.FeedService, articleService *services.ArticleService, scheduler *services.SchedulerService) *Handlers {
	return &Handlers{
		logger:         logger,
		feedService:    feedService,
		articleService: articleService,
        scheduler:      scheduler,
	}
}

// Feed management handlers
func (h *Handlers) ListFeeds(w http.ResponseWriter, r *http.Request) {
    feeds, err := h.feedService.ListFeeds(r.Context(), false)
    if err != nil {
        h.logger.Error("Failed to list feeds", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(feeds)
}

func (h *Handlers) CreateFeed(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Title         string `json:"title"`
        URL           string `json:"url"`
        Description   string `json:"description"`
        SiteURL       string `json:"site_url"`
        FaviconURL    string `json:"favicon_url"`
        FetchInterval int    `json:"fetch_interval"`
        CategoryIDs   []int  `json:"category_ids"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    if payload.URL == "" {
        http.Error(w, "url is required", http.StatusBadRequest)
        return
    }
    create := &models.FeedCreate{
        Title:         payload.Title,
        URL:           payload.URL,
        Description:   payload.Description,
        SiteURL:       payload.SiteURL,
        FaviconURL:    payload.FaviconURL,
        FetchInterval: payload.FetchInterval,
        CategoryIDs:   payload.CategoryIDs,
    }
    feed, err := h.feedService.CreateFeed(r.Context(), create)
    if err != nil {
        h.logger.Error("Failed to create feed", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // Trigger immediate refresh in background with a detached context
    go func(feedID int) {
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
        defer cancel()
        if err := h.scheduler.RefreshFeedByID(ctx, feedID); err != nil {
            h.logger.Error("Post-create refresh failed", "feed_id", feedID, "error", err)
        }
    }(feed.ID)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(feed)
}

func (h *Handlers) GetFeed(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    feed, err := h.feedService.GetFeed(r.Context(), id)
    if err != nil {
        h.logger.Error("Failed to get feed", "id", id, "error", err)
        http.Error(w, "Not Found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(feed)
}

func (h *Handlers) UpdateFeed(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    var payload struct {
        Title         *string `json:"title"`
        Description   *string `json:"description"`
        SiteURL       *string `json:"site_url"`
        FaviconURL    *string `json:"favicon_url"`
        FetchInterval *int    `json:"fetch_interval"`
        Enabled       *bool   `json:"enabled"`
        CategoryIDs   []int   `json:"category_ids"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    update := &models.FeedUpdate{
        Title:         payload.Title,
        Description:   payload.Description,
        SiteURL:       payload.SiteURL,
        FaviconURL:    payload.FaviconURL,
        FetchInterval: payload.FetchInterval,
        Enabled:       payload.Enabled,
        CategoryIDs:   payload.CategoryIDs,
    }
    feed, err := h.feedService.UpdateFeed(r.Context(), id, update)
    if err != nil {
        h.logger.Error("Failed to update feed", "id", id, "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(feed)
}

func (h *Handlers) DeleteFeed(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    if err := h.feedService.DeleteFeed(r.Context(), id); err != nil {
        h.logger.Error("Failed to delete feed", "id", id, "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) RefreshFeed(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    if err := h.scheduler.RefreshFeedByID(r.Context(), id); err != nil {
        h.logger.Error("Failed to refresh feed", "id", id, "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusAccepted)
}

// RefreshAllFeeds triggers a refresh of all enabled feeds
func (h *Handlers) RefreshAllFeeds(w http.ResponseWriter, r *http.Request) {
    // Run in background with a detached context that survives request end
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
        defer cancel()
        h.scheduler.RefreshAll(ctx)
    }()
    w.WriteHeader(http.StatusAccepted)
}

// Article management handlers
func (h *Handlers) ListArticles(w http.ResponseWriter, r *http.Request) {
    // Parse query params
    q := r.URL.Query()
    var feedIDPtr *int
    if s := q.Get("feed_id"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            feedIDPtr = &v
        }
    }
    limit := 50
    if s := q.Get("limit"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            limit = v
        }
    }
    offset := 0
    if s := q.Get("offset"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            offset = v
        }
    }
    sortBy := q.Get("sort_by")
    if sortBy == "" {
        sortBy = "published_at"
    }
    sortOrder := q.Get("sort_order")
    if sortOrder == "" {
        sortOrder = "desc"
    }

    params := &models.ArticleListParams{
        FeedID:    feedIDPtr,
        Limit:     limit,
        Offset:    offset,
        SortBy:    sortBy,
        SortOrder: sortOrder,
    }
    articles, err := h.articleService.ListArticles(r.Context(), params)
    if err != nil {
        h.logger.Error("Failed to list articles", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(articles)
}

func (h *Handlers) GetArticle(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    article, err := h.articleService.GetArticle(r.Context(), id)
    if err != nil {
        h.logger.Error("Failed to get article", "id", id, "error", err)
        http.Error(w, "Not Found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(article)
}

func (h *Handlers) MarkAsRead(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    user := auth.GetUserFromContext(r)
    if user.IsAnonymous() {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    if err := h.articleService.MarkAsRead(r.Context(), id, user.ID); err != nil {
        h.logger.Error("Failed to mark as read", "id", id, "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ToggleStar(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    if err := h.articleService.ToggleStar(r.Context(), id); err != nil {
        h.logger.Error("Failed to toggle star", "id", id, "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetArticleContent(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    article, err := h.articleService.GetArticle(r.Context(), id)
    if err != nil {
        h.logger.Error("Failed to get article content", "id", id, "error", err)
        http.Error(w, "Not Found", http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "id":      article.ID,
        "title":   article.Title,
        "content": article.Content,
        "link":    article.Link,
    })
}

// Category management handlers
func (h *Handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category listing
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category creation
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category update
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement category deletion
	w.WriteHeader(http.StatusNotImplemented)
}

// Statistics and dashboard handlers
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get stats
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get dashboard
	w.WriteHeader(http.StatusNotImplemented)
}

// Web interface handlers
func (h *Handlers) RSSDashboard(w http.ResponseWriter, r *http.Request) {
    component := viewrss.RSSDashboard()
    component.Render(r.Context(), w)
}

func (h *Handlers) AddFeedPage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement add feed page
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) ViewArticlePage(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement view article page
	w.WriteHeader(http.StatusNotImplemented)
}
