// Package httpx serves the f - clone notebook.
package httpx

import (
	"bytes"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"fireflies-clone/internal/capture"
	"fireflies-clone/internal/meeting"
	"fireflies-clone/internal/store"
	"fireflies-clone/internal/summary"
	"fireflies-clone/web"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// view is the data passed to every template.
type view struct {
	Title        string
	Meetings     []meeting.Meeting
	Meeting      meeting.Meeting
	CurrentID    string
	Error        string
	DraftTitle   string
	Participants string
	Speaker      string
	ScriptDone   bool
	Refresh      string
}

// New returns the application handler.
func New(st *store.Store) http.Handler {
	h := &handler{store: st, tmpl: mustTemplates()}
	static, err := fs.Sub(web.FS, "static")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /", h.list)
	mux.HandleFunc("GET /new", h.newForm)
	mux.HandleFunc("POST /meetings", h.create)
	mux.HandleFunc("GET /meetings/{id}/live", h.live)
	mux.HandleFunc("POST /meetings/{id}/lines", h.addLine)
	mux.HandleFunc("POST /meetings/{id}/stop", h.stop)
	mux.HandleFunc("GET /meetings/{id}", h.notepad)
	mux.HandleFunc("POST /meetings/{id}/actions/{actionID}", h.toggle)
	mux.HandleFunc("POST /meetings/{id}/delete", h.remove)
	return mux
}

type handler struct {
	store *store.Store
	tmpl  *template.Template
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "list", view{Title: "Meetings"})
}

func (h *handler) newForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "new", view{
		Title:        "New meeting",
		Participants: "Maya, Jordan, Priya",
	})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	if err := readForm(w, r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	title := clip(strings.TrimSpace(r.FormValue("title")), 120)
	participants := parseParticipants(r.FormValue("participants"))
	if title == "" {
		h.render(w, http.StatusBadRequest, "new", view{
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
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
}

func (h *handler) live(w http.ResponseWriter, r *http.Request) {
	m, ok := h.meeting(w, r.PathValue("id"))
	if !ok {
		return
	}
	if !m.IsActive() {
		http.Redirect(w, r, "/meetings/"+m.ID, http.StatusSeeOther)
		return
	}
	h.render(w, http.StatusOK, "live", h.liveView(m, ""))
}

func (h *handler) addLine(w http.ResponseWriter, r *http.Request) {
	if err := readForm(w, r); err != nil {
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
		h.render(w, http.StatusBadRequest, "live", h.liveView(m, "This meeting already has 500 lines."))
		return
	}

	var line capture.Line
	if r.FormValue("source") == "script" {
		rec := capture.NewRecorder(m.Participants, m.Sentences)
		next, more := rec.Next()
		if !more {
			http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
			return
		}
		line = next
	} else {
		line.Speaker = clip(strings.TrimSpace(r.FormValue("speaker")), 40)
		line.Text = clip(strings.TrimSpace(r.FormValue("text")), 2000)
		if line.Text == "" {
			h.render(w, http.StatusBadRequest, "live", h.liveView(m, "Write a line before adding it."))
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
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+m.ID+"/live", http.StatusSeeOther)
}

func (h *handler) stop(w http.ResponseWriter, r *http.Request) {
	m, ok := h.finish(w, r.PathValue("id"))
	if !ok {
		return
	}
	h.render(w, http.StatusOK, "transcribing", view{
		Title:     "Transcribing",
		Meeting:   m,
		CurrentID: m.ID,
		Refresh:   "/meetings/" + m.ID,
	})
}

func (h *handler) notepad(w http.ResponseWriter, r *http.Request) {
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
	h.render(w, http.StatusOK, "notepad", view{
		Title:     m.Title,
		Meeting:   m,
		CurrentID: m.ID,
	})
}

func (h *handler) toggle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		h.missing(w)
		return
	}
	if _, err := h.store.ToggleAction(id, r.PathValue("actionID")); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.missing(w)
			return
		}
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/meetings/"+id, http.StatusSeeOther)
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validID(id) {
		h.missing(w)
		return
	}
	if err := h.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.missing(w)
			return
		}
		h.fail(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// finish runs the summary at most once. A completed meeting is returned as-is.
func (h *handler) finish(w http.ResponseWriter, id string) (meeting.Meeting, bool) {
	if !validID(id) {
		h.missing(w)
		return meeting.Meeting{}, false
	}
	m, already, err := h.store.BeginFinish(id, time.Now())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			h.missing(w)
			return meeting.Meeting{}, false
		}
		h.fail(w, err)
		return meeting.Meeting{}, false
	}
	if already {
		return m, true
	}
	m, err = h.store.Finish(id, summary.Build(m.Sentences))
	if err != nil {
		h.fail(w, err)
		return meeting.Meeting{}, false
	}
	return m, true
}

