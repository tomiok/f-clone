package summary

import (
	"strings"
	"testing"

	"fireflies-clone/internal/capture"
	"fireflies-clone/internal/meeting"
)

func fixture() []meeting.Sentence {
	lines := capture.MockConversation{}.Lines(nil)
	out := make([]meeting.Sentence, len(lines))
	for i, ln := range lines {
		out[i] = meeting.Sentence{Speaker: ln.Speaker, Text: ln.Text, StartSec: float64(i), EndSec: float64(i + 1)}
	}
	return out
}

func TestBuildFixture(t *testing.T) {
	got := Build(fixture())
	if !strings.Contains(got.Overview, "agreed to ship the annual plan") {
		t.Fatalf("overview = %q", got.Overview)
	}
	if len(got.Notes) == 0 {
		t.Fatal("expected notes")
	}
	if len(got.Keywords) == 0 || len(got.Keywords) > 6 {
		t.Fatalf("keywords = %#v", got.Keywords)
	}

	var texts []string
	owners := map[string]string{}
	for _, a := range got.Actions {
		if strings.HasSuffix(a.Text, "?") {
			t.Fatalf("question stored as an action: %q", a.Text)
		}
		if strings.Contains(strings.ToLower(a.Text), "should look") {
			t.Fatalf("vague line stored as an action: %q", a.Text)
		}
		texts = append(texts, a.Text)
		owners[a.Text] = a.Owner
		if a.ID == "" {
			t.Fatal("action missing id")
		}
	}
	joined := strings.Join(texts, "\n")
	if !strings.Contains(joined, "Maya will send the pricing sheet") {
		t.Fatalf("missing named action in %#v", texts)
	}
	if owners["Maya will send the pricing sheet to finance tomorrow."] != "Maya" {
		t.Fatalf("maya owner = %q", owners["Maya will send the pricing sheet to finance tomorrow."])
	}
	if owners["I'll book the customer webinar for next Thursday."] != "Jordan" {
		t.Fatalf("jordan owner = %q", owners["I'll book the customer webinar for next Thursday."])
	}
	if strings.Contains(joined, "Do we have a deadline") {
		t.Fatal("question was included")
	}
}

func TestBuildEmpty(t *testing.T) {
	got := Build(nil)
	if got.Overview != "Nothing was captured." {
		t.Fatalf("overview = %q", got.Overview)
	}
	if len(got.Notes) != 0 || len(got.Actions) != 0 || len(got.Keywords) != 0 {
		t.Fatalf("empty result leaked data: %#v", got)
	}
}

func TestBuildBlankLines(t *testing.T) {
	got := Build([]meeting.Sentence{{Speaker: "Maya", Text: "   "}})
	if got.Overview != "Nothing was captured." {
		t.Fatalf("overview = %q", got.Overview)
	}
}
