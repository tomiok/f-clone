// Package summary builds the notepad from transcript sentences.
//
// Build is pure: no files, no network. An LLM client can return the same
// Result later without the handlers changing.
package summary

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"fireflies-clone/internal/meeting"
)

const emptyOverview = "Nothing was captured."

// Result is the generated side of a meeting.
type Result struct {
	Overview string
	Notes    []string
	Keywords []string
	Actions  []meeting.Action
}

var (
	cue         = regexp.MustCompile(`(?i)\b(decided|agreed|decision|deadline|risk|ship|shipping)\b`)
	commit      = regexp.MustCompile(`(?i)\b(i'll|i will|we'll|we will|let's|lets|need to|follow[\s-]?up)\b`)
	namedWill   = regexp.MustCompile(`\b([A-Z][a-z]+)\s+will\b`)
	firstPerson = regexp.MustCompile(`(?i)^(i'll|i will)\b`)
	sentenceRe  = regexp.MustCompile(`[^.!?]+[.!?]?`)
)

// stopwords are dropped before scoring and keywords.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {}, "if": {},
	"in": {}, "on": {}, "for": {}, "to": {}, "of": {}, "at": {}, "by": {},
	"from": {}, "with": {}, "as": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "it": {}, "this": {}, "that": {},
	"these": {}, "those": {}, "we": {}, "you": {}, "they": {}, "i": {}, "he": {},
	"she": {}, "them": {}, "our": {}, "your": {}, "their": {}, "not": {}, "no": {},
	"so": {}, "do": {}, "does": {}, "did": {}, "have": {}, "has": {}, "had": {},
	"will": {}, "would": {}, "can": {}, "could": {}, "should": {}, "about": {},
	"into": {}, "over": {}, "after": {}, "before": {}, "than": {}, "then": {},
	"also": {}, "just": {}, "very": {}, "what": {}, "when": {}, "which": {},
}

// Build returns an overview, notes, keywords, and action items.
// Overview sentences are quoted from the transcript, in the order they were spoken.
func Build(sentences []meeting.Sentence) Result {
	units := split(sentences)
	if len(units) == 0 {
		return Result{Overview: emptyOverview}
	}

	tf := map[string]int{}
	for _, u := range units {
		seen := map[string]struct{}{}
		for _, tok := range tokens(u.text) {
			if _, ok := seen[tok]; ok {
				continue
			}
			seen[tok] = struct{}{}
			tf[tok]++
		}
	}

	type scored struct {
		index int
		text  string
		score float64
	}
	ranked := make([]scored, len(units))
	for i, u := range units {
		ranked[i] = scored{index: i, text: u.text, score: score(u.text, tf, i, len(units))}
	}
	byScore := append([]scored(nil), ranked...)
	sort.SliceStable(byScore, func(i, j int) bool {
		if byScore[i].score == byScore[j].score {
			return byScore[i].index < byScore[j].index
		}
		return byScore[i].score > byScore[j].score
	})

	overviewN := 2
	if len(units) < overviewN {
		overviewN = len(units)
	}
	chosen := map[int]struct{}{}
	overviewIdx := make([]int, 0, overviewN)
	for _, s := range byScore {
		if len(overviewIdx) == overviewN {
			break
		}
		overviewIdx = append(overviewIdx, s.index)
		chosen[s.index] = struct{}{}
	}
	sort.Ints(overviewIdx)
	parts := make([]string, len(overviewIdx))
	for i, idx := range overviewIdx {
		parts[i] = units[idx].text
	}

	notes := make([]string, 0, 4)
	for _, s := range byScore {
		if _, ok := chosen[s.index]; ok {
			continue
		}
		notes = append(notes, s.text)
		if len(notes) == 4 {
			break
		}
	}
	if len(notes) == 0 {
		notes = append(notes, parts...)
	}
	sortNotes := notesInSpokenOrder(units, notes)

	return Result{
		Overview: strings.Join(parts, " "),
		Notes:    sortNotes,
		Keywords: keywords(units),
		Actions:  actions(sentences),
	}
}

type unit struct {
	text string
}

func split(sentences []meeting.Sentence) []unit {
	var out []unit
	for _, s := range sentences {
		for _, part := range sentenceRe.FindAllString(s.Text, -1) {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, unit{text: part})
			}
		}
	}
	return out
}

func score(text string, tf map[string]int, index, n int) float64 {
	toks := tokens(text)
	if len(toks) == 0 {
		return 0
	}
	seen := map[string]struct{}{}
	sum := 0.0
	for _, tok := range toks {
		if _, ok := seen[tok]; ok {
			continue
		}
		seen[tok] = struct{}{}
		sum += float64(tf[tok])
	}
	s := sum / math.Sqrt(float64(len(toks)))
	if n > 1 {
		s *= 1 + 0.15*(1-float64(index)/float64(n-1))
	}
	if cue.MatchString(text) {
		s *= 1.25
	}
	return s
}

func notesInSpokenOrder(units []unit, notes []string) []string {
	pos := map[string]int{}
	for i, u := range units {
		if _, ok := pos[u.text]; !ok {
			pos[u.text] = i
		}
	}
	sort.SliceStable(notes, func(i, j int) bool {
		return pos[notes[i]] < pos[notes[j]]
	})
	return notes
}

func keywords(units []unit) []string {
	type kw struct {
		word  string
		count int
		first int
	}
	seen := map[string]*kw{}
	var order []string
	n := 0
	for _, u := range units {
		used := map[string]struct{}{}
		for _, tok := range tokens(u.text) {
			if _, ok := used[tok]; ok {
				continue
			}
			used[tok] = struct{}{}
			if item, ok := seen[tok]; ok {
				item.count++
				continue
			}
			seen[tok] = &kw{word: tok, count: 1, first: n}
			order = append(order, tok)
			n++
		}
	}
	items := make([]kw, 0, len(order))
	for _, word := range order {
		items = append(items, *seen[word])
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].first < items[j].first
		}
		return items[i].count > items[j].count
	})
	limit := 6
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]string, limit)
	for i := 0; i < limit; i++ {
		out[i] = items[i].word
	}
	return out
}

func actions(sentences []meeting.Sentence) []meeting.Action {
	var out []meeting.Action
	seen := map[string]struct{}{}
	for _, s := range sentences {
		text := strings.TrimSpace(s.Text)
		if text == "" || strings.HasSuffix(text, "?") {
			continue
		}
		if !commit.MatchString(text) && !namedWill.MatchString(text) {
			continue
		}
		key := strings.ToLower(text)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		owner := ""
		if m := namedWill.FindStringSubmatch(text); m != nil {
			owner = m[1]
		} else if firstPerson.MatchString(text) {
			owner = s.Speaker
		}
		out = append(out, meeting.Action{
			ID:    "a" + itoa(len(out)+1),
			Text:  text,
			Owner: owner,
		})
	}
	return out
}

func tokens(text string) []string {
	var b strings.Builder
	var out []string
	flush := func() {
		if b.Len() == 0 {
			return
		}
		word := b.String()
		b.Reset()
		if len(word) < 3 {
			return
		}
		if _, skip := stopwords[word]; skip {
			return
		}
		out = append(out, word)
	}
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [8]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}
