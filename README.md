# f - clone

A small web app in the shape of [Fireflies](https://fireflies.ai/): record a meeting, keep a transcript, and open a notepad with an overview, notes, and action items.

It is one Go binary. Pages are `html/template`. There is no database and no third-party module.

## Run

Go 1.26 or newer.

```bash
go test ./...
go run ./cmd/app
```

Open http://localhost:8080. The notebook already contains one finished meeting, "Q3 pricing decision". Record starts another. On the live page the notetaker adds lines on its own, and you can add one yourself. Stop opens the notepad.

`PORT` changes the port (default `8080`). Meetings are stored in `data/meetings.json`, or in `DATA_FILE` if that is set. That directory is created on startup and is not part of the git repo.

## Assumptions

- This is the website only. There is no desktop app, browser extension, login, or calendar connection.
- Recording does not join a call and does not use a microphone. `capture.Recorder` walks a fixed conversation, and typed lines use the same path. A real recorder would be another type behind that interface.
- The transcript is those lines, with the speaker and the time since the meeting started. Nothing on the server turns audio into text.
- The overview, notes, keywords, and action items are extracted from the transcript in process. They quote the meeting. They do not call an LLM.
- One process, one JSON file. The store mutex is enough for that. More than one process needs a real database.
- Action items are English commitment lines (`I'll`, `let's`, `need to`, `Maya will`, `follow up`). A question is not an action.
- Meetings are not private to a user. Anyone who can open the server can open them.

## Libraries

None beyond the Go standard library: `net/http`, `html/template`, `embed`, and `encoding/json`.
