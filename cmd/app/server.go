package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server is the HTTP server with a graceful shutdown.
type Server struct {
	*http.Server
}

// Start runs ListenAndServe until SIGINT, then shuts down.
func (s *Server) Start() {
	log.Printf("server is running on port %s", s.Addr)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("closed server error %s", err.Error())
		}
	}()
	s.gracefulShutdown()
}

func (s *Server) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT)
	sig := <-quit
	log.Printf("server is shutting down %s", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.SetKeepAlivesEnabled(false)
	if err := s.Shutdown(ctx); err != nil {
		log.Printf("could not gracefully shutdown the server %s", err.Error())
	}
	log.Println("server stopped")
}
