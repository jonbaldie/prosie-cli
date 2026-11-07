# Exploratory testing: sign-in, backups, and chat

**Date:** 2026-11-07.

**Source:** `origin/main` at `52af84688925b3f418b6f798bf548aa4bc603a46`, also tagged `v0.8.0`.

**Result:** Three journeys completed against a local fixture. Separate build and test checks found two confirmed bugs, [#119](https://github.com/jonbaldie/prosie-cli/issues/119) and [#121](https://github.com/jonbaldie/prosie-cli/issues/121). No product code was changed.

## Setup

Go 1.26.8 on Linux amd64 built the CLI with `GOFLAGS=-mod=readonly`. `go.mod` has no external dependencies and there is no lockfile or dependency installation step. The Go cache, module cache, temporary files, binary, and synthetic Git repository were inside the worktree.

The [driver](evidence/2026-11-07/run.py) starts the existing [HTTP fixture](evidence/2026-10-17/mock-api.py) on a random loopback port. It uses synthetic credentials and an empty fixture state. The driver owns the server and stops it in a `finally` block. Its temporary directory is also removed on exit. No Docker resources or browser sessions were started. The fixture supplies API responses and stored state. These results do not establish that the live backend behaves the same way.

## Journeys

### Sign in and sign out

A writer needs a session that another CLI process can use, then needs to remove it. Token login worked, and a new process read the session. Both a new credential file and a replacement for a file with mode `0644` had mode `0600`. Logout removed the file, and status then exited 1. The synthetic device flow also completed with `--no-browser --json`, and status read its saved session. This repeats the permission check associated with [#101](https://github.com/jonbaldie/prosie-cli/issues/101). It does not test a real browser or OAuth provider.

### Save prose and back up a book

A writer needs saved prose and a recoverable export. The driver created a book, updated its chapter from standard input, and read the exact spaces and paragraph breaks in a new process. A zero word target exited 1. Markdown exports matched JSON `content`. An output path with a missing parent exited 1.

For the book and chapter, DOCX file output, raw output, and base64-decoded JSON all matched the fixture's 796-byte archive. The ZIP check passed. Each JSON export was checked twice. This checks the repair for [#103](https://github.com/jonbaldie/prosie-cli/issues/103); the fixture document is fixed and does not render the saved prose.

The driver also checked deletion. An extra positional argument exited 1 before any request. Cancelled deletion left the book in fixture state. Confirmed deletion removed it. The cancellation prompt and message went to stderr.

### Keep a chat thread

A writer needs a saved conversation that can be exported and restored. The driver sent a question, continued it, and read the saved messages. It exported and imported the thread, then compared the message text. An invalid JSON import exited 1 without adding a conversation. A fixture-controlled 502 caused a send to exit 1; a later send succeeded. A JSON stream completed and a later `chat show` included the turn. The fixture did not test a live model or the new `chat retry` command.

The [transcript](evidence/2026-11-07/transcript.txt) records commands, output, and exit codes. The [requests](evidence/2026-11-07/requests.json) omit authorization headers. The [DOCX metadata](evidence/2026-11-07/docx-source.json) records the archive hash.

## Confirmed bug

### A local binary can report another repository's version

[Issue #119](https://github.com/jonbaldie/prosie-cli/issues/119) has the complete reproduction. This is related to the closed issue [#60](https://github.com/jonbaldie/prosie-cli/issues/60), not a new report of its original fixed symptom.

A clean `v0.8.0` source checkout built the binary twice. Both binaries ran in a synthetic Git repository tagged `v9.9.9`. Both runs printed `prosie version 9.9.9`. Removing the synthetic tag made the same binary print `prosie version 0.0.0-dev`. The README says a clean exact-tag local build reports the source tag. A user can therefore mistake the installed binary's version for an unrelated project's version. The [replay output](evidence/2026-11-07/version-replay.txt) records the checks. The issue contains commands to repeat them.

## Other observations and limits

The first driver attempt stopped at a test assertion after the cancelled deletion. The fixture stores book IDs as strings, but the driver checked for an integer key. The [first transcript](evidence/2026-11-07/first-attempt.txt) and [error](evidence/2026-11-07/first-driver.log) are preserved. After correcting the driver, the full run passed. This was a driver error, not a product bug.

The first cleanup port probe failed with `address already in use` after the server stopped. The probe did not set `SO_REUSEADDR`. TCP connections in `TIME_WAIT` can prevent that rebind even after the listener stops. The [error](evidence/2026-11-07/cleanup-first.log) is preserved. The corrected probe rebinds the port with `SO_REUSEADDR` after the server thread stops. The [cleanup record](evidence/2026-11-07/cleanup.txt) gives its PID and port.

The first `go test ./...` run failed on the clean tagged worktree. The [initial test log](evidence/2026-11-07/go-test-initial.log) records the failures. After the tracked documentation index was edited, the source checkout was no longer clean and the [same test command passed](evidence/2026-11-07/go-test.log).

A follow-up isolated the output failure test from the tagged binary directory. On a clean `v0.8.0` checkout, `TMPDIR` was an untagged nested Git repository. `go test ./internal/cmd -run '^TestOutputFailureStateIsPerExecution$' -count=1` still failed both subtests. The test expects `0.0.0-dev`, but the version command printed `0.8.0`. The [targeted test log](evidence/2026-11-07/go-test-clean-tag.log) records the result. [Issue #121](https://github.com/jonbaldie/prosie-cli/issues/121) tracks this test failure. The earlier explanation that the test binary location alone caused it was not established.

No unresolved product candidates remain from the selected journeys. Live authentication, production data, real model responses, series, codex, and `chat retry` were not tested. The fixture does not prove backend conformance. No usability recommendation is made from the fixture-only pass.

## Replay

From a checkout, set `TMPDIR` to an existing scratch directory and run:

```sh
GOFLAGS=-mod=readonly go build -o "$TMPDIR/prosie" .
python3 docs/exploratory-testing/evidence/2026-11-07/run.py "$TMPDIR/prosie"
```

The driver overwrites its transcript, request capture, and DOCX metadata in its own evidence directory. Use a disposable checkout to preserve the published evidence. It exits 0 when its assertions pass. The synthetic server shuts down on normal exit or a Python exception. A forced process termination is outside that guarantee.
