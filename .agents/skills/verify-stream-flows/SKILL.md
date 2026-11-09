---
name: verify-stream-flows
description: Verify SSE frame dispatch and retained stream errors when changing stream reading or result handling.
---

# Verify stream flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A pending frame must be dispatched at EOF, or the last event can be lost. A done event must not clear a prior stream error. Return errors in context, read, stream, then missing-completion order; otherwise a failure can be hidden.

1. Check changes since sync commit `3dd1171b29bd4dccc8c4e995438f658f18b818b1`:

   ```sh
   git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/client/stream.go
   ```

   Coverage: `internal/client/stream.go`, functions `eventFrame.accept`, `eventFrame.dispatch`, `readEvents`, `streamTokens.accept`, `streamEvents.accept`, `streamSSE`, and `streamError`. See `verification/state-machines/streams/README.md` for the action mapping and which observed behaviors have no invariant.

2. From the repository root run `python3 verification/state-machines/verify.py streams`. Read EVERY result with `/statechart` by name, using the supplied view. For defect evidence run `python3 verification/state-machines/verify.py streams --self-test`. These tools add no CLI runtime dependency.
3. On failure, compare the shortest path with actual source order and replay it through the Go tests. A model counterexample is not itself proof that Go fails. A test failure can be an intended contract change, a defect, or stale evidence; establish which before changing a rule.
4. After an intended change, reconcile each mapped action, invariant, test and matching real-source and model mutation. Run the self-test and baseline again. Move the sync commit only after confirming that the model and tests describe the new source. Use `/maintain-formal-verification-skill` for stale verification.

The frame model bounds input to three symbolic lines; the result model bounds input to three events. They assume source-order callbacks and abstract parsing. Use `Frame-chart.cfg` for a two-line chart; the three-line baseline exceeds 500 states. The precedence mutant exercises context versus read error, not each ordering pair. They do not prove network behavior, liveness, arbitrary input sizes, or server persistence. Done and error do not stop reading. A non-EOF read error does not flush the frame.
