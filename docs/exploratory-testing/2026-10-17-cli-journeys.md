# Exploratory Testing Report: prosie-cli Authoring, Generation, and Chat

**Date:** 2026-10-17
**Scope:** Write and keep chapter prose, generate and recover from a save conflict, and sign in to ask the book a question
**Source:** `v0.5.3`, commit `238113022af625fbcf9e8b02d0da1b29bd1e79b4`
**Build tool:** Go 1.26.3 on macOS arm64, `go build -trimpath` with `GOFLAGS=-mod=readonly`
**Evidence:** [`evidence/2026-10-17/`](evidence/2026-10-17/)

## Setup and starting state

- I built the CLI from `main` into `build/prosie` in the workspace. `go.mod` has no external modules.
- I used a new config file in the run temporary directory, with `PROSIE_API_URL`, `PROSIE_CONFIG_PATH`, and `PROSIE_API_TOKEN=test-token` unless a step cleared them.
- I used a Python fixture on `127.0.0.1`: [`mock-api.py`](evidence/2026-10-17/mock-api.py). It keeps data in memory. Shapes follow `routes/api.php` and the CLI client. It used no live Prosie account.
- [`record.sh`](evidence/2026-10-17/record.sh) runs one CLI command and records the command, stdout, stderr, and exit code.
- The local prosie checkout used for contract notes was 42 commits behind its remote `main`. The fixture is not that server.

## Journeys explored

### 1. Write a book and keep the prose

**Goal:** Create a book, put prose in chapters from a file and from stdin, reorder and export the book, and keep a duplicate independent of later edits.

**Actions:** I created a book with a premise and a target of 1000 words. I updated Chapter 1 from a file, created a chapter from stdin, and rejected a combined `--content` and `--file`. I reordered chapters, exported Markdown, duplicated the book, then changed the original chapter. I imported a DOCX from a file and from stdin, cancelled a delete, and confirmed a delete.

**Result:** File and stdin prose kept spaces and line breaks. Reorder put the named chapters first and left the omitted chapter after them. Export bytes matched the fixture Markdown, including paragraph breaks. The duplicate kept the prose from before the later edit. A bad DOCX exited 1 with the API message on stderr. A cancelled delete left the chapter in place. Transcript: [`j1-author.txt`](evidence/2026-10-17/j1-author.txt).

### 2. Generate prose and recover from a save conflict

**Goal:** Continue, rewrite, summarize, undo, and reject chapter prose, and see unsaved text when another session changes the chapter.

**Actions:** I ran `generate continue` with `--instruction`, `--words`, and `--lines`, then with `--no-persist`. I rewrote a selection and undid it. I summarized the chapter. I set the fixture to `conflict`, `truncate`, and `error`, and ran the stream commands again, including `chat stream`. I sent SIGINT during a delayed stream.

**Result:** Ordinary paths saved or withheld prose as the flags requested. A second undo exited 1. A conflict exited 1, wrote the unsaved text to stderr, and did not change the chapter. Non-streaming and `--json` conflicts left stdout empty. Streaming conflicts left the provisional tokens on stdout. A stream with no `done` event exited 1. SIGINT printed `Generation cancelled.` and called the cancel route. Transcripts: [`j2-j3.txt`](evidence/2026-10-17/j2-j3.txt), [`sigint.txt`](evidence/2026-10-17/sigint.txt).

### 3. Sign in, ask the book a question, and change the model

**Goal:** Log in, send a chat turn, keep the thread, export it, and change the model without printing a provider key.

**Actions:** I checked status with no token, logged in with `--token`, logged out while `PROSIE_API_TOKEN` was set, and completed device login with `--no-browser --json`. I sent a chat, showed it, streamed a reply, exported JSON, and imported that JSON. I cancelled a delete, failed a send, and continued the created thread. I updated the model with `PROSIE_OPENROUTER_API_KEY` set.

