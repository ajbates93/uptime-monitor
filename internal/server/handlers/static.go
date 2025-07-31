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

	// Serve the file
	http.ServeFile(w, r, fullPath)
}
