---
name: verify-transfer-flows
description: Verify upload and export lifecycle changes, including multipart input errors and file export failures.
---

# Verify transfer flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A failed upload copy must not produce a request, and an opened upload file must close. A failed file replacement must not return success. Chat file output must not be rejected by stdout JSON decoding.

Coverage: `internal/client/upload.go` (`UploadFile.postRequest`, `encode`); `internal/command/upload.go` (`OpenUpload`); `internal/command/export.go` (`WriteExportFile`, `ExportProse`); book and chat import handlers and book, chapter and chat export handlers under `internal/commands/`. The assertions cover copy failure, upload file cleanup, file replacement failure, and chat file-versus-stdout routing, not every behavior in those paths. See `verification/state-machines/transfers/README.md` for mappings and limitations.

Last synced to `3dd1171b29bd4dccc8c4e995438f658f18b818b1`. From the repository root inspect changes with:

```sh
git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/client/upload.go internal/command/upload.go internal/command/export.go internal/commands/book/book_import.go internal/commands/chat/chat_import.go internal/commands/book/book_export.go internal/commands/chapter/chapter_export.go internal/commands/chat/chat_export.go
python3 verification/state-machines/verify.py transfers
python3 verification/state-machines/verify.py transfers --self-test
```

Read EVERY result with /statechart. The models bound each run to one transfer and split encoding into field, part, copy, and close steps. Filesystem replacement is atomic in the model. Local tests do not prove server behavior, rollback, durability, concurrency, exact output bytes for all formats, or all failure paths. No CLI runtime dependency is added.

On failure, inspect the actual source, test output, and model path. Decide whether the code, test, or model is wrong before changing an invariant. The self-test must catch each real-source overlay and corresponding model mutant, then pass the unchanged baseline. If source anchors or behavior changed, reconcile actions, tests, mutants, and the coverage record. Move the sync commit only after confirming all mappings at the new commit. Use /maintain-formal-verification-skill for stale verification.