**Result:** Token login saved the session. Logout warned that the environment token stayed active. Device login wrote two complete JSON values: the user code, then the authenticated result. Chat show and export kept the turns. A failed send named the new conversation id, and a later send on that id succeeded. The key was in the request body and was not in stdout or stderr. `llm show` reported the key as configured. Transcript: [`j2-j3.txt`](evidence/2026-10-17/j2-j3.txt). Extra checks: [`probes.txt`](evidence/2026-10-17/probes.txt).

## Confirmed bugs

### 1. `book create --target-words` drops a non-positive target and exits 0

**Issue:** [#96](https://github.com/jonbaldie/prosie-cli/issues/96)

**User impact:** A writer who sets a bad target sees a successful create. The book has no target. `book show` does not show the value they set.

**Starting state:** A reset fixture. No books. `PROSIE_API_TOKEN=test-token`.

**Replay:**

1. `prosie book create --title "Negative Target" --target-words -5 --json`
2. `prosie book show 101 --json`
3. `prosie book create --title "Valid Target" --target-words 1000 --json`

**Expected:** The command does not report success for a target it did not save. It exits 1, as `generate continue --words 0` does, or it sends `target_word_count` and shows the API error.

**Actual:** Step 1 exited 0. Stderr was empty. The request body was `{"title":"Negative Target"}`. The saved book had no `target_word_count`. `--target-words 0` did the same. `--target-words 1000` sent and saved the target. `book update --target-words 0` does send the field.

**Repeat check:** The negative create gave the same result two times after a fixture reset. Evidence: [`bug-target-words.txt`](evidence/2026-10-17/bug-target-words.txt), [`bug-target-words-requests.json`](evidence/2026-10-17/bug-target-words-requests.json).

## Unresolved candidates

None. A cancelled delete still exits 0 with empty stdout. That remains the unresolved note from the 2026-10-10 report. Closed issue [#12](https://github.com/jonbaldie/prosie-cli/issues/12) does not require exit 1. I did not file it again.

## Rejected candidates

- **Stdin book import title:** `book import -` with no `--title` sends `title=Imported Book`. The help text is correct.
- **Stream that ends with no `done` event:** `generate continue`, `generate rewrite --stream`, and `chat stream` exit 1 and write `stream ended before completion` to stderr. Issue [#86](https://github.com/jonbaldie/prosie-cli/issues/86) stays fixed.
- **Conflict candidate:** The unsaved text is on stderr. Non-streaming and `--json` leave stdout empty. Streaming leaves only the provisional tokens on stdout. The chapter does not change.
- **Provider key in LLM output:** `llm update --json` with `PROSIE_OPENROUTER_API_KEY` set does not print the key. `llm show` prints `configured`.
- **Message that starts with `--`:** `chat send <book-id> -- --why` sends the message `--why`.
- **Malformed config:** With a broken `config.json`, `book list` exits 1 even when `PROSIE_API_TOKEN` is set. The stderr names the file and the JSON error. This matches the fix for [#50](https://github.com/jonbaldie/prosie-cli/issues/50).

## Usability observations

- `book import` says `with 1 chapters`. The 2026-10-10 report noted the same text.
- `chapter show` prints stored HTML, including `<p>` tags. `chapter export` returns the same chapter as Markdown paragraphs. After `generate continue`, the stream prints plain prose and `chapter show` prints the saved HTML.
- `chat send` text output prints the answer and not the new conversation id. `chat send --json` includes `conversation_id`. A failed send prints the id on stderr.
- `book export --format DOCX` is sent unchanged. The fixture, like the API `format=docx` check, treats that as Markdown. The CLI writes those bytes to the `--output` path and exits 0.
- A title that contains a newline breaks the `book list` row. Other rows stay aligned.
- `auth login --json` writes two JSON values. Each value is complete. This matches closed issue [#48](https://github.com/jonbaldie/prosie-cli/issues/48).

## Limits and unexplored areas

- All checks used the fixture, not the live API. The fixture does not apply every backend validation rule.
- I did not open a browser for device login. I used `--no-browser`.
- I did not test series or codex commands. The 2026-10-10 report covers those journeys.
- I did not test a live DOCX import against the backend parser. The fixture accepts a file that starts with `PK`.
