---
name: verify-checker-flows
description: Verify checker cache validation and scratch-mutant exit handling when changing the config-save verifier or its verification protocol.
---

# Verify checker flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A returned checker must match its expected digest. A corrupt checker can make a verification result untrustworthy. An accepted scratch defect must exit nonzero and print its expected marker. Otherwise an unrelated failure or a successful test can be mistaken for a caught defect.

Coverage: `verification/config-save/verify.py` functions `checker`, `run`, `model`, `replace_once`, `self_test`, and `main`. The model represents their ordered lifecycle and four defect iterations, but does not prove their internals. Details and exclusions are in `verification/state-machines/verification/README.md`. The sync commit is `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

1. Inspect changes with `git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- verification/config-save/verify.py verification/state-machines/verification .agents/skills/verify-checker-flows`.
2. From the root run `python3 verification/state-machines/verify.py verification`. For the scratch defect proof run `python3 verification/state-machines/verify.py verification --self-test`.
3. Read EVERY result with /statechart. A model failure is a bounded counterexample, not automatically a product bug. Replay it against the source and the controlled process evidence. A setup error is not a caught defect. The common harness is trusted infrastructure, not covered by this model.
4. For changed behavior, reconcile each action and property with the source. Check each scratch source mutation and model mutation still fails for the same reason, then repeat the unchanged baseline. Update the model, tests, manifest, and coverage notes together. Advance the sync commit only after that review. Use /maintain-formal-verification-skill for stale verification.

The finite model uses one cache attempt and four defect iterations. It assumes SHA-256, subprocess isolation and file operations behave as specified; it does not cover concurrent cache changes or S18 itself. The controlled actual main execution uses stub Go and Java processes. The checks add no CLI runtime dependency.
