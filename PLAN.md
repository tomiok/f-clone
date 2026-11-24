# Plan: Fireflies challenge, in Go

Email constraint, which overrides the Notion line "TypeScript preferred": build this in Go. The working app should feel like the product (record, live notes, processing, notepad). Do not cover the UI in disclaimers. The short write-up still explains what was built and why, because that is a deliverable and the follow-ups will go there.

Reviewer questions and the answers we want ready are in [FOLLOWUPS.md](FOLLOWUPS.md). Read that before the submission call.

## Scope: the web app

Fireflies' real install is the website. The desktop app, Chrome extension, and phone apps are optional clients. This challenge clones the website only.

In the browser, and nothing else:

- Meetings list (their Notebook)
- A live meeting page (their Live Notes, simplified)
- A notepad: summary on the left, transcript on the right

Not in this build: a desktop app, a browser extension, a mobile app, Google or Microsoft login, calendar auto-join, or a bot that enters Zoom, Meet, or Teams. There is no install step. `go run` serves the site, and the first screen is the meeting list.

Capture still goes through the `Recorder` interface. On the web app that means the live page, not a separate program.

## What has to ship

| Requirement | Done when |
| --- | --- |
| Meeting recording | Start and stop a meeting from the UI. Capture is a placeholder behind a small interface. No microphone, no API key, no Zoom bot. |
| Transcription | The finished meeting has sentences: speaker, start, end, text. |
| Summary and action items | Overview, notes, keywords, and action items with an owner when the line supports one. |
| Interface | Notebook list, a live meeting screen, and a notepad (summary on the left, transcript on the right). |
| Go | One module, stdlib HTTP and templates, tests on the pure packages. |
| Write-up | README for setup and run. APPROACH.md for what and why, including time spent. |
| Later, not now | GitHub repo and a URL for the running app. Do not push until asked. |

Time box: under 10 hours. Cut live "ask", speaker rename, and playback before cutting record, transcript, summary, or action items.

## Product shape

Match the Fireflies loop, not the rest of Fred (email, Slack, voice agents, CRM).

```text
Record
  → Active (timer, lines appearing)
  → Processing (short, visible)
  → Completed notepad
```

Screens:

- **Meetings.** List with title, when, duration, one-line overview, action count. One completed sample is already there.
- **Live.** Title, participants, timer, lines as they arrive, Stop. Stopping with no lines is allowed. The notepad then says there was nothing to transcribe.
- **Notepad.** Left: keywords, overview, notes, action items you can check off. Right: transcript, speaker and timestamp. Delete meeting from here.

Copy in the UI uses those words. It does not say "simulated", "fake", or "demo mode".

## Language and layout

Go 1.26 is on this machine. Module path `fireflies-clone`. Stdlib only, unless a test forces otherwise: `net/http`, `html/template`, `embed`, `encoding/json`, `log/slog`.

```text
cmd/app/deps.go              wire the store and the handler
cmd/app/server.go            HTTP server
cmd/app/main/main.go         route table
internal/meeting/meeting.go  types and status
internal/capture/capture.go  Recorder interface + scripted recorder
internal/summary/summary.go  overview, notes, keywords, actions
internal/summary/summary_test.go
internal/store/store.go      JSON file, mutex, seed
internal/store/store_test.go
internal/meeting/handler/    meeting list, notepad, stop, delete
internal/conversation/handler/ live page and transcript lines
web/                         templates and page render
web/templates/*.tmpl
web/static/app.css
data/                        gitignored runtime file
README.md
APPROACH.md
FOLLOWUPS.md                 already written, update answers if the design shifts
```

No Node, no React, no database server. `go test ./...` and `go run ./cmd/app/main` are the whole workflow.

## Packages

**meeting.** The document the rest of the app passes around.

```go
type Status string // active | processing | completed

type Sentence struct {
    Speaker   string
    StartSec  float64
    EndSec    float64
    Text      string
}

type Action struct {
    ID    string
    Text  string
    Owner string // empty if none
    Done  bool
}

type Meeting struct {
    ID           string
    Title        string
    CreatedAt    time.Time
    DurationSec  float64
    Status       Status
    Participants []string
    Sentences    []Sentence
    Keywords     []string
    Overview     string
    Notes        []string
    Actions      []Action
}
```

**capture.Recorder.** One method the live page and the tests can call.

