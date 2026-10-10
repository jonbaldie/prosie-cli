# Exploratory Testing Report: prosie-cli Imports, Generation, Codex, and Series

**Date:** 2026-10-10
**Scope:** Book and chat import, prose generation (continue, rewrite, undo, summarize), and codex and series management
**Source:** `v0.5.1`, commit `bfb04adf0b23ef666ce2b029bb111156048f130e`
**Build tool:** Go 1.26.3 on macOS arm64, `go build -trimpath` with `GOFLAGS=-mod=readonly`
**Evidence:** [`evidence/2026-10-10/`](evidence/2026-10-10/)

## Setup and starting state

- I built the CLI from `main` into `build/prosie` in the workspace.
- I used a new config file in a temporary directory, with `PROSIE_API_URL`, `PROSIE_CONFIG_PATH`, and `PROSIE_API_TOKEN=test-token`.
- I used a Python fixture on `127.0.0.1`: [`mock-api.py`](evidence/2026-10-10/mock-api.py). It keeps data in memory. Its request and response shapes come from the backend responders in the prosie checkout (commit `af49b46`). It used no live Prosie account.
- The fixture has a stream mode: `normal`, `truncate`, or `error`. Set it with `POST /_fixture/mode`. In `truncate` mode the fixture sends half of the `delta` events, then closes the response with no `done` or `error` event.
- [`r.sh`](evidence/2026-10-10/r.sh) runs one CLI command and records the command, stdout, stderr, and exit code.
- I restarted the fixture once to add a chapter-create route. The restart reset the fixture data. [`fixture-requests.json`](evidence/2026-10-10/fixture-requests.json) and [`fixture-state.json`](evidence/2026-10-10/fixture-state.json) show only the requests and state after the restart.

## Journeys explored

### 1. Import a manuscript and a chat history

**Goal:** Import a DOCX book and a JSON conversation, from a file and from stdin, and get a clear error for bad input.

**Actions:** I imported a DOCX with `--title`, with a title from the filename, and from stdin (`-`). I imported a chat JSON from a file and from stdin. I then tried a missing file, a directory, a non-DOCX file, invalid JSON, an unknown book, and empty stdin.

**Result:** Each import was correct. The title came from `--title`, from the filename, or was "Imported Book" from stdin. The multipart field names and filenames matched the backend contract. Each error path wrote to stderr and exited `1`. Transcript: [`j1-imports.txt`](evidence/2026-10-10/j1-imports.txt).

### 2. Generate, rewrite, undo, and summarize prose

**Goal:** Extend and change chapter prose with AI generation, and get a clear result when the stream fails.

**Actions:** I ran `generate continue` with stream and with `--no-stream`, `--instruction`, `--words`, `--lines`, `--json`, and `--no-persist`. I ran `generate rewrite` with `--selection-file` and `--persist`, then `generate undo` two times. I ran `generate summarize` with and without persist. Then I set the fixture to `error` mode and to `truncate` mode and ran the stream commands again.

**Result:** The ordinary paths were correct. The request bodies matched the backend validation. The second undo returned a 422 error and exit `1`. An `error` event wrote `stream error: Upstream provider failed.` to stderr and exited `1`. A stream that stopped with no `done` event exited `0`. See confirmed bug 1. Transcripts: [`j2-generate.txt`](evidence/2026-10-10/j2-generate.txt), [`j2-stream-failures.txt`](evidence/2026-10-10/j2-stream-failures.txt), [`j2-truncated-replay.txt`](evidence/2026-10-10/j2-truncated-replay.txt).

### 3. Manage codex entries and a series

**Goal:** Create, change, and remove codex entries, and group books into a series.

**Actions:** I created codex entries with `--content`/`--details` and `--type`/`--category`, then listed, showed, updated, and deleted them. I tried an invalid type and a missing `--details`. I created a series, attached a book, showed and updated the series, and detached the book. I cancelled a delete at the prompt.

**Result:** Each operation was correct and the lasting state matched. The invalid type returned a 422 error and exit `1`. The missing `--details` gave a local error and exit `1`. After the detach, `series show --json` omits `story_ids`; this is the known issue [#55](https://github.com/jonbaldie/prosie-cli/issues/55). Transcript: [`j3-codex-series.txt`](evidence/2026-10-10/j3-codex-series.txt).

## Confirmed bugs

### 1. A stream that ends with no `done` event exits `0` with partial prose

**Issue:** [#86](https://github.com/jonbaldie/prosie-cli/issues/86)

**User impact:** The user and scripts cannot see that the output is incomplete. `prosie generate continue 103 && next-step` continues after truncated prose. The backend persists the prose only before it sends `done`, so the chapter can stay unchanged while the CLI reports success.

**Starting state:** A new fixture. Book 101 and chapter 103 with the content "The door was shut and the hall was cold." Fixture stream mode `truncate`.

**Replay:**

1. `curl -X POST -H 'Authorization: Bearer test-token' $PROSIE_API_URL/_fixture/mode -d '{"stream":"truncate"}'`
2. `prosie generate continue 103`
3. `prosie generate rewrite 103 --selection "The door was shut and the hall was cold." --prompt "Tense." --stream --persist`

**Expected:** A non-zero exit code and an error on stderr, the same as the `error` event path.

**Actual:** Step 2 wrote `The lantern guttered. She ` and exited `0`. Step 3 wrote `REWRITTEN(The door was shut ` and exited `0`. Stderr was empty. The chapter content did not change after step 2.

**Repeat check:** Step 2 gave the same result two times from the same state. A run in `normal` mode then appended the full continuation, so the fixture and the chapter were correct. The mode changes were sent with `curl` and are not in the transcript.

## Unresolved candidates

### A cancelled delete with `--json` exits `0` with empty stdout

`prosie codex delete 105 --json` with `n` or with no stdin writes `Deletion cancelled.` to stderr, writes nothing to stdout, and exits `0`. A script cannot tell a cancelled delete from a completed one by the exit code. Closed issue [#12](https://github.com/jonbaldie/prosie-cli/issues/12) proposed exit `1` for a cancelled delete in its fix sketch. But its acceptance criteria require only that the prompt goes to stderr and stdout stays clean, and the CLI does this now. The intended exit code is not specified, so I did not file an issue.

## Rejected candidates

- **Directory as an import file:** `book import somedir` gives `failed to copy file data: read somedir: is a directory` and exit `1`. The message is clear enough, and the behaviour is correct.
- **`--json --stream` on `generate continue`:** The CLI uses the non-stream path. The help text says `--json` disables streaming, so this is correct.
- **Partial text with no newline after an `error` event:** Stdout keeps the partial text and the error goes to stderr with exit `1`. This is correct for a stream.

## Usability observations

- The chapter count in the import message is not grammatical: "with 1 chapters".
- `chat import abc` sends a request to `/stories/abc` and gets `API error (404): Not Found`. The CLI does no local ID check. Issue [#56](https://github.com/jonbaldie/prosie-cli/issues/56) covers ID parsing.
- `series delete` accepts `-y`/`--yes` but not `--force`. This is not a bug, but other CLIs often use `--force`.

## Limits and unexplored areas

- All tests used a fixture, not the live API. The fixture copies the backend shapes but not all backend validation.
- The fixture has no chat stream or chat export, so I did not test `chat stream` with a truncated stream. It uses the same stream decoder, so it is possibly affected by bug 1.
- In the fixture, `rewrite --persist` changes the chapter before it streams, also in `truncate` mode. The backend persists only after the stream completes. Thus, the chapter state after the truncated rewrite is not valid evidence.
- I did not test browser login, chapter word counts, or Ctrl-C cancel during a stream.
