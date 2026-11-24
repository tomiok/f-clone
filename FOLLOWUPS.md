# Follow-up questions

Likely questions after they run the app, plus the answer we should be able to give from the code in [PLAN.md](PLAN.md). Update an answer if the implementation drifts. Do not describe a speech model, a bot, or an LLM we did not ship.

## The loop

**Walk from Record to the action items.**

Create stores an Active meeting and redirects to the live page. The page appends sentences (scripted recorder and anything typed) with speaker and elapsed time. Stop sets Processing, `summary.Build` runs once on those sentences, the store saves overview, notes, keywords, and actions, status becomes Completed, and the notepad renders that document. Checking an action only flips `Done`. It does not summarize again.

**Why those screens and not a single form that pastes a transcript?**

Fireflies is a loop: live meeting, a processing status, then a notepad with summary on one side and the transcript on the other. A paste box would meet the words of the brief and miss the product. The placeholder is inside capture, not instead of the loop.

**What happens if stop is clicked twice, or the process dies in between?**

Stop moves Active to Processing, runs `summary.Build` once, then stores Completed. A second stop sees Completed and returns the same meeting, so checked action items stay checked. If the process dies after the Processing write and before Completed, opening the notepad (or stopping again) finishes that meeting. The JSON write is a temp file plus rename, so a crash during the write keeps the previous file.

**What if nobody spoke?**

The meeting still completes. Overview says nothing was captured. There are no made-up action items.

## Recording

**The brief says a placeholder is fine. What is the placeholder?**

`conversation.Recorder` has `Next() (Line, bool)`. The implementation we ship walks a fixed multi-speaker script. The live page turns each line into a sentence. Typed lines use the same append path. Nothing joins a real call.

**How would a real recorder replace it?**

A second type with the same method: read audio, return speaker and text as lines. HTTP handlers stay on the interface. Do not pretend the script is already doing that.

**Why not the microphone in the browser?**

The graded path has to work with no device permission and no vendor account. A browser recorder would be a second implementation of the same interface, not a rewrite. We left it out on purpose.

**Where would a Zoom bot sit?**

Outside this process. The bot would join, stream audio, and push lines into the same append API the live page uses. Calendar auto-join is a scheduler that creates Active meetings and starts that bot. Not in this repo.

## Transcript

**What is a transcript here?**

An ordered `[]Sentence`: speaker, start seconds, end seconds, text. That matches Fireflies' public sentence object closely enough to extend (their extra fields are raw text, per-line tags, and an edit separate from the original).

**How do speakers get their names?**

The script and the typed form already know the name, the way a calendar-backed bot often does. We do not run diarization. Uploads in the real product often come back as Speaker 1 until a person renames them. If they ask us to add rename: update that speaker on the sentences, then call `summary.Build` again, and say why checkmarks on old actions would reset.

**Why keep start and end if we never play audio?**

The notepad shows when the line was said, and a later audio player or "jump to this line" needs the offsets. They are recorded from the live timer, not guessed afterward.

## Summary and action items

**How does the summary work?**

Extractive, in `internal/summary`, no network. Sentences are scored by shared content words, with a boost for decision language. The overview is the top sentences put back in spoken order. Notes are the next-best lines. Keywords are the top content words. The code is deterministic so the tests can lock a fixture.

**Why not an LLM?**

No key, no latency, no cost, and the tests are stable. The cost is that we quote the meeting instead of paraphrasing it, and we miss soft commitments that do not match the patterns. The function signature `Build([]Sentence) Result` is the seam: an HTTP model client can return the same struct.

**What becomes an action, and what does not?**

Commitment language: I'll, we'll, let's, need to, "Maya will", follow up, a deadline. The owner is the speaker for first person, or the named person for "Maya will". A question is never an action. Vague discussion is not an action just because it mentions "should" in passing. Be ready to open the test fixture and show the included and excluded lines.

**A 2-hour meeting?**

This scorer is one pass over the sentences, fine for a challenge. It will over-weight repeated jargon and it will not group topics into chapters. A longer meeting wants chunking or a model behind the same `Build` function. We did not do that.

## Go

**Why stdlib HTTP and templates?**

The app is a few routes and server-rendered pages. `net/http` with method-and-path patterns is enough. A framework would add a dependency without a route group, middleware chain, or JSON API that needs it.

**Where is concurrency?**

The store mutex serializes JSON read-modify-write so two live tabs cannot tear the file. Handlers do not share other mutable state. Summary is pure. We do not start a goroutine per meeting, because stop is fast and a background job would need cancellation and a restart story we do not have.

**Why a JSON file?**

One binary, no database to install, and the data is a document we always load whole. The store interface is small (get, list, save). SQLite would be the swap if we needed queries across many meetings. Say what we would not do: stick JSON and then hand-roll indexes.

**How do you test this?**

Table tests on `summary.Build` for the fixture cases above, including empty input. Store tests use `t.TempDir()` and never the developer's `data/` directory. Handlers are thin. If they ask for an HTTP test, `httptest` around create, append, stop, and a check that stop twice does not duplicate actions.

**Templates and safety.**

`html/template` escapes sentence text. We do not mark transcript text as trusted HTML. Ids in URLs are generated by us, and the store looks up by id. It does not open a user-supplied path.

## Product cuts

**What did you leave out, and why?**

Ask-across-the-transcript, speaker rename plus regenerate, audio playback, channels, share links, analytics. The email asked for recording, transcription, summary, action items, and a simple UI. Those four work without the rest. The first thing to add with another day is speaker rename that re-runs `Build`, because their real notepad does that and it uses the seam we already have.

**How does this differ from Fireflies as it works today?**

Same loop, smaller document. We do not join calls, we do not run their model stack, we do not do Live Assist, AI Skills, or the CRM push. The notepad layout (summary one side, transcript the other, status before it is ready) is the part worth looking like.

## If they ask us to change it live

| Ask | Where the change goes |
| --- | --- |
| Real mic or a pasted transcript | New `Recorder`, or one append path. Do not fork the notepad. |
| Call an LLM for the overview | New body behind `summary.Build`. Keep the fixture tests on the extractive version. |
| Rename a speaker | Update sentences, run `Build` again, replace the saved summary. |
| Search the notebook | Filter `List` by title, speaker, and overview. No new store. |
| Two people at once | The mutex is enough for one process. More than one process needs a real database. |
| Auth | A single owner id on `Meeting` and a check in the handlers. Not a new summary pipeline. |
