package web

import (
	"bytes"
	"html/template"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"fireflies-clone/internal/meeting"
)

// View is the data passed to every template.
type View struct {
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

// Pages renders the shared notebook shell.
type Pages struct {
	tmpl     *template.Template
	meetings func() []meeting.Meeting
}

// NewPages parses the templates once. meetings fills the left rail.
func NewPages(meetings func() []meeting.Meeting) *Pages {
	t, err := template.New("").Funcs(template.FuncMap{
		"dur":    formatDuration,
		"when":   formatWhen,
		"offset": formatOffset,
	}).ParseFS(FS, "templates/*.tmpl")
	if err != nil {
		panic(err)
	}
	return &Pages{tmpl: t, meetings: meetings}
}

// Render writes one template. A nil meeting list is filled from the store.
func (p *Pages) Render(w http.ResponseWriter, status int, name string, v View) {
	if v.Meetings == nil && p.meetings != nil {
		v.Meetings = p.meetings()
	}
	if v.Title == "" {
		v.Title = "f - clone"
	}
	var buf bytes.Buffer
	if err := p.tmpl.ExecuteTemplate(&buf, name, v); err != nil {
		slog.Error("render", "template", name, "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.Copy(w, &buf)
}

// Missing writes the not-found page.
func (p *Pages) Missing(w http.ResponseWriter) {
	p.Render(w, http.StatusNotFound, "missing", View{Title: "Not found"})
}

// Fail logs err and writes a plain 500. The page body does not include the error.
func (p *Pages) Fail(w http.ResponseWriter, err error) {
	slog.Error("request", "err", err)
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// ValidID reports whether id can be used in a route.
func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// ReadForm parses a small urlencoded body.
func ReadForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	return r.ParseForm()
}

// Clip limits s to max runes.
func Clip(s string, max int) string {
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
