package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (app *application) listWebsitesHandler(w http.ResponseWriter, r *http.Request) {
	websites, err := app.getActiveWebsites()
	if err != nil {
		app.logger.Error("Failed to get active websites", "error", err)
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"websites": websites}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, nil)
		return
	}
}

func (app *application) getWebsiteHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.logger.Error("Failed to get ID from query string", "error", err)
		app.serverErrorResponse(w, r, nil)
		return
	}

	website, err := app.getWebsiteByID(id)
	if err != nil {
		app.logger.Error("Failed to get website by ID", "error", err)
		app.serverErrorResponse(w, r, nil)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"website": website}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, nil)
		return
	}
}
