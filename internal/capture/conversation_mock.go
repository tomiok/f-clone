package capture

import "strings"

// MockConversation is the fixed pricing dialogue used until a real
// Conversation exists. Reimplement Conversation in another file to
// replace it. Do not add real capture here.
type MockConversation struct{}

func (MockConversation) Lines(participants []string) []Line {
	lines := []Line{
		{"Maya", "We need to decide the launch price before Friday."},
		{"Jordan", "The risk is shipping at forty nine if support cannot handle the volume."},
		{"Priya", "I think we should look at the competitor page again."},
		{"Maya", "We agreed to ship the annual plan at ninety nine dollars."},
		{"Maya", "Maya will send the pricing sheet to finance tomorrow."},
		{"Jordan", "I'll book the customer webinar for next Thursday."},
		{"Priya", "Do we have a deadline for the help center update?"},
		{"Jordan", "Let's follow up with support on staffing before we announce."},
	}
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
			ln.Text = strings.ReplaceAll(ln.Text, ln.Speaker, name)
			ln.Speaker = name
		}
		out[i] = ln
	}
	return out
}
