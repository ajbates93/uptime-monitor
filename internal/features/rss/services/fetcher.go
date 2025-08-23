package services

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"the-ark/internal/core"
	"the-ark/internal/features/rss/models"
	"time"
)

// RSSFeed represents the structure of an RSS feed
type RSSFeed struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

// Channel represents the channel element in RSS
type Channel struct {
	Title         string `xml:"title"`
	Link          string `xml:"link"`
	Description   string `xml:"description"`
	Language      string `xml:"language"`
	LastBuildDate string `xml:"lastBuildDate"`
	Items         []Item `xml:"item"`
}

// Item represents an RSS item/article
type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Content     string `xml:"content"`
	Author      string `xml:"author"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// AtomFeed represents the structure of an Atom feed
type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Link    []AtomLink  `xml:"link"`
	Entries []AtomEntry `xml:"entry"`
}

// AtomLink represents a link in Atom feeds
type AtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

// AtomEntry represents an entry in Atom feeds
type AtomEntry struct {
	Title   string     `xml:"title"`
	Link    []AtomLink `xml:"link"`
	Content string     `xml:"content"`
	Author  string     `xml:"author>name"`
	Updated string     `xml:"updated"`
	ID      string     `xml:"id"`
}

// FetcherService handles RSS feed fetching and parsing
type FetcherService struct {
	client *http.Client
	logger *core.Logger
	config *models.FetcherConfig
}

// FetcherConfig holds configuration for the fetcher
type FetcherConfig struct {
	UserAgent            string
	Timeout              time.Duration
	MaxConcurrentFetches int
}

// NewFetcherService creates a new fetcher service
func NewFetcherService(logger *core.Logger, config *models.FetcherConfig) *FetcherService {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &FetcherService{
		client: client,
		logger: logger,
		config: config,
	}
}

// FetchFeed fetches and parses an RSS feed
func (f *FetcherService) FetchFeed(ctx context.Context, feedURL string) (*models.ParsedFeed, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", f.config.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	// Make request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse feed based on content type
	contentType := resp.Header.Get("Content-Type")
	parsedFeed, err := f.parseFeed(body, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	f.logger.Info("Successfully fetched and parsed feed", "url", feedURL, "articles", len(parsedFeed.Articles))
	return parsedFeed, nil
}

// parseFeed parses RSS or Atom feed content
func (f *FetcherService) parseFeed(content []byte, contentType string) (*models.ParsedFeed, error) {
	// Try to parse as RSS first
	if strings.Contains(contentType, "rss") || strings.Contains(contentType, "xml") {
		var rssFeed RSSFeed
		if err := xml.Unmarshal(content, &rssFeed); err == nil {
			return f.parseRSSFeed(&rssFeed)
		}
	}

	// Try to parse as Atom
	var atomFeed AtomFeed
	if err := xml.Unmarshal(content, &atomFeed); err == nil {
		return f.parseAtomFeed(&atomFeed)
	}

	// Try generic XML parsing for RSS
	var rssFeed RSSFeed
	if err := xml.Unmarshal(content, &rssFeed); err == nil && rssFeed.Version != "" {
		return f.parseRSSFeed(&rssFeed)
	}

	return nil, fmt.Errorf("unable to parse feed as RSS or Atom")
}

// parseRSSFeed converts RSS feed to our internal format
func (f *FetcherService) parseRSSFeed(rss *RSSFeed) (*models.ParsedFeed, error) {
	feed := &models.ParsedFeed{
		Title:       rss.Channel.Title,
		Link:        rss.Channel.Link,
		Description: rss.Channel.Description,
		Language:    rss.Channel.Language,
		Articles:    make([]models.ParsedArticle, 0, len(rss.Channel.Items)),
	}

	for _, item := range rss.Channel.Items {
		article := models.ParsedArticle{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     item.Content,
			Author:      item.Author,
			GUID:        item.GUID,
		}

		// Parse publication date
		if item.PubDate != "" {
			if pubDate, err := parseDate(item.PubDate); err == nil {
				article.PublishedAt = &pubDate
			}
		}

		feed.Articles = append(feed.Articles, article)
	}

	return feed, nil
}

// parseAtomFeed converts Atom feed to our internal format
func (f *FetcherService) parseAtomFeed(atom *AtomFeed) (*models.ParsedFeed, error) {
	feed := &models.ParsedFeed{
		Title:       atom.Title,
		Description: "",
		Articles:    make([]models.ParsedArticle, 0, len(atom.Entries)),
	}

	// Find the main link
	for _, link := range atom.Link {
		if link.Rel == "" || link.Rel == "alternate" {
			feed.Link = link.Href
			break
		}
	}

	for _, entry := range atom.Entries {
		article := models.ParsedArticle{
			Title:       entry.Title,
			Description: "",
			Content:     entry.Content,
			Author:      entry.Author,
			GUID:        entry.ID,
		}

		// Find the main link for the entry
		for _, link := range entry.Link {
			if link.Rel == "" || link.Rel == "alternate" {
				article.Link = link.Href
				break
			}
		}

		// Parse publication date
		if entry.Updated != "" {
			if pubDate, err := parseDate(entry.Updated); err == nil {
				article.PublishedAt = &pubDate
			}
		}

		feed.Articles = append(feed.Articles, article)
	}

	return feed, nil
}

// parseDate parses various date formats commonly used in RSS feeds
func parseDate(dateStr string) (time.Time, error) {
	// Common RSS date formats
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 MST",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
