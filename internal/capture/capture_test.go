package capture

import (
	"strings"
	"testing"

	"fireflies-clone/internal/meeting"
)

func TestRecorderWalksTheScriptOnce(t *testing.T) {
	lines := MockConversation{}.Lines(nil)
	rec := NewRecorder(MockConversation{}, nil, nil)
	var got []Line
	for {
		ln, ok := rec.Next()
		if !ok {
			break
		}
		got = append(got, ln)
	}
	if len(got) != len(lines) {
		t.Fatalf("got %d lines, want %d", len(got), len(lines))
	}
	if _, ok := rec.Next(); ok {
		t.Fatal("Next after the end returned a line")
	}
}

func TestRecorderSkipsLinesAlreadyStored(t *testing.T) {
	first := MockConversation{}.Lines(nil)[0]
	rec := NewRecorder(MockConversation{}, nil, []meeting.Sentence{{
		Speaker: first.Speaker,
		Text:    first.Text,
	}})
	ln, ok := rec.Next()
	if !ok {
		t.Fatal("expected a second line")
	}
	if ln.Text == first.Text {
		t.Fatalf("repeated the stored line: %q", ln.Text)
	}
}

func TestLinesForRenamesSpeakers(t *testing.T) {
	lines := MockConversation{}.Lines([]string{"Alice", "Ben", "Chen"})
	if lines[0].Speaker != "Alice" {
		t.Fatalf("speaker = %q", lines[0].Speaker)
	}
	found := false
	for _, ln := range lines {
		if ln.Speaker == "Alice" && strings.Contains(ln.Text, "Alice will send") {
			found = true
		}
		if strings.Contains(ln.Text, "Maya") {
			t.Fatalf("old name left in %q", ln.Text)
		}
	}
	if !found {
		t.Fatal("named commitment was not rewritten")
	}
}
