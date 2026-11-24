// Package capture turns a live meeting into transcript lines.
//
// Recorder is the seam in front of recording. The shipped implementation
// walks a fixed conversation. A microphone or a meeting bot would be another
// type with the same Next method, and the HTTP handlers would not change.
package capture

import (
	"strings"

	"fireflies-clone/internal/meeting"
)

// Line is one turn of speech before it has a timestamp.
type Line struct {
	Speaker string
	Text    string
}

// Recorder yields the next line of a meeting. The bool is false when there
// is nothing further to capture.
type Recorder interface {
	Next() (Line, bool)
}

// ProductMeeting is the conversation behind the seeded notepad and the
// default live meeting.
func ProductMeeting() []Line {
	return []Line{
		{"Maya", "We need to decide the launch price before Friday."},
		{"Jordan", "The risk is shipping at forty nine if support cannot handle the volume."},
		{"Priya", "I think we should look at the competitor page again."},
		{"Maya", "We agreed to ship the annual plan at ninety nine dollars."},
		{"Maya", "Maya will send the pricing sheet to finance tomorrow."},
		{"Jordan", "I'll book the customer webinar for next Thursday."},
		{"Priya", "Do we have a deadline for the help center update?"},
		{"Jordan", "Let's follow up with support on staffing before we announce."},
	}
}

// LinesFor rewrites the sample conversation onto the meeting's participants,
// in first-seen order. Unknown extra speakers stay as written.
func LinesFor(participants []string) []Line {
	lines := ProductMeeting()
	if len(participants) == 0 {
		return lines
	}
	order := []string{"Maya", "Jordan", "Priya"}
	repl := map[string]string{}
	for i, name := range order {
		if i >= len(participants) {
			break
		}
		repl[name] = participants[i]
	}
	out := make([]Line, len(lines))
	for i, ln := range lines {
		if name, ok := repl[ln.Speaker]; ok && name != ln.Speaker {
			ln.Text = replaceName(ln.Text, ln.Speaker, name)
			ln.Speaker = name
		}
		out[i] = ln
	}
	return out
}

func replaceName(text, from, to string) string {
	return strings.ReplaceAll(text, from, to)
}

// script walks LinesFor from the first line that is not already stored.
type script struct {
	lines []Line
	i     int
}

// NewRecorder positions a recorder at the next uncaptured script line.
// Typed lines in the middle do not consume the script.
func NewRecorder(participants []string, sentences []meeting.Sentence) Recorder {
	lines := LinesFor(participants)
	return &script{lines: lines, i: consumed(lines, sentences)}
}

func (s *script) Next() (Line, bool) {
	if s.i >= len(s.lines) {
		return Line{}, false
	}
	ln := s.lines[s.i]
	s.i++
	return ln, true
}

func consumed(lines []Line, sentences []meeting.Sentence) int {
	i := 0
	for _, s := range sentences {
		if i >= len(lines) {
			break
		}
		if s.Speaker == lines[i].Speaker && s.Text == lines[i].Text {
			i++
		}
	}
	return i
}
