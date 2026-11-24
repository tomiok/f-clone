package store

import (
	"path/filepath"
	"testing"
	"time"

	"fireflies-clone/internal/meeting"
	"fireflies-clone/internal/summary"
)

func TestOpenSeedsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meetings.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 1 || list[0].ID != "seed-pricing" {
		t.Fatalf("seed = %#v", list)
	}
	if list[0].Status != meeting.StatusCompleted || list[0].Overview == "" {
		t.Fatalf("seed not completed: %#v", list[0])
	}

	again, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.List()) != 1 {
		t.Fatal("reopen rewrote the seed")
	}
}

func TestCreateAppendFinishToggleDelete(t *testing.T) {
	s := openEmpty(t)
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	m, err := s.Create("Launch", []string{"Maya"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get("missing"); ok {
		t.Fatal("missing id was found")
	}

	m, err = s.Append(m.ID, meeting.Sentence{
		Speaker: "Maya", StartSec: 1, EndSec: 3, Text: "I'll send the recap tomorrow.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Sentences) != 1 {
		t.Fatalf("sentences = %d", len(m.Sentences))
	}

	got, done, err := s.BeginFinish(m.ID, now.Add(30*time.Second))
	if err != nil || done {
		t.Fatalf("begin done=%v err=%v", done, err)
	}
	if got.Status != meeting.StatusProcessing {
		t.Fatalf("status = %s", got.Status)
	}
	res := summary.Build(got.Sentences)
	got, err = s.Finish(m.ID, res)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != meeting.StatusCompleted || len(got.Actions) != 1 {
		t.Fatalf("finish = status %s actions %d", got.Status, len(got.Actions))
	}
	if got.Actions[0].Owner != "Maya" {
		t.Fatalf("owner = %q", got.Actions[0].Owner)
	}

	got, err = s.ToggleAction(m.ID, got.Actions[0].ID)
	if err != nil || !got.Actions[0].Done {
		t.Fatalf("toggle = done %v err %v", got.Actions[0].Done, err)
	}

	_, err = s.Finish(m.ID, summary.Result{})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get(m.ID)
	if !ok || !got.Actions[0].Done || got.Overview == "" {
		t.Fatal("second finish wiped the meeting")
	}

	if _, err := s.Append(m.ID, meeting.Sentence{Text: "too late"}); err != ErrNotActive {
		t.Fatalf("append after stop = %v", err)
	}
	if err := s.Delete(m.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(m.ID); ok {
		t.Fatal("deleted meeting still present")
	}
	if err := s.Delete(m.ID); err != ErrNotFound {
		t.Fatalf("second delete = %v", err)
	}
}

func openEmpty(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "meetings.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("seed-pricing"); err != nil {
		t.Fatal(err)
	}
	return s
}
