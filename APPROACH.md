# What I built, and why

f - clone is a Go web app with three screens: the meeting list, a live meeting, and a notepad. Summary sits on the left of the notepad, the transcript on the right. That is the Fireflies loop (capture, then notes), not the rest of the product (calendar bots, email, Slack, voice agents).

The brief allows a placeholder for recording, so capture is an interface, `capture.Recorder`, with one method: `Next() (Line, bool)`. The version in this repo walks a short multi-speaker script. Typed lines go through the same append path. The handlers do not know which one produced a sentence. A microphone or a bot would be a second implementation, not a new notepad.

Stopping the meeting runs `summary.Build` once. It is a pure function: score sentences, quote the strongest ones as the overview and the notes, pull keywords, and keep commitment lines as action items with an owner when the line has one. I did not call an LLM. The result is testable, it needs no API key, and the same function signature is where a model would plug in later. Checking an action item only flips `Done`. A second stop does not rebuild the list, so those checkmarks stay.

State moves from active, to processing, to completed. The processing write happens before the summary is stored. Opening the notepad finishes a meeting that was left in processing. The file is replaced with a temp-file rename.

I left out accounts, sharing, search, speaker rename, audio playback, and Ask. The first thing I would add is renaming a speaker and running `Build` again, because the real notepad does that and this code already has the seam.

Time: one working session. A fair estimate for building this by hand is about six hours, inside the ten-hour cap.
