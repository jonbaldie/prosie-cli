# CLI input, dispatch, output and deletion

Source sync commit: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`. Inventory: S01, S02, S03, S06, S07. These are bounded protocol models, not proofs of arbitrary Go execution.

| ID | Model | Retained state and checked rule |
| --- | --- | --- |
| S01 | `Dispatch.tla` | Nested copied environments, inherited and leaf JSON mode, caller/sibling isolation, help/unknown/leaf returns, and reset across two root calls. `ParentCopyIsolation` protects the caller's mode. See [mapping](Dispatch-mapping.md). |
| S02 | `Arguments.tla` | Scan index, flag and positional lists, value consumption, separator, parser outcome, values and visited flags. `ScannerRetention` checks the split against a recursive reference. See [mapping](Arguments-mapping.md). |
| S03 | `Output.tla` | Zero to two attempts, underlying write count, first error/short-write latch, and handler/final result. `NoWriteAfterFailure` prevents another underlying write after failure. Dispatch also models per-execution reset. |
| S06 | `Cli.tla` | Validation, yes flag or first-line answer/EOF/scan error, approval, client setup, callback invocation/result, reporting and terminal state. `DeletionRequiresApproval` protects invocation even when the callback fails. |
| S07 | `Content.tla` | Inline/path presence, file or stdin acquisition, attempted consumption, read errors, omitted/empty patch field and rewrite validation. `NoConflictConsumption` prevents stdin consumption on a content-source conflict. See [mapping](Content-mapping.md). |

## Source mapping

S01 covers `internal/command/dispatch.go:Dispatch` and the orchestration boundary in `internal/cmd/root.go:RootCmd.Execute`. S02 covers `internal/command/args.go:splitArgs`, `flagNeedsValue`, `ParseFlagsAndArgs`, `Args.Validate`, and presence helpers in `internal/command/input.go`. The representative leaf accepts one positional and a JSON flag. It is not every command's flag set.

S03 maps `Write`, `Return` and `Done` to `internal/cmd/root.go:outputWriter.Write` and the final result selection in `RootCmd.Execute`. Short-write and returned-error classes differ. The current registered handlers return 0 or 1. A nonzero handler result is retained; an output error changes a zero result to 1. Return selection is modeled but not a separately mutation-backed invariant. The real reset test executes the root twice; the latch test calls the actual writer and checks its returned error and underlying call count.

S06 maps `ValidateDelete` to parsing and ID validation in the callers; `Confirm` to `internal/command/confirmation.go:ConfirmDeletion`; and `Authenticate`, `Delete`, `ReportDelete` to `internal/command/delete.go:ExecuteDelete`. Validation outcomes are input classes. Confirmation normalizes the first scanned answer; empty/refusal, EOF and scan error do not approve. `--yes` skips the prompt. `invoked` records the local callback call. `deleted` records only its nil error, **not** independently observed server deletion. A callback error may follow a server effect. A later report failure does not roll back the callback.

S07 covers `internal/command/content.go:ReadContent`, `internal/command/input.go:ChapterInput`, `internal/commands/chapter/input.go:params`, and acquisition/validation in `internal/commands/generate/generate_rewrite.go`. Data classes are empty, whitespace-only or nonblank; path classes are empty, file or stdin. Patch omission is not the same as an explicit empty field. JSON field encoding and text normalization are transformations observed at the boundary, not extra machines.

## Run and reject defects

```sh
python3 verification/state-machines/verify.py cli
python3 verification/state-machines/verify.py cli --self-test
```

The five scratch defects mutate the actual source: leak JSON into the parent, fail to consume a flag value, bypass the output latch, bypass confirmation, and read stdin despite conflicting inline input. Matching model defects must violate their named rules. The runner requires running-test failure markers, then repeats the unchanged checks. All file and reader fixtures are local; no live service is used.

Read every result with `/statechart`. Each manifest entry names its config; the views are `Dispatch-view.mjs`, `Arguments-view.mjs`, `output-view.mjs`, `view.mjs`, and `Content-view.mjs`, respectively. Deletion is the compact user flow. Large default graphs still need their full rule reports; a compact chart does not replace those checks.

## Assumptions and limits

Two dispatch levels and two root executions, at most three tokens from a finite alphabet, and at most two output attempts bound the models. The scanner invariant does not prove the Go standard library parser. Presence, rewrite validation and patch JSON have actual-code checks but no separate model invariants. The content model records attempted stdin consumption, not exact partial bytes. The output model does not include arbitrary writer implementations that violate the `io.Writer` contract.

Local synchronous helper calls can be atomic model steps when their internal assignments have no competing observer. I/O failures are outcomes, not claims that I/O itself is indivisible. Authentication, HTTP, file publication and remote persistence use the owning subsystem boundaries. There is no composed proof of the full CLI, remote rollback, arbitrary text, concurrency or eventual completion.

Use the owning project-local skill after covered source changes. Update the source mapping, model, actual-code checks, paired defects and view together; advance the sync commit only after the checks and review pass. Use `/maintain-formal-verification-skill` for stale verification.
