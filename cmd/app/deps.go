package app

import (
	"fmt"
	"os"
	"path/filepath"

	conversationhandler "fireflies-clone/internal/conversation/handler"
	meetinghandler "fireflies-clone/internal/meeting/handler"
	"fireflies-clone/internal/store"
	"fireflies-clone/web"
)

// Deps is everything main needs to serve the notebook.
type Deps struct {
	Port         string
	Meetings     meetinghandler.Handler
	Conversation conversationhandler.Handler
}

// NewDeps opens the meeting store and builds one handler per business unit.
// Routes stay in main.
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
	pages := web.NewPages(st.List)
	return Deps{
		Port:         port,
		Meetings:     meetinghandler.New(st, pages),
		Conversation: conversationhandler.New(st, pages),
	}, nil
}
