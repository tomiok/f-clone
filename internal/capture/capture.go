// Package capture turns a live meeting into transcript lines.
//
// Recorder is the seam in front of recording. Conversation is where the
// lines come from. The shipped conversation is MockConversation. A real
// capture reimplements Conversation, and this walker stays the same.
package capture

import "fireflies-clone/internal/meeting"

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

// Conversation is the dialogue a recorder walks. Participants are the
// names to use, in first-seen order. A real meeting reimplements this
// instead of editing the mock.
type Conversation interface {
	Lines(participants []string) []Line
}

// script walks a conversation from the first line that is not already stored.
type script struct {
	lines []Line
	i     int
}

// NewRecorder positions a recorder at the next uncaptured line.
// Lines already stored, including typed ones, do not consume the conversation.
func NewRecorder(conv Conversation, participants []string, sentences []meeting.Sentence) Recorder {
	lines := conv.Lines(participants)
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
