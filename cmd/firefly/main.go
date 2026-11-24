package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fireflies-clone/internal/httpx"
	"fireflies-clone/internal/store"
)

func main() {
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
		slog.Error("open store", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpx.New(st),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}
