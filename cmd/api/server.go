package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:     fmt.Sprintf(":%d", app.config.port),
		ErrorLog: slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
		Handler:  app.routes(),
	}

	app.logger.Info("Starting server", "addr", srv.Addr)

	// TODO: will implement graceful shutdown later

	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	app.logger.Info("Server stopped")
	return nil
}
