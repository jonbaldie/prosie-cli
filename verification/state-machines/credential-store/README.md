# Private credential storage: S18

This subsystem reuses the checked two-writer model, real-file failure matrix, source mutants, and project-local `verify-config-save` skill in `verification/config-save/`. Read `verification/config-save/README.md` for each action's source mapping, all four invariants, assumptions, bounds, and counterexample replays.

From the repository root:

```sh
python3 verification/state-machines/verify.py credential-store
python3 verification/state-machines/verify.py credential-store --self-test
```

The common runner delegates to the original verifier. It does not substitute another filesystem model. Read every result with `/statechart`. Use the existing SingleSave configuration and `verification/config-save/view.mjs` for a compact user flow. Use this directory's `view.mjs` for the full two-writer result and its defect paths. It shows both saves and separates all four failure conditions, including the publish-before-write defect phase. The default two-writer check covers all interleavings, but can exceed the chart's 500-state budget. The original model, verifier, and single-save view are unchanged.

`Save`, `replaceFile`, `writeAndPublish`, and their file hooks in `internal/config/config.go` are unchanged between the original sync commit and `3dd1171b29bd4dccc8c4e995438f658f18b818b1`. Link traversal is now modeled separately by the `file-storage` subsystem; the credential model still assumes a stable resolved destination. Their separate passes are not a composed proof against hostile filesystem changes.
