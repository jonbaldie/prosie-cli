# Exploratory Testing Report: prosie-cli Journeys

**Date**: 2026-09-16  
**Scope**: Core CLI user journeys (Book lifecycle, Codex & Series management, AI Generation & Chat workflows)  
**Binary build**: `bin/prosie` (built from `main` branch with Go 1.24)  
**Configuration**: Production API (`https://prosie.app`) and isolated recording mock server  
**Evidence directory**: `scratch/evidence/`

---

## 1. Setup and Starting State

- **Environment**: macOS, `go version go1.24.0 darwin/arm64`.
- **Authentication**: Authenticated against `https://prosie.app` with `read`, `write`, and `generate` API token scopes.
- **Isolated test artifacts**: Created temporary books, chapters, series, and codex entries during tests and deleted them during clean up.
- **Quality gate verification**: Ran `make messgo` and `go test ./...` to verify the baseline build.

---

## 2. Journeys Explored

### Journey 1: Book and Chapter Authoring Lifecycle
- **Goal**: Create a book, inspect book state, create chapters with prose, reorder chapters, export book and chapters, update chapter content, delete chapters, and delete the book.
- **Actions Attempted**:
  1. `prosie book create --title "Journey One Exploration" --json`
  2. `prosie book show <id>` and `prosie book show <id> --json`
  3. `prosie chapter create <book-id> --title "Chapter 2: The Arrival" --content "..." --json`
  4. `prosie chapter list <book-id>`
  5. `prosie chapter reorder <book-id> <id2,id1> --json`
  6. `prosie chapter export <chapter-id>` and `prosie chapter export <chapter-id> --json`
  7. `prosie book export <book-id> --json`
  8. `prosie chapter update <chapter-id> --title "..." --summary "..." --json`
  9. `prosie chapter delete <chapter-id> --yes --json`
  10. Attempt to delete last chapter in book: `prosie chapter delete <last-chapter-id> --yes --json`
  11. `prosie book delete <book-id> --yes --json`
- **Observed Outcomes**:
  - Chapter creation, list, reorder, and export functioned correctly.
  - Attempting to delete the last chapter correctly failed with HTTP 422 ("A story must keep at least one scene"), wrote the error message to `stderr`, and exited with status code 1.
  - Deletion commands with `--json` revealed a bug in confirmation handling (see Issue #12).

### Journey 2: Series and Codex Lore Management
- **Goal**: Create a series, attach and detach a book, create story codex entries, view and update codex entries, and clean up.
- **Actions Attempted**:
  1. `prosie series list` and `prosie series list --json`
  2. `prosie series create --title "Chronicles of the Bell" --description "..." --json`
  3. `prosie series attach <series-id> <book-id> --json`
  4. `prosie series show <series-id> --json`
  5. `prosie codex create <book-id> --name "The Low Gate" --details "..." --type "lore" --json`
  6. `prosie codex list <book-id>`
  7. `prosie codex show <codex-id>`
  8. `prosie codex update <codex-id> --details "Updated details" --json`
  9. `prosie series detach <series-id> <book-id> --json`
  10. `prosie codex delete <codex-id> --yes --json`
  11. `prosie series delete <series-id> --yes --json`
- **Observed Outcomes**:
  - Series creation, attachment, detachment, and deletion operated as expected.
  - Codex creation, show, list, and update worked as expected for story-level entries.

### Journey 3: AI Generation & Novel Chat Workflow
- **Goal**: Check LLM provider configuration and available models, test chat message sending and streaming, export and import chat conversations, and evaluate generation flags.
- **Actions Attempted**:
  1. `prosie llm show` and `prosie llm show --json`
  2. `prosie llm models --json`
  3. `prosie chat list <book-id> --json`
  4. `prosie chat show <conv-id> --json`
  5. `prosie chat export <conv-id> --json`
  6. `prosie chat send -c <conv-id> <message>`
  7. `prosie chat stream <book-id> <message> --json`
  8. `prosie generate summarize <chapter-id>`
- **Observed Outcomes**:
  - `prosie llm show` and `prosie llm models` returned structured provider and model data.
  - `prosie chat stream` rejected the `--json` flag with an error (see Issue #13).
  - `prosie chat send -c <conv-id>` dropped the first word of multi-word messages (see Issue #14).
  - `prosie generate summarize` hardcodes `persist: true` and lacks `--no-persist` (see Issue #15).

---

## 3. Confirmed Bugs Filed

| Issue | Title | Description |
| --- | --- | --- |
| [#12](https://github.com/jonbaldie/prosie-cli/issues/12) | Deletion confirmation prompts write to stdout, corrupting `--json` output | `ConfirmDeletion` outputs prompts and cancellation notices to stdout instead of stderr. When `--json` is supplied, stdout is corrupted with plain text, breaking JSON parsers. |
| [#13](https://github.com/jonbaldie/prosie-cli/issues/13) | `chat stream` command does not define `--json` flag, failing with exit code 1 | `prosie chat stream` omits `--json`, violating the requirement that all commands must support `--json`. |
| [#14](https://github.com/jonbaldie/prosie-cli/issues/14) | `messageInput` drops first word of multi-token messages when continuing conversation with `--conversation` | In `internal/commands/chat/message_input.go`, multiple positional arguments cause `args[0]` to be silently discarded. |
| [#15](https://github.com/jonbaldie/prosie-cli/issues/15) | Parity gap: `generate summarize` lacks `--no-persist` flag and hardcodes `persist=true` | `GenerationSummaryResponder.php` validates `persist`, but the CLI client hardcodes `persist: true` and the command does not expose `--no-persist`. |

---

## 4. Usability Observations

1. **Negative numbers as arguments**: When passing negative integers as arguments, Go's flag parser treats them as flags unless the `--` delimiter is used. Adding positional argument indicators or handling negative values in `ParseFlagsAndArgs` would improve user experience.
2. **Series Codex parity**: The backend provides `/api/series/{series}/codex-entries`, `/api/series-codex-entries/{codexEntry}`, and append routes, but the CLI does not currently expose commands for series-level codex entries.
