package handlers

import (
	"net/http"
	"path/filepath"
	"strings"
)

// StaticHandler serves static files from the views/assets directory
func StaticHandler(w http.ResponseWriter, r *http.Request) {
	// Remove the /assets prefix from the path
	path := strings.TrimPrefix(r.URL.Path, "/assets")

	// Construct the full file path
	fullPath := filepath.Join("views/assets", path)

	// Set proper MIME types based on file extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	}

	// Serve the file
	http.ServeFile(w, r, fullPath)
}
