// Package meeting holds the document the rest of the app passes around.
package meeting

import "time"

// Status is where a meeting is in the capture loop.
type Status string

const (
	StatusActive     Status = "active"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
)

// Sentence is one spoken line. Times are seconds from the start of the meeting.
type Sentence struct {
	Speaker  string  `json:"speaker"`
	StartSec float64 `json:"startSec"`
	EndSec   float64 `json:"endSec"`
	Text     string  `json:"text"`
}

// Action is a follow-up extracted from the transcript.
type Action struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Owner string `json:"owner,omitempty"`
	Done  bool   `json:"done"`
}

// IsActive reports whether the meeting is still being captured.
func (m Meeting) IsActive() bool { return m.Status == StatusActive }

// IsProcessing reports whether notes are being built.
func (m Meeting) IsProcessing() bool { return m.Status == StatusProcessing }

// Meeting is one recorded conversation and the notes produced from it.
type Meeting struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	CreatedAt    time.Time  `json:"createdAt"`
	DurationSec  float64    `json:"durationSec"`
	Status       Status     `json:"status"`
	Participants []string   `json:"participants"`
	Sentences    []Sentence `json:"sentences"`
	Keywords     []string   `json:"keywords"`
	Overview     string     `json:"overview"`
	Notes        []string   `json:"notes"`
	Actions      []Action   `json:"actions"`
}