func (h *handler) meeting(w http.ResponseWriter, id string) (meeting.Meeting, bool) {
	if !validID(id) {
		h.missing(w)
		return meeting.Meeting{}, false
	}
	m, ok := h.store.Get(id)
	if !ok {
		h.missing(w)
		return meeting.Meeting{}, false
	}
	return m, true
}

func (h *handler) liveView(m meeting.Meeting, message string) view {
	_, more := capture.NewRecorder(m.Participants, m.Sentences).Next()
	return view{
		Title:      m.Title,
		Meeting:    m,
		CurrentID:  m.ID,
		Error:      message,
		Speaker:    firstSpeaker(m),
		ScriptDone: !more,
	}
}

func (h *handler) render(w http.ResponseWriter, status int, name string, v view) {
	if v.Meetings == nil {
		v.Meetings = h.store.List()
	}
	if v.Title == "" {
		v.Title = "f - clone"
	}
	var buf bytes.Buffer
	if err := h.tmpl.ExecuteTemplate(&buf, name, v); err != nil {
		slog.Error("render", "template", name, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, &buf)
}

func (h *handler) missing(w http.ResponseWriter) {
	h.render(w, http.StatusNotFound, "missing", view{Title: "Not found"})
}

func (h *handler) fail(w http.ResponseWriter, err error) {
	slog.Error("request", "err", err)
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}

func mustTemplates() *template.Template {
	t, err := template.New("").Funcs(template.FuncMap{
		"dur":    formatDuration,
		"when":   formatWhen,
		"offset": formatOffset,
	}).ParseFS(web.FS, "templates/*.tmpl")
	if err != nil {
		panic(err)
	}
	return t
}

func readForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	return r.ParseForm()
}

func validID(id string) bool {
	return idPattern.MatchString(id)
}

func parseParticipants(raw string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		name := clip(strings.TrimSpace(part), 40)
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

func firstSpeaker(m meeting.Meeting) string {
	if len(m.Participants) == 0 {
		return "Speaker"
	}
	return m.Participants[0]
}

func clip(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func formatDuration(sec float64) string {
	if sec < 0 || math.IsNaN(sec) {
		sec = 0
	}
	total := int(math.Round(sec))
	min := total / 60
	s := total % 60
	if min == 0 {
		return itoa(s) + "s"
	}
	return itoa(min) + "m " + pad(s) + "s"
}

func formatOffset(sec float64) string {
	if sec < 0 || math.IsNaN(sec) {
		sec = 0
	}
	total := int(sec)
	return itoa(total/60) + ":" + pad(total%60)
}

func formatWhen(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	local := t.Local()
	stamp := local.Format("Jan 2, 15:04")
	if label := labelForZone(localZone()); label != "" {
		return stamp + " " + label
	}
	return stamp
}

func localZone() string {
	if tz := os.Getenv("TZ"); tz != "" && tz != "Local" {
		return tz
	}
	target, err := os.Readlink("/etc/localtime")
	if err != nil {
		return ""
	}
	const marker = "zoneinfo/"
	if i := strings.LastIndex(target, marker); i >= 0 {
		return target[i+len(marker):]
	}
	return ""
}

func labelForZone(name string) string {
	if name == "" || name == "Local" {
		return ""
	}
	if name == "America/Buenos_Aires" || strings.HasPrefix(name, "America/Argentina/") {
		return "BS AS/Argentina"
	}
	parts := strings.Split(name, "/")
	if len(parts) >= 2 {
		city := strings.ReplaceAll(parts[len(parts)-1], "_", " ")
		region := strings.ReplaceAll(parts[len(parts)-2], "_", " ")
		return city + "/" + region
	}
	return strings.ReplaceAll(name, "_", " ")
}

func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
