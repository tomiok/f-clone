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
	h := d.Handler
	static, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	r.Get("/", h.List)
	r.Get("/new", h.NewForm)
	r.Post("/meetings", h.Create)
	r.Get("/meetings/{id}/live", h.Live)
	r.Post("/meetings/{id}/lines", h.AddLine)
	r.Post("/meetings/{id}/stop", h.Stop)
	r.Get("/meetings/{id}", h.Notepad)
	r.Post("/meetings/{id}/actions/{actionID}", h.Toggle)
	r.Post("/meetings/{id}/delete", h.Delete)
}
