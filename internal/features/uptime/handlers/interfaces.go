package handlers

import "the-ark/internal/features/uptime/models"

type ServerInterface interface {
	GetActiveWebsites() ([]models.Website, error)
	GetWebsiteByID(websiteID int) (*models.Website, error)
	GetLastWebsiteStatus(websiteID int) (*models.WebsiteStatus, error)
	CheckWebsite(website models.Website) error
}
