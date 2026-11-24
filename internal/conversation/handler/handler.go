// Package handler serves the live conversation: the timer page and the
// lines appended while a meeting is active.
package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"fireflies-clone/internal/conversation"
	"fireflies-clone/internal/meeting"
	"fireflies-clone/internal/store"
	"fireflies-clone/web"
)

// Handler is the HTTP API for a meeting's conversation.
type Handler struct {
	store *store.Store
	pages *web.Pages
}

// New builds the conversation handler. Routes are registered in main.
func New(st *store.Store, pages *web.Pages) Handler {
	return Handler{store: st, pages: pages}
}

// Live renders the meeting while lines are still arriving.
func (h Handler) Live(w http.ResponseWriter, r *http.Request) {
	m, ok := h.meeting(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !m.IsActive() {
		http.Redirect(w, r, "/meetings/"+m.ID, http.StatusSeeOther)
		return
	}
	h.pages.Render(w, http.StatusOK, "live", h.liveView(m, ""))
}

// AddLine appends one line, either the next mock line or a typed one.
func (h Handler) AddLine(w http.ResponseWriter, r *http.Request) {
	if err := web.ReadForm(w, r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	m, ok := h.meeting(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !m.IsActive() {
		http.Redirect(w, r, "/meetings/"+m.ID, http.StatusSeeOther)
		return
	}
	if len(m.Sentences) >= 500 {
		h.pages.Render(w, http.StatusBadRequest, "live", h.liveView(m, "This meeting already has 500 lines."))
		return
	}

	var line conversation.Line
	if r.FormValue("source") == "script" {
		rec := conversation.NewRecorder(conversation.MockConversation{}, m.Participants, m.Sentences)
		next, more := rec.Next()
		if !more {
			http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
			return
		}
		line = next
	} else {
		line.Speaker = web.Clip(strings.TrimSpace(r.FormValue("speaker")), 40)
		line.Text = web.Clip(strings.TrimSpace(r.FormValue("text")), 2000)
		if line.Text == "" {
			h.pages.Render(w, http.StatusBadRequest, "live", h.liveView(m, "Write a line before adding it."))
			return
		}
		if line.Speaker == "" {
			line.Speaker = firstSpeaker(m)
		}
	}

	start := time.Since(m.CreatedAt).Seconds()
	if start < 0 {
		start = 0
	}
	// Resume keeps the transcript where stop left it.
	if r.FormValue("held") == "1" && len(m.Sentences) > 0 {
		start = m.Sentences[len(m.Sentences)-1].EndSec
	}
	id := m.ID
	var err error
	m, err = h.store.Append(id, meeting.Sentence{
		Speaker:  line.Speaker,
		Text:     line.Text,
		StartSec: start,
		EndSec:   start + 2,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotActive) {
			http.Redirect(w, r, "/meetings/"+id, http.StatusSeeOther)
			return
		}
		h.pages.Fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
}

func (h Handler) liveView(m meeting.Meeting, message string) web.View {
	_, more := conversation.NewRecorder(conversation.MockConversation{}, m.Participants, m.Sentences).Next()
	return web.View{
		Title:      m.Title,
		Meeting:    m,
		CurrentID:  m.ID,
		Error:      message,
		Speaker:    firstSpeaker(m),
		ScriptDone: !more,
	}
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

func firstSpeaker(m meeting.Meeting) string {
	if len(m.Participants) == 0 {
		return "Speaker"
	}
	return m.Participants[0]
}
