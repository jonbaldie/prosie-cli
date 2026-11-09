---
name: verify-deletion-confirmation
description: Check the deletion confirmation guard when changing shared deletion code or its confirmation input handling.
---

# Check deletion confirmation

Without `--yes`, a delete action must run only when the first input line is `y` or `yes`, ignoring case and surrounding whitespace. An accidental delete can remove a user's work.

## Run the check

From the repository root, run `go test ./internal/command -run '^TestDeletionConfirmationBounded$' -count=1` with Go 1.26.3. This is the version used to confirm the check. The repository's `go.mod` sets its minimum Go version.

The test runs `ExecuteDelete` and counts calls to the supplied delete action. It checks both values of the `--yes` option for every ASCII byte string of length zero, one, and two. It also checks the longer inputs in the test. These are generated tests of Go code, not a formal proof. They do not cover every input, Unicode, network requests, or individual CLI command wiring.

## Review changes

The coverage record is `internal/command/confirmation.go:ConfirmDeletion` and `internal/command/delete.go:ExecuteDelete`. The last commit at which the check was compared with those functions is `54e27e8fabf9e19957fdda1e7b5bd0acd621fe9e`. Review changes with `git diff 54e27e8fabf9e19957fdda1e7b5bd0acd621fe9e -- internal/command/confirmation.go internal/command/delete.go`.

If the test fails, read the input, the `yes` value, and the delete call count in the failure. Compare that result with the intended user-facing rule and the current code. If the rule is unchanged, fix the code rather than weakening the test. If the rule changed intentionally, update the test and this skill. Move the recorded commit forward only after you confirm that the check describes the code at that commit. Run the check again.

This check does not establish that every delete command calls `ExecuteDelete`. Inspect command wiring separately when it changes.
