package main

import (
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"fireflies-clone/cmd/app"
	"fireflies-clone/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	d, err := app.NewDeps()
	if err != nil {
		return err
	}

	r := chi.NewRouter()
	srv := &http.Server{
		Addr:              ":" + d.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	routes(r, d)
	serv := app.Server{Server: srv}
	serv.Start()
	return nil
}

func routes(r *chi.Mux, d app.Deps) {
	static, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	r.Get("/", d.Meetings.List)
	r.Get("/new", d.Meetings.NewForm)
	r.Post("/meetings", d.Meetings.Create)
	r.Get("/meetings/{id}", d.Meetings.Notepad)
	r.Post("/meetings/{id}/stop", d.Meetings.Stop)
	r.Post("/meetings/{id}/actions/{actionID}", d.Meetings.Toggle)
	r.Post("/meetings/{id}/delete", d.Meetings.Delete)

	r.Get("/meetings/{id}/live", d.Conversation.Live)
	r.Post("/meetings/{id}/lines", d.Conversation.AddLine)
}
