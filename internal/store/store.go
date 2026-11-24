// Package store persists meetings as one JSON file.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fireflies-clone/internal/conversation"
	"fireflies-clone/internal/meeting"
	"fireflies-clone/internal/summary"
)

// ErrNotFound is returned when the id is not in the file.
var ErrNotFound = errors.New("meeting not found")

// ErrNotActive is returned when a line is added after the meeting has stopped.
var ErrNotActive = errors.New("meeting is not active")

// Store is safe for concurrent handlers in one process.
type Store struct {
	mu       sync.Mutex
	path     string
	meetings []meeting.Meeting
}

// Open reads path, or writes the seed meeting when the file does not exist.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.meetings = []meeting.Meeting{sample()}
		if err := s.flush(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytesTrim(b)) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, &s.meetings); err != nil {
		return nil, err
	}
	return s, nil
}

func bytesTrim(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\t' || b[i] == '\r') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\t' || b[j-1] == '\r') {
		j--
	}
	return b[i:j]
}

// List returns meetings newest first.
func (s *Store) List() []meeting.Meeting {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]meeting.Meeting(nil), s.meetings...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Get returns a copy of the meeting.
func (s *Store) Get(id string) (meeting.Meeting, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.find(id)
	if !ok {
		return meeting.Meeting{}, false
	}
	return clone(m), true
}

// Create starts an active meeting.
func (s *Store) Create(title string, participants []string, now time.Time) (meeting.Meeting, error) {
	id, err := newID()
	if err != nil {
		return meeting.Meeting{}, err
	}
	m := meeting.Meeting{
		ID:           id,
		Title:        title,
		CreatedAt:    now.UTC(),
		Status:       meeting.StatusActive,
		Participants: append([]string(nil), participants...),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meetings = append(s.meetings, m)
	if err := s.flush(); err != nil {
		s.meetings = s.meetings[:len(s.meetings)-1]
		return meeting.Meeting{}, err
	}
	return clone(&s.meetings[len(s.meetings)-1]), nil
}

// Append adds a sentence while the meeting is active.
func (s *Store) Append(id string, sentence meeting.Sentence) (meeting.Meeting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.find(id)
	if !ok {
		return meeting.Meeting{}, ErrNotFound
	}
	if m.Status != meeting.StatusActive {
		return meeting.Meeting{}, ErrNotActive
	}
	m.Sentences = append(m.Sentences, sentence)
	if sentence.Speaker != "" && !contains(m.Participants, sentence.Speaker) {
		m.Participants = append(m.Participants, sentence.Speaker)
	}
	if sentence.EndSec > m.DurationSec {
		m.DurationSec = sentence.EndSec
	}
	if err := s.flush(); err != nil {
		return meeting.Meeting{}, err
	}
	return clone(m), nil
}

// BeginFinish moves an active meeting to processing and remembers the duration.
// already is true when the meeting was already completed and must not be summarized again.
func (s *Store) BeginFinish(id string, now time.Time) (meeting.Meeting, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.find(id)
	if !ok {
		return meeting.Meeting{}, false, ErrNotFound
	}
	if m.Status == meeting.StatusCompleted {
		return clone(m), true, nil
	}
	if m.Status == meeting.StatusActive {
		m.Status = meeting.StatusProcessing
		elapsed := now.Sub(m.CreatedAt).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		if elapsed > m.DurationSec {
			m.DurationSec = elapsed
		}
		if err := s.flush(); err != nil {
			return meeting.Meeting{}, false, err
		}
	}
	return clone(m), false, nil
}

// Finish stores the summary. It is a no-op when the meeting is already completed,
// so a second stop does not rebuild action items and clear their checkmarks.
func (s *Store) Finish(id string, result summary.Result) (meeting.Meeting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.find(id)
	if !ok {
		return meeting.Meeting{}, ErrNotFound
	}
	if m.Status == meeting.StatusCompleted {
		return clone(m), nil
	}
	m.Status = meeting.StatusCompleted
	m.Overview = result.Overview
	m.Notes = result.Notes
	m.Keywords = result.Keywords
	m.Actions = result.Actions
	if err := s.flush(); err != nil {
		return meeting.Meeting{}, err
	}
	return clone(m), nil
}

// ToggleAction flips Done. Missing meetings and missing actions return ErrNotFound.
func (s *Store) ToggleAction(id, actionID string) (meeting.Meeting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.find(id)
	if !ok {
		return meeting.Meeting{}, ErrNotFound
	}
	found := false
	for i := range m.Actions {
		if m.Actions[i].ID == actionID {
			m.Actions[i].Done = !m.Actions[i].Done
			found = true
			break
		}
	}
	if !found {
		return meeting.Meeting{}, ErrNotFound
	}
	if err := s.flush(); err != nil {
		return meeting.Meeting{}, err
	}
	return clone(m), nil
}

// Delete removes a meeting.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.meetings {
		if s.meetings[i].ID != id {
			continue
		}
		s.meetings = append(s.meetings[:i], s.meetings[i+1:]...)
		return s.flush()
	}
	return ErrNotFound
}

func (s *Store) find(id string) (*meeting.Meeting, bool) {
	for i := range s.meetings {
		if s.meetings[i].ID == id {
			return &s.meetings[i], true
		}
	}
	return nil, false
}

func (s *Store) flush() error {
	b, err := json.MarshalIndent(s.meetings, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func clone(m *meeting.Meeting) meeting.Meeting {
	out := *m
	out.Participants = append([]string(nil), m.Participants...)
	out.Sentences = append([]meeting.Sentence(nil), m.Sentences...)
	out.Keywords = append([]string(nil), m.Keywords...)
	out.Notes = append([]string(nil), m.Notes...)
	out.Actions = append([]meeting.Action(nil), m.Actions...)
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func newID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func sample() meeting.Meeting {
	lines := conversation.MockConversation{}.Lines(nil)
	sentences := make([]meeting.Sentence, len(lines))
	var speakers []string
	seen := map[string]struct{}{}
	for i, ln := range lines {
		start := float64(i * 8)
		sentences[i] = meeting.Sentence{
			Speaker:  ln.Speaker,
			StartSec: start,
			EndSec:   start + 6,
			Text:     ln.Text,
		}
		if _, ok := seen[ln.Speaker]; !ok {
			seen[ln.Speaker] = struct{}{}
			speakers = append(speakers, ln.Speaker)
		}
	}
	res := summary.Build(sentences)
	end := sentences[len(sentences)-1].EndSec
	return meeting.Meeting{
		ID:           "seed-pricing",
		Title:        "Q3 pricing decision",
		CreatedAt:    time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC),
		DurationSec:  end,
		Status:       meeting.StatusCompleted,
		Participants: speakers,
		Sentences:    sentences,
		Keywords:     res.Keywords,
		Overview:     res.Overview,
		Notes:        res.Notes,
		Actions:      res.Actions,
	}
}
