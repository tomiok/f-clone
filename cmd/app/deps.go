package app

import (
	"fmt"
	"os"
	"path/filepath"

	"fireflies-clone/internal/httpx"
	"fireflies-clone/internal/store"
)

// Deps is everything main needs to serve the notebook.
type Deps struct {
	Port    string
	Handler httpx.Handler
}

// NewDeps opens the meeting store and builds the handler. Routes stay in main.
func NewDeps() (Deps, error) {
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
		return Deps{}, fmt.Errorf("open store: %w", err)
	}
	return Deps{
		Port:    port,
		Handler: httpx.New(st),
	}, nil
}
