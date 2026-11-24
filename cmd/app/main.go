package main

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fireflies-clone/internal/httpx"
	"fireflies-clone/internal/store"
)

func main() {
	log.Fatal(run())
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	path := os.Getenv("DATA_FILE")
	if path == "" {
		path = filepath.Join("data", "meetings.json")
	}

	st, err := store.Open(path)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpx.New(st),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
