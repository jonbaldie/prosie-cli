---
name: verify-config-save
description: Verify credential replacement when changing config.Save, replaceFile, writeAndPublish, their file hooks, or the order of create, write, sync, close, and rename.
---

# Verify credential replacement

A save must publish only complete private credentials after successful write, sync, and close operations, and must not publish after an I/O failure. A broken replacement can expose credentials or destroy the previous configuration.

## Check a change

1. Read `verification/config-save/README.md` for the code mapping, assumptions, and limits.
2. Compare the covered code with the sync commit below. Confirm that the model still describes each operation and error path.
3. From the repository root, run `python3 verification/config-save/verify.py`. The command checks the real Go failure matrix and the two-writer TLA+ model. A nonzero exit is a failure, not a pass with warnings.
4. Read every TLA+ result with `/statechart`. For a failure, explain its shortest path in the code's words. For a pass, use the single-save configuration in `verification/config-save/SingleSave.cfg` to draw the flow. The full two-save model can exceed the chart's state budget.
5. Before changing a property or the verification, run `python3 verification/config-save/verify.py --self-test`. This must catch each isolated Go and model defect and pass again on the unchanged sources.

Use Python 3, curl, Java 11 or later, and Go. Run the permission self-test on Linux or macOS. The runner selects Go 1.26.3 and checksum-pins TLC from TLA+ release 1.7.4. The first run needs network access for TLC and, if needed, the Go toolchain download. No tool is added to the CLI's runtime dependencies.

## Read a failure

A TLC counterexample is a finding in the model, not automatically a product bug. Replay the same operations against `Save` with the existing file hooks. If the Go behavior differs, fix the model or its assumptions. If Go has the same defect, report the finding and preserve the counterexample. Change product code only when the task permits it. Keep the intended property unchanged.

The Go matrix uses real temporary files and all combinations of five injected failure points. Its operation log must agree with the model's call order. It does not test concurrent operating-system execution. A pass from either layer alone does not establish the combined claim.

## Keep the model in step

The sync commit is `54e27e8fabf9e19957fdda1e7b5bd0acd621fe9e`.

The covered file is `internal/config/config.go`. The covered functions are `Save`, `replaceFile`, and `writeAndPublish`. The covered hooks are `tempFile`, `createTemp`, and `rename`. `resolveLinks` is a boundary assumption, not a verified function.

Run `git diff 54e27e8fabf9e19957fdda1e7b5bd0acd621fe9e -- internal/config/config.go`. Review callers separately if they change the destination or share file hooks. Update the model, matrix, and self-test together when an intended design change alters an operation or its error path. Move the sync commit forward only after confirming that the verification describes the code at that commit. Rerun both commands after the update.

Use `/maintain-formal-verification-skill` when intervening changes did not recheck this verification.
