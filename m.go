package main

import (
	"log/slog"
	"net/http"
	"rocketseat/api"
	"rocketseat/models"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("failed to execute code", "error", err)
	}

	slog.Info("all systems offline")
}

func run() error {
	db := models.DB[*models.User]{}
	handler := api.NewHandler(db)

	s := http.Server{
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
		WriteTimeout: time.Second * 10,
		Addr:         "localhost:8080",
		Handler:      handler,
	}

	if err := s.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
