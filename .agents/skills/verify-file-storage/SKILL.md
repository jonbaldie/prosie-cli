---
name: verify-file-storage
description: Check bounded symbolic-link traversal and export-file replacement when changing file storage or config link resolution.
---

# Verify file storage

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A relative link must resolve from its own directory. Otherwise a write can reach the wrong path. A successful replacement of a regular file must retain its permission bits. Otherwise the export can become inaccessible or too widely readable. Regular or missing targets must not be published before a complete nonfailed write, successful sync and successful close. Before rename, failure must leave the old bytes in place.

Coverage at sync commit `3dd1171b29bd4dccc8c4e995438f658f18b818b1`:
- `internal/config/config.go`: `resolveLinks`.
- `internal/filewrite/replace.go`: `Replace`, `resolveLinks`, `replaceRegular`, `writeAndSync`.

1. Compare the covered source with the sync commit:
   `git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/config/config.go internal/filewrite/replace.go`
2. From the repository root run `python3 verification/state-machines/verify.py file-storage`. For the scratch defects and post-mutation baseline, run `python3 verification/state-machines/verify.py file-storage --self-test`. These checks add no CLI runtime dependency.
3. Read EVERY TLA+ result with /statechart by name: `Links.tla` and `Replace.tla`, their matching cfg files, and `view.mjs` in `verification/state-machines/file-storage/`. Read `README.md` there for the action mapping, test chain, assumptions and edge cases.
4. On failure, replay the path against the source and actual test. Decide whether the implementation or the model is wrong; do not weaken a property to hide a defect. A passing bounded model does not prove the Go program. Bounds are 40 link iterations and one replacement. Use `Links-chart.cfg` for the smaller chart. Link names and path transformation are symbolic; this is not a proof for arbitrary paths. Filesystem calls and rename semantics are assumptions; concurrent path changes and crash durability are outside scope. Cleanup failure is modeled but cleanup is not guaranteed. The publication source mutant uses the existing Linux/Darwin file-size-limit subprocess test; its required PASS/FAIL markers prevent a skipped or absent test from counting. Other observed branches have no asserted guarantee.
5. After an intended source change, reconcile every affected action, test and source/model mutant. Run the self-test and baseline again. Move the sync commit only after confirming that all covered source still matches. Use /maintain-formal-verification-skill when verification is stale.
