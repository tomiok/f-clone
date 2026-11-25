# f - clone

A small web app in the shape of [Fireflies](https://fireflies.ai/): record a meeting, keep a transcript, and open a notepad with an overview, notes, and action items.

It is one Go binary. Pages are `html/template`. There is no database. The router is chi.

## Run

Go 1.26 or newer.

```bash
make test
make run
make lint
```

`make install-tools` installs golangci-lint. `go test ./...` and `go run ./cmd/app/main` do the same as `make test` and `make run`.

Open http://localhost:8080. The notebook already contains one finished meeting, "Q3 pricing decision". Record starts a live meeting. Lines appear on their own. Stop pauses the clock, and Resume continues the transcript from that moment without starting the clock again. When the conversation finishes, the notepad opens.

The Google and Outlook buttons do nothing. There is no way to type a line on the live page.

`PORT` changes the port (default `8080`). Meetings are stored in `data/meetings.json`, or in `DATA_FILE` if that is set. That directory is created on startup and is not part of the git repo. The time on each meeting uses the machine timezone (`TZ`, or `/etc/localtime`). `America/Argentina/*` and `America/Buenos_Aires` show as `BS AS/Argentina`.

### Docker

No Go install is required.

```bash
docker build -t f-clone .
docker run --rm -p 8080:8080 -e TZ=America/Argentina/Buenos_Aires -v f-clone-data:/data f-clone
```

`make docker` builds the same image. The JSON file lives in the `f-clone-data` volume at `/data/meetings.json`.

## Assumptions

- This is the website only. There is no desktop app, browser extension, login, or calendar connection. The calendar buttons are placeholders.
- Recording does not join a call and does not use a microphone. `conversation.Recorder` walks `MockConversation`, a fixed script in `conversation_mock.go`. A real recorder is another type that implements `Conversation`.
- The transcript is those lines, with the speaker and the time since the meeting started. Nothing on the server turns audio into text.
- The overview, notes, keywords, and action items are extracted from the transcript in process. They quote the meeting. They do not call an LLM.
- One process, one JSON file. The store mutex is enough for that. More than one process needs a real database.
- Action items are English commitment lines (`I'll`, `let's`, `need to`, `Maya will`, `follow up`). A question is not an action.
- Meetings are not private to a user. Anyone who can open the server can open them.

## Libraries

[chi](https://github.com/go-chi/chi) for the route table in `cmd/app/main`. Everything else is the standard library: `net/http`, `html/template`, `embed`, and `encoding/json`.
