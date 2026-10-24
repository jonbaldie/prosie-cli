# Exploratory testing: sign-in, document backup, and chat

**Date:** 2026-10-24  
**Source:** remote `main`, `4710664d789b0e325cffdbae4edd36e35c07e867`  
**Result:** Three tasks completed. One confirmed bug: [#103](https://github.com/jonbaldie/prosie-cli/issues/103). No product changes.

## Setup

- Go 1.26.3 on macOS arm64. Build: `GOFLAGS=-mod=readonly go build -trimpath -o build/exploratory-2026-10-24/prosie .`.
- The binary SHA-256 was `1927647a2ccecbd38e5778358a00710ebfeddb9e0e7abe61b95893337c8714ba`.
- `go.mod` has no external dependencies. There is no dependency lockfile or install step. Read-only module mode prevented dependency changes.
- Build output stayed in this workspace. Go used the supplied shared build cache. Scratch config and exported files used the supplied `TMPDIR`. No dependency or build path was a symlink into another checkout.
- Each run started a new HTTP fixture on a random loopback port, with empty books and conversations and no saved credentials. Tokens and user details were synthetic. No live account was used.
- The [driver](evidence/2026-10-24/run.py) reuses the [2026-10-17 fixture](evidence/2026-10-17/mock-api.py). It replaces DOCX responses with a deterministic ZIP that contains the document XML, relationships, and content types. The earlier fixture returned only a short ZIP-like header and text. That could not expose binary encoding loss.

## Tasks and outcomes

### 1. Sign in, reopen the session, and sign out

**Success criterion:** A new CLI process can use the saved session. The credential file is private. Logout removes it.

I checked status without credentials, used token login, checked status in another process, and logged out. I then repeated login over an existing config file with mode `0644`. Both new and replaced credential files had mode `0600`. Logout removed the file, and status exited 1. Device login with `--no-browser --json` completed, and a later status command used that session.

This confirms the observed permission behavior after [#101](https://github.com/jonbaldie/prosie-cli/issues/101). Device login still emits two separate JSON values, as documented in the previous report.

### 2. Save prose and export a backup

**Success criterion:** Saved prose keeps its spaces and paragraph breaks. Exported document bytes can be recovered without loss, or the command reports an error.

I created a book with a 1000-word target, updated its chapter from stdin, and read the chapter in a new process. Prose matched the input exactly. A zero target exited 1. Book and chapter Markdown exports matched their JSON `content`. An output path with a missing parent directory exited 1, with the error on stderr.

For both resources, DOCX file output and raw stdout matched the fixture response exactly and passed ZIP validation. DOCX JSON output lost data. Each resource failed twice, and a full repeat from new fixture and config state gave the same result.

### 3. Ask a question, keep the thread, and retry a failed send

**Success criterion:** A saved thread keeps each message. Export and import preserve message text. A failed send reports an error and permits a later retry.

I sent a question, continued the conversation with a multi-word message, and read it back. The first word stayed in place. I exported the thread to a file, imported it, and read the new thread. Message text matched. An invalid JSON import exited 1 and did not add a conversation. A fixture-controlled HTTP 502 send exited 1 with empty stdout. A retry on the same conversation succeeded. A JSON stream also succeeded, and a later `chat show` included the saved turn.

All CLI commands, stdout, stderr, exit codes, and archive checks are in the [transcript](evidence/2026-10-24/transcript.txt). The [request capture](evidence/2026-10-24/requests.json) omits authorization headers.

## Confirmed bug

### DOCX JSON exports lose binary data and exit 0

**Issue:** [#103](https://github.com/jonbaldie/prosie-cli/issues/103)

**Impact:** A backup script cannot recover the document from JSON `content`, despite a successful exit.

**Starting conditions:** The fixture returns the same 796-byte DOCX for each export mode. Book 101 and chapter 102 exist. The CLI has a saved fixture session.

**Replay:**

```sh
prosie book export 101 --format docx -o book.docx --json
prosie book export 101 --format docx > raw.docx
prosie book export 101 --format docx --json > book.json
python3 -c 'import json; from pathlib import Path; Path("from-json.docx").write_bytes(json.load(open("book.json"))["content"].encode())'
python3 -m zipfile -t from-json.docx
```

Repeat with `chapter export 102`. To run the full isolated replay from the repository root:

```sh
GOFLAGS=-mod=readonly go build -trimpath -o build/exploratory-2026-10-24/prosie .
python3 docs/exploratory-testing/evidence/2026-10-24/run.py build/exploratory-2026-10-24/prosie
```

**Expected:** Preserve binary data with an explicit lossless encoding, or reject this flag combination with exit 1 and an error on stderr. Do not silently replace bytes.

**Actual:** Both commands exit 0 with empty stderr. JSON reports 796 bytes, but `content` contains 227 replacement characters. UTF-8 encoding produces 1250 bytes. ZIP validation fails with `Bad offset for central directory`. File output and raw stdout preserve all 796 bytes.

The [source metadata](evidence/2026-10-24/docx-source.json) records the source hash. Both JSON replays produced SHA-256 `36e1e9858dcda9924a312708596380f655e2cf24f793068c0c7a3e7b1cc1a265`, not the source hash. `ExportProse` in `internal/command/export.go` converts arbitrary download bytes to a string before JSON encoding. Invalid UTF-8 becomes U+FFFD.

## Other observations and limits

- No unresolved candidates remain. Credential permissions, zero-target rejection, Markdown JSON content, continued chat messages, and output-file errors behaved as expected.
- Workaround for #103: use `--format docx --output backup.docx --json`. JSON then reports file metadata, and the file preserves the bytes.
- The SUT was the built CLI process. Exit codes, stdout, credential files, downloaded files, and outgoing requests came from that process. The fixture supplied API replies and saved state; its behavior is not evidence that the real backend works.
- The DOCX archive checks establish byte preservation and ZIP integrity, not successful opening in Microsoft Word. The fixture DOCX is fixed and does not render the edited chapter.
- No live OAuth provider, browser approval, LLM, DOCX parser, network retry policy, series, or codex task was tested. None of the selected tasks was blocked.

## Checks and cleanup

- [Full driver replay](evidence/2026-10-24/driver.log): exit 0.
- [`go test ./...`](evidence/2026-10-24/go-test.log), with read-only modules: exit 0.
- [`make messgo`](evidence/2026-10-24/messgo.log): exit 0.
- [Run log](evidence/2026-10-24/run-log.md): setup and diagnostic notes.
- The driver stopped its server and removed scratch files. The run build and Python bytecode were removed. The pre-existing `build/prosie` was not changed.
- This run created no Docker containers, images, volumes, or networks. There were no run-created Docker resources to remove. Unrelated resources were not changed.
