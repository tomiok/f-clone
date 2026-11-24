// Package handler serves the meeting notebook: the list, a new meeting,
// the finished notepad, and its action items.
package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"fireflies-clone/internal/meeting"
	"fireflies-clone/internal/store"
	"fireflies-clone/internal/summary"
	"fireflies-clone/web"
)

// Handler is the HTTP API for meetings.
type Handler struct {
	store *store.Store
	pages *web.Pages
}

// New builds the meeting handler. Routes are registered in main.
func New(st *store.Store, pages *web.Pages) Handler {
	return Handler{store: st, pages: pages}
}

// List renders the notebook.
func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	h.pages.Render(w, http.StatusOK, "list", web.View{Title: "Meetings"})
}

// NewForm renders the form that starts a meeting.
func (h Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	h.pages.Render(w, http.StatusOK, "new", web.View{
		Title:        "New meeting",
		Participants: "Maya, Jordan, Priya",
	})
}

// Create stores an active meeting and opens the live page.
func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := web.ReadForm(w, r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	title := web.Clip(strings.TrimSpace(r.FormValue("title")), 120)
	participants := parseParticipants(r.FormValue("participants"))
	if title == "" {
		h.pages.Render(w, http.StatusBadRequest, "new", web.View{
			Title:        "New meeting",
			Error:        "Give the meeting a title.",
			DraftTitle:   title,
			Participants: r.FormValue("participants"),
		})
		return
	}
	if len(participants) == 0 {
		participants = []string{"Maya", "Jordan", "Priya"}
	}
	m, err := h.store.Create(title, participants, time.Now())
	if err != nil {
		h.pages.Fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
}

// Stop finishes the meeting and shows the transcribing page.
func (h Handler) Stop(w http.ResponseWriter, r *http.Request) {
	m, ok := h.finish(w, r.PathValue("id"))
	if !ok {
		return
	}
	h.pages.Render(w, http.StatusOK, "transcribing", web.View{
		Title:     "Transcribing",
		Meeting:   m,
		CurrentID: m.ID,
		Refresh:   "/meetings/" + m.ID,
	})
}

// Notepad renders the finished meeting.
func (h Handler) Notepad(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, ok := h.meeting(w, id)
	if !ok {
		return
	}
	if m.IsActive() {
		http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
		return
	}
	if m.IsProcessing() {
		m, ok = h.finish(w, id)
		if !ok {
			return
		}
	}
	h.pages.Render(w, http.StatusOK, "notepad", web.View{
		Title:     m.Title,
		Meeting:   m,
		CurrentID: m.ID,
	})
}

// Toggle flips an action item.
func (h Handler) Toggle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !web.ValidID(id) {
		h.pages.Missing(w)
		return
	}
	if _, err := h.store.ToggleAction(id, r.PathValue("actionID")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.pages.Missing(w)
			return
		}
		h.pages.Fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+id, http.StatusSeeOther)
}

// Delete removes a meeting and returns to the list.
func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !web.ValidID(id) {
		h.pages.Missing(w)
		return
	}
	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.pages.Missing(w)
			return
		}
		h.pages.Fail(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// finish runs the summary at most once. A completed meeting is returned as-is.
func (h Handler) finish(w http.ResponseWriter, id string) (meeting.Meeting, bool) {
	if !web.ValidID(id) {
		h.pages.Missing(w)
		return meeting.Meeting{}, false
	}
	m, already, err := h.store.BeginFinish(id, time.Now())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.pages.Missing(w)
			return meeting.Meeting{}, false
		}
		h.pages.Fail(w, err)
		return meeting.Meeting{}, false
	}
	if already {
		return m, true
	}
	m, err = h.store.Finish(id, summary.Build(m.Sentences))
	if err != nil {
		h.pages.Fail(w, err)
		return meeting.Meeting{}, false
	}
	return m, true
}

func (h Handler) meeting(w http.ResponseWriter, id string) (meeting.Meeting, bool) {
	if !web.ValidID(id) {
		h.pages.Missing(w)
		return meeting.Meeting{}, false
	}
	m, ok := h.store.Get(id)
	if !ok {
		h.pages.Missing(w)
		return meeting.Meeting{}, false
	}
	return m, true
}

func parseParticipants(raw string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		name := web.Clip(strings.TrimSpace(part), 40)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
		if len(out) == 12 {
			break
		}
	}
	return out
}
