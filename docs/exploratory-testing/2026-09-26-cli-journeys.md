# Exploratory Testing Report: prosie-cli Journeys

**Date:** 2026-09-26
**Scope:** Authentication, book and chapter work, chat, JSON output, and local version builds
**Source:** tag `v0.4.0`, commit `36a887950e37d6b5ac3e645d43db76452f5e7c34`
**Build tool:** Go 1.26.3 on macOS arm64
**Evidence:** [`evidence/2026-09-26/`](evidence/2026-09-26/)

## Setup and starting state

- I built the CLI from the source tag.
- I used a new config file in `/tmp`. It held a test token for a local API fixture.
- I used a Python server on `127.0.0.1`. It kept books, chapters, and chat in memory. It used no live Prosie account or API token.
- I saved the CLI output, exit codes, and API request bodies. The fixture script is [`mock-api.py`](evidence/2026-09-26/mock-api.py).
- I stopped the fixture after the test. The fixture data did not reach Prosie.
- I also downloaded the official arm64 asset for `v0.4.0`. Its checksum matched the published checksum file.

## Journeys explored

### 1. Log in, reopen, and log out

**Goal:** Save a token, use it in a new CLI process, then remove it.

**Actions:** I checked status with no token, logged in with a test token, checked status again, logged out, then checked status again.

**Result:** Login returned the user and scopes. A new process read the saved token. The config file had mode `0600`. Logout removed the saved token. The last status call returned `authenticated: false` and exit code `1`. Each command wrote no error to stderr. The full output is in [`cli-transcript.txt`](evidence/2026-09-26/cli-transcript.txt).

### 2. Create and remove a book and chapter

**Goal:** Create a book, write a chapter, export the prose, then remove the test data.

**Actions:** I created a book, showed it, listed its first chapter, created and updated a second chapter, exported the chapter and book, then deleted the chapter and book. I listed books after removal.

**Result:** Each command returned valid JSON and exit code `0`. The final book list was empty. The request capture shows the data sent to the fixture. [`final-state.json`](evidence/2026-09-26/final-state.json) shows no books or chapters.

### 3. Send and stream chat messages

**Goal:** Start a chat, continue it with a multiword message, stream a reply, and export the conversation.

**Actions:** I sent a message to a new conversation, sent a second message with `--conversation`, streamed a third message with `--json`, then exported the conversation.

**Result:** Each command returned valid JSON and exit code `0`. The fixture received the full second message, including its first word. The stream returned the full assistant message. The export held all sent messages and replies. This confirms that the old first-word loss in [issue #14](https://github.com/jonbaldie/prosie-cli/issues/14) did not recur in this build. The outputs and request bodies are in [`cli-transcript.txt`](evidence/2026-09-26/cli-transcript.txt) and [`requests.json`](evidence/2026-09-26/requests.json).

## Confirmed bugs

### Local builds use an old version on a release tag

**Issue:** [#60](https://github.com/jonbaldie/prosie-cli/issues/60)

**User impact:** A local build at a release tag can report the wrong version. The local release script can also give its archives the wrong version in their names.

**Starting state:** Source tag `v0.4.0`; no version argument or `VERSION` value for the local release script.

**Replay:**

1. Run `go build -trimpath -o /tmp/prosie .`.
2. Run `/tmp/prosie --version`.
3. Run `DIST_DIR=/tmp/prosie-dist ./scripts/build-release.sh`.
4. Extract the arm64 archive and run its `prosie --version` command.

**Expected:** The local binary and archive use version `0.4.0`, from the checked out tag.

**Actual:** The local binary prints `prosie version 0.1.0`. The release script builds `v0.1.0`, names the archives `prosie_0.1.0_*`, and its binary prints `prosie version 0.1.0`.

**Repeat check:** The plain local build and the no-argument release build both gave `0.1.0`. The official `v0.4.0` binary prints `0.4.0`, and its checksum passed. The local release script also prints `0.4.0` when I pass `v0.4.0`. See [`version-check.txt`](evidence/2026-09-26/version-check.txt), [`release-local-default.log`](evidence/2026-09-26/release-local-default.log), [`release-local-explicit-version.log`](evidence/2026-09-26/release-local-explicit-version.log), and [`official-release-checksums.txt`](evidence/2026-09-26/official-release-checksums.txt).

### Root `--json` is sent as chat text after `--`

**Issue:** [#58](https://github.com/jonbaldie/prosie-cli/issues/58), already open. I did not create a duplicate issue.

**User impact:** The command can send extra text to a chat and can return plain text when the user asked for JSON.

**Starting state:** The local fixture had an open conversation with ID `300`.

**Replay:** Run `prosie --json chat send --conversation 300 -- 'Hold the lantern for the final beat.'`.

**Expected:** Exit code `0`, JSON output, and the message `Hold the lantern for the final beat.`

**Actual:** Exit code `0`, plain text output, and the fixture received `Hold the lantern for the final beat. --json`.

**Repeat check:** The same call returned the same wrong output twice. A call with `--json` after the command flags returned JSON. See [`global-json-repeat.txt`](evidence/2026-09-26/global-json-repeat.txt) and [`requests.json`](evidence/2026-09-26/requests.json).

## Rejected candidates

- The earlier first-word loss in chat send did not recur. The fixture received the whole message.
- `chat stream --json` worked. It returned JSON and exit code `0`.
- The book and chapter create, update, export, and delete actions worked in this fixture.

## Usability observations

- A root `--json` flag can become part of the chat message when the command uses `--`. This can change both the request and the output format. Issue #58 tracks this behaviour.
- A default local build can say `0.1.0` on the `v0.4.0` source tag. This can make a local release build hard to identify. Issue #60 tracks this behaviour.

## Limits and unexplored areas

- I did not use the live Prosie API or a live account. The fixture checked CLI output and request bodies, not backend rules.
- I did not test series, codex, DOCX import, generation, browser login, or long streams.
- I tested selected `--json` commands. I did not check every CLI command.
