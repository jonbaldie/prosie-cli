---
name: verify-generation-flows
description: Check generation cancellation and token finalization when changing continue, rewrite, summarize, or generation output.
---

# Verify generation flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A continuation cancellation attempt must use a fresh context. A dead context can prevent the cancel request from reaching the server. Stream finalization must not add a newline when the last token is empty or absent. An extra newline changes the output. JSON continuation must not stream; streaming changes the output protocol.

## Scope

Sync commit: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

Covered paths and functions:
- `internal/commands/generate/generate_continue.go`: `parseContinueArgs`, `useStreaming`, `executeGenerateContinue`, `streamContinue`, `continuationError`.
- `internal/commands/generate/generate_rewrite.go`: `executeGenerateRewrite`, `streamRewrite`.
- `internal/commands/generate/generate_summarize.go`: `executeGenerateSummarize`.
- `internal/commands/generate/output.go`: `newTokenOutput`.

From the repository root, inspect changes with:

```sh
git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/commands/generate/generate_continue.go internal/commands/generate/generate_rewrite.go internal/commands/generate/generate_summarize.go internal/commands/generate/output.go
python3 verification/state-machines/verify.py generation
```

For the proof run, use `python3 verification/state-machines/verify.py generation --self-test`. Read EVERY result with /statechart. The tests use a local server; no CLI runtime dependency is added.

The finite model derives mode and persistence from symbolic flags and tracks visited length classes, token and result classes, and cancellation selection. It does not establish actual argument parsing, exact token content, delivery, server cancellation, signal timing, or output-write success. The cancel call is best effort; no timeout duration is proved. Use `Generation-chart.cfg` for the bounded default continuation chart. Read `verification/state-machines/generation/README.md` for action mappings and evidence.

On failure, replay the path against the named source and tests. A model mismatch is not a product bug until reproduced against real code. For an intended source change, reconcile the model, tests and mutant pairs; make each defect fail the running real test and its named invariant without changing the invariant. Repeat the self-test and baseline. Move the sync commit only after confirming all covered behavior still matches. For stale verification, use /maintain-formal-verification-skill.
