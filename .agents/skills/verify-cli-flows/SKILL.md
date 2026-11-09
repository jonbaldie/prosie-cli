---
name: verify-cli-flows
description: Verify CLI dispatch, argument scanning, output error latching, deletion approval, and content or patch input after their source changes.
---

# Verify CLI flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

Check five bounded protocol models and actual Go code. The rules protect copied JSON mode, argument value consumption, the first output failure, deletion approval, and stdin before source-conflict rejection. A defect can change an unrelated command's format, lose a flag value, hide an output failure, delete without approval, or consume input on a rejected command.

Source sync commit: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

Coverage:

- S01: `internal/command/dispatch.go:Dispatch`; `internal/cmd/root.go:RootCmd.Execute`.
- S02: `internal/command/args.go:splitArgs`, `flagNeedsValue`, `ParseFlagsAndArgs`, `Args.Validate`; `internal/command/input.go:VisitedFlags`, `Provided`.
- S03: `internal/cmd/root.go:outputWriter.Write`, `RootCmd.Execute`.
- S06: `internal/command/confirmation.go:ConfirmDeletion`; `internal/command/delete.go:ExecuteDelete`.
- S07: `internal/command/content.go:ReadContent`; `internal/command/input.go:ChapterInput`; `internal/commands/chapter/input.go:params`; acquisition and validation in `internal/commands/generate/generate_rewrite.go`.

1. Compare covered source with the sync commit. Read `verification/state-machines/cli/README.md` and its mapping files. Keep operation order, retained state and failure paths consistent with the source.
2. Run `python3 verification/state-machines/verify.py cli`. Run `python3 verification/state-machines/verify.py cli --self-test` to require rejection of five isolated source/model defect pairs and a repeated baseline. These checks add no CLI runtime dependency.
3. Read EVERY TLA+ result with `/statechart`, using the manifest configs and matching views. Replay each unexpected path against the actual Go helper or command before calling it a product defect. Preserve the distinction between callback success and server state.
4. For an intentional change, update the model, real-code tests, paired defects, view and mapping together. Re-run the self-test, full Go tests and `make messgo`. Move the sync commit forward only after the mapping is reviewed. Use `/maintain-formal-verification-skill` for intervening changes without checks.

Bounds: two dispatch levels and root calls, a finite argument alphabet with at most three tokens, at most two output attempts, and symbolic content/read outcomes. The Go standard library parser, arbitrary data, backend effects and a composed proof of the whole CLI are outside the proof. Presence, result precedence and patch/validation branches are modeled observations unless the README names a checked rule. Successful I/O and stable external behavior are not implied by a model pass.