```go
type Line struct {
    Speaker string
    Text    string
}

type Recorder interface {
    // Next returns the next line and false when the script is over.
    Next() (Line, bool)
}
```

The shipped recorder walks a fixed product-meeting script (a few speakers, a decision, a named owner, a first-person commitment, a question). The live page calls it on a short interval and appends sentences using the elapsed timer. Typed lines from the form use the same append path, so a person can talk over the script. A future ASR implementation would satisfy the same interface and the handlers would not change. That seam is the answer to the recording follow-up.

**summary.** Pure function, no I/O.

```go
func Build(sentences []Sentence) Result
```

- Split each sentence on `.?!` if needed, else keep the line.
- Score by content-word overlap. Small boost for decision words: decided, deadline, risk, ship, agreed.
- Overview: best 2 sentences, joined in original order.
- Notes: next-best lines, as bullets.
- Keywords: top content words.
- Actions: commitment lines only (`I'll`, `we'll`, `let's`, `need to`, `Name will`, `follow up`, a weekday or "tomorrow"). A question is not an action. "I'll …" belongs to that line's speaker. "Maya will …" belongs to Maya.
- Empty input: overview says nothing was captured, no invented actions.

**store.** `sync.Mutex` around a JSON file under `data/`. Create, list, get, set sentences, set status, save summary, toggle action, delete. Seed only when the file is missing. IDs from `crypto/rand`, hex, not user input, so paths cannot escape.

**Routes,** registered in `cmd/app/main`, one handler package per business unit:

| Method and path | What it does |
| --- | --- |
| GET / | Meeting list |
| POST /meetings | Create Active meeting, redirect to live |
| GET /meetings/{id}/live | Live page |
| POST /meetings/{id}/lines | Append a line or pull the next scripted line |
| POST /meetings/{id}/stop | Status Processing, run summary, status Completed, redirect to notepad |
| GET /meetings/{id} | Notepad |
| POST /meetings/{id}/actions/{actionID} | Toggle done |
| POST /meetings/{id}/delete | Remove and redirect home |

Templates escape by default. No `template.HTML` on transcript text. Status changes happen in the store before the redirect, so a refresh during processing cannot double-apply. Summary runs once, inside stop.

Processing can be synchronous inside the stop handler. A meeting this size finishes in milliseconds, but the status still passes through `processing` so the model matches the real product. If that flash is invisible, the live page can show "Transcribing…" for a beat before the redirect. Do not add a queue.

## Sample meeting

Ship one completed meeting in the seed, about a pricing decision, so the notepad is reviewable before anyone records. The scripted recorder can reuse that conversation so record and seed stay consistent.

## Build order

### 1. Module and types

- [x] `go mod init`, packages above, meeting types
- [x] Summary tests first, against a fixed fixture (decision, Maya's action, a first-person action, a question, filler)
- [x] Store tests in `t.TempDir()`: create, append, complete, toggle, delete, missing id

Acceptance: `go test ./...` passes with no UI yet.

### 2. HTTP and notepad

- [x] Seed loads
- [x] List page and notepad render the seed
- [x] Toggle action and delete work
- [x] Unknown id returns a normal 404 page

Acceptance: open the seed meeting and check off an action.

### 3. Record path

- [x] Create a meeting, land on the live screen, timer running
- [x] Scripted lines append with speaker and offset
- [x] Optional typed line uses the same path
- [x] Stop runs summary once and opens the notepad with overview, notes, keywords, actions

Acceptance: record, stop, read a notepad that came from that session.

### 4. Write-up

- [x] README: Go version, `go run`, `go test`, where data lives, libraries (stdlib only if that stays true)
- [x] APPROACH.md: the loop, why the Recorder interface, why extractive summary, what was left out, time spent
- [x] If a follow-up answer in FOLLOWUPS.md no longer matches the code, fix the answer

Acceptance: someone else can run it from the README and open both the seed meeting and a meeting they just recorded.

## Out of scope

- A bot that joins Zoom, Meet, or a phone call
- Accounts, teams, calendar auto-join, recap email
- A speech vendor or an LLM key as a requirement
- Video, soundbites, comments, analytics, CRM
- Their logo, their copy, their assets

## Time box

| Phase | Target |
| --- | --- |
| Types, summary, store, tests | 2.5 h |
| Templates and notepad | 2 h |
| Live recording path | 2 h |
| README, APPROACH, polish | 1 h |
| Buffer | 1 h |
| **Total** | **about 8.5 h** |
