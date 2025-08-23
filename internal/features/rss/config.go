package rss

import (
	"fmt"
	"the-ark/internal/core"
)

// Config represents RSS feature configuration
type Config struct {
	Enabled              bool
	FetchInterval        int
	MaxArticlesPerFeed   int
	ImageCacheSize       string
	CleanupInterval      int
	UserAgent            string
	MaxConcurrentFetches int
}

// NewConfig creates RSS config from core config
func NewConfig(coreConfig *core.Config) *Config {
	return &Config{
		Enabled:              coreConfig.Features.RSS.Enabled,
		FetchInterval:        coreConfig.Features.RSS.FetchInterval,
		MaxArticlesPerFeed:   coreConfig.Features.RSS.MaxArticlesPerFeed,
		ImageCacheSize:       coreConfig.Features.RSS.ImageCacheSize,
		CleanupInterval:      coreConfig.Features.RSS.CleanupInterval,
		UserAgent:            coreConfig.Features.RSS.UserAgent,
		MaxConcurrentFetches: coreConfig.Features.RSS.MaxConcurrentFetches,
	}
}

// Validate validates the RSS configuration
func (c *Config) Validate() error {
	if c.FetchInterval < 300 || c.FetchInterval > 86400 {
		return fmt.Errorf("fetch interval must be between 300 and 86400 seconds")
	}

	if c.MaxArticlesPerFeed < 10 || c.MaxArticlesPerFeed > 1000 {
		return fmt.Errorf("max articles per feed must be between 10 and 1000")
	}

	if c.MaxConcurrentFetches < 1 || c.MaxConcurrentFetches > 20 {
		return fmt.Errorf("max concurrent fetches must be between 1 and 20")
	}

	return nil
}
