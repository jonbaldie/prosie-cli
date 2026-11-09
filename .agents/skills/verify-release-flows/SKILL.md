---
name: verify-release-flows
description: Check release artifact cleanup, version selection, and release or CI workflow changes against bounded models and source checks.
---

# Verify release flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

The local release script must remove old release archives before it starts a build, or stale files can mix with new artifacts. The version helper must accept a tag only after a successful clean status check, or a dirty checkout can report a released version. The release join depends on all five matrix jobs. Its source-contract check is not a scheduler proof.

Coverage: `scripts/build-release.sh` version selection, cleanup, build loop and checksums; `internal/version/version.go` functions `init`, `repositoryDirs`, `exactTagVersion`; `.github/workflows/release.yml` build and release jobs; `.github/workflows/ci.yml` test job. Source sync commit: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

1. From the repository root, inspect changes with `git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- scripts/build-release.sh internal/version/version.go .github/workflows/release.yml .github/workflows/ci.yml`.
2. Run `python3 verification/state-machines/verify.py release`. For the scratch-defect proof run, add `--self-test`. These tools add no CLI runtime dependency. Read EVERY result with /statechart.
3. On failure, inspect the source, real test output, and model path in `verification/state-machines/release/README.md`. Decide whether the source has a defect or the model/test no longer matches the source. Workflow checks inspect text only, not GitHub execution.
4. Reconcile each changed behavior with the action mapping and bounds, then update model, evidence and defect together. Confirm each scratch source defect reaches the expected running assertion and each matching model defect breaks its invariant. Rerun the proof and baseline before moving the sync commit. For stale verification, use /maintain-formal-verification-skill.

The model bounds the release to five targets and CI to five steps. It retains symbolic partial outputs and shell version-source classes. It assumes GitHub default dependency and matrix fail-fast semantics. The real shell test uses stub tools in scratch, never a release matrix. Source-contract checks do not prove scheduler execution. See the README for the remaining control-flow gaps, abstractions and limits.
