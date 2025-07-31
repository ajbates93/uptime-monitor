package handlers

import "the-ark/internal/server/models"

// ServerInterface defines the methods that handlers need from the server
type ServerInterface interface {
	GetActiveWebsites() ([]models.Website, error)
	GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error)
	GetWebsiteByID(websiteID int) (*models.Website, error)
	CheckWebsite(website models.Website) error
}
