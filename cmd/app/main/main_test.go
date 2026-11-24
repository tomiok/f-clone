package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"fireflies-clone/cmd/app"
	"fireflies-clone/internal/httpx"
	"fireflies-clone/internal/store"
)

func TestNotebookAndRecordFlow(t *testing.T) {
	h := newTestServer(t)

	home := get(t, h, "/")
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "Q3 pricing decision") {
		t.Fatalf("home = %d %s", home.Code, home.Body.String())
	}
	if !strings.Contains(home.Body.String(), "Connect Google Calendar") || !strings.Contains(home.Body.String(), "Connect Outlook Calendar") {
		t.Fatal("calendar buttons missing")
	}

	created := post(t, h, "/meetings", url.Values{
		"title":        {"Pricing sync"},
		"participants": {"Maya, Jordan"},
	})
	if created.Code != http.StatusSeeOther {
		t.Fatalf("create = %d %s", created.Code, created.Body.String())
	}
	loc := created.Header().Get("Location")
	id := strings.TrimSuffix(strings.TrimPrefix(loc, "/meetings/"), "/live")
	if id == "" || id == loc {
		t.Fatalf("location = %q", loc)
	}

	live := get(t, h, loc)
	if live.Code != http.StatusOK || !strings.Contains(live.Body.String(), "Pricing sync") || !strings.Contains(live.Body.String(), "id=\"resume\"") {
		t.Fatalf("live = %d %s", live.Code, live.Body.String())
	}

	added := post(t, h, "/meetings/"+id+"/lines", url.Values{
		"speaker": {"Maya"},
		"text":    {"I'll send the recap tomorrow."},
	})
	if added.Code != http.StatusSeeOther {
		t.Fatalf("line = %d %s", added.Code, added.Body.String())
	}

	stopped := post(t, h, "/meetings/"+id+"/stop", nil)
	if stopped.Code != http.StatusOK || !strings.Contains(stopped.Body.String(), "Transcribing") {
		t.Fatalf("stop = %d %s", stopped.Code, stopped.Body.String())
	}

	page := get(t, h, "/meetings/"+id)
	body := page.Body.String()
	if page.Code != http.StatusOK || !strings.Contains(body, "send the recap tomorrow.") {
		t.Fatalf("notepad = %d %s", page.Code, body)
	}
	if !strings.Contains(body, "Maya") || !strings.Contains(body, "Overview") {
		t.Fatalf("notepad missing notes: %s", body)
	}

	actionID := "a1"
	toggled := post(t, h, "/meetings/"+id+"/actions/"+actionID, nil)
	if toggled.Code != http.StatusSeeOther {
		t.Fatalf("toggle = %d", toggled.Code)
	}
	checked := get(t, h, "/meetings/"+id)
	if !strings.Contains(checked.Body.String(), "is-done") {
		t.Fatalf("action not checked: %s", checked.Body.String())
	}

	post(t, h, "/meetings/"+id+"/stop", nil)
	again := get(t, h, "/meetings/"+id)
	if !strings.Contains(again.Body.String(), "is-done") {
		t.Fatal("second stop cleared the checkmark")
	}

	missing := get(t, h, "/meetings/does-not-exist")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing = %d", missing.Code)
	}
}

func TestScriptLineAndDelete(t *testing.T) {
	h := newTestServer(t)
	created := post(t, h, "/meetings", url.Values{"title": {"Sync"}})
	loc := created.Header().Get("Location")
	id := strings.TrimSuffix(strings.TrimPrefix(loc, "/meetings/"), "/live")

	added := post(t, h, "/meetings/"+id+"/lines", url.Values{"source": {"script"}})
	if added.Code != http.StatusSeeOther {
		t.Fatalf("script = %d", added.Code)
	}
	live := get(t, h, "/meetings/"+id+"/live")
	if !strings.Contains(live.Body.String(), "decide the launch price") {
		t.Fatalf("script line missing: %s", live.Body.String())
	}

	removed := post(t, h, "/meetings/"+id+"/delete", nil)
	if removed.Code != http.StatusSeeOther {
		t.Fatalf("delete = %d", removed.Code)
	}
	if got := get(t, h, "/meetings/"+id); got.Code != http.StatusNotFound {
		t.Fatalf("after delete = %d", got.Code)
	}
}

func TestResumeContinuesFromStoppedTime(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "meetings.json"))
	if err != nil {
		t.Fatal(err)
	}
	h := handlerServer(httpx.New(st))
	created := post(t, h, "/meetings", url.Values{"title": {"Hold"}})
	id := strings.TrimSuffix(strings.TrimPrefix(created.Header().Get("Location"), "/meetings/"), "/live")
	post(t, h, "/meetings/"+id+"/lines", url.Values{"source": {"script"}})
	m, ok := st.Get(id)
	if !ok || len(m.Sentences) != 1 {
		t.Fatal("expected one line")
	}
	held := post(t, h, "/meetings/"+id+"/lines", url.Values{"source": {"script"}, "held": {"1"}})
	if held.Code != http.StatusSeeOther {
		t.Fatalf("held = %d", held.Code)
	}
	m, ok = st.Get(id)
	if !ok || len(m.Sentences) != 2 {
		t.Fatal("expected two lines")
	}
	if m.Sentences[1].StartSec != m.Sentences[0].EndSec {
		t.Fatalf("resume jumped the clock: %#v", m.Sentences)
	}
}

func TestEmptyStop(t *testing.T) {
	h := newTestServer(t)
	created := post(t, h, "/meetings", url.Values{"title": {"Silent"}})
	id := strings.TrimSuffix(strings.TrimPrefix(created.Header().Get("Location"), "/meetings/"), "/live")
	post(t, h, "/meetings/"+id+"/stop", nil)
	page := get(t, h, "/meetings/"+id)
	if !strings.Contains(page.Body.String(), "Nothing was captured.") {
		t.Fatalf("empty notepad = %s", page.Body.String())
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "meetings.json"))
	if err != nil {
		t.Fatal(err)
	}
	return handlerServer(httpx.New(st))
}

func handlerServer(h httpx.Handler) http.Handler {
	r := chi.NewRouter()
	routes(r, app.Deps{Handler: h})
	return r
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func post(t *testing.T, h http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(http.MethodPost, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}
