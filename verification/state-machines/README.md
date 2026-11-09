# State-machine verification

This directory models the 28 recorded protocols: 21 runtime families, five tooling flows, and two agent templates. There is one owning project-local skill per subsystem. Repeated endpoint wrappers use shared protocols; they are not separate local machines. The source inventory was made at `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

## Run

From the repository root:

```sh
python3 verification/state-machines/verify.py all
python3 verification/state-machines/verify.py all --self-test
python3 -m unittest discover -s verification/state-machines -p test_verify.py -v
```

Use a subsystem name instead of `all` for a focused change. The first command checks actual code and bounded TLA+ models. The second must also catch matching defects in scratch production sources and models, remove the scratch changes, then pass again. A failed build, missing success marker, or unexpected failure is an error, not evidence that a mutant was caught. The third command tests fail-closed behavior in the new runner.

Prerequisites: Python 3, Go, Java 11 or later, and curl. Full self-tests require Linux or macOS for POSIX file and shell checks. The runner selects Go 1.26.3 and reuses the SHA-256-pinned TLC release 1.7.4 downloader from `verification/config-save/verify.py`. Initial tool downloads require network access. Checked commands have a five-minute timeout. These are development tools, not CLI runtime dependencies.

## Known finding: wizard copy failure

The current wizard can lose a saved key after a suppressed grep-copy error. In the controlled replay, both saved defaults are displayed, the second copy exits 2 without bytes, and the script succeeds with only the secret key left in the env file. This is a reproduced response to an injected tool failure, not a live incident. Product code is unchanged.

The wizard baseline explicitly assumes successful copies. The separate `WizardIOFault.cfg` checks the unchanged `DefaultPreserved` rule without that assumption and finds a counterexample. The normal agent-template command requires both the actual-source replay marker and that exact model violation. Command success means the expected checks ran; it does not mean this defect is fixed. See [the replay and assumptions](agent-templates/README.md).

## Ownership and skills

`coverage.json` is the machine-ID ownership record. Each subsystem manifest must contain exactly its assigned IDs. Each declared model invariant must have a matching source/model failure demonstration. Source files and functions, model actions, individual bounds, and exact test evidence are in each subsystem README and skill.

| Subsystem | Inventory IDs | Project-local skill | Model and source mapping |
| --- | --- | --- | --- |
| CLI input, dispatch and output | S01, S02, S03, S06, S07 | `verify-cli-flows` | [cli](cli/README.md) |
| Authentication | S04, S05 | `verify-auth-flows` | [auth](auth/README.md) |
| Import and export | S08, S09 | `verify-transfer-flows` | [transfers](transfers/README.md) |
| Conversation turns and rendering | S10, S11 | `verify-chat-flows` | [chat](chat/README.md) |
| Generation and cancellation | S12, S13 | `verify-generation-flows` | [generation](generation/README.md) |
| SSE parsing and completion | S14, S15 | `verify-stream-flows` | [streams](streams/README.md) |
| HTTP ownership and hydration | S16, S17 | `verify-http-flows` | [http](http/README.md) |
| Private credential publication | S18 | `verify-config-save` | [credential-store](credential-store/README.md) |
| Link traversal and export publication | S19, S20 | `verify-file-storage` | [file-storage](file-storage/README.md) |
| Version, release and CI | S21, T01, T02, T03 | `verify-release-flows` | [release](release/README.md) |
| Verification tools | T04, T05 | `verify-checker-flows` | [verification](verification/README.md) |
| Executable human-step templates | A01, A02 | `verify-agent-template-flows` | [agent-templates](agent-templates/README.md) |

The private credential model remains the original two-writer protocol. Its runner is delegated, not replaced. Other models cover their documented bounded calls and event classes. Separate model passes do not establish a composed proof of the whole CLI.

## Why TLA+ and real-code checks

The main questions concern order, retained values, cancellation, partial success, and ownership. TLC can explore all reachable states in each finite model. Go tests and controlled shell/Python runs check the actual source at that boundary. Scratch mutants show that the two layers can reject matching implementation defects. Generated or enumerated tests remain bounded tests; they are not a proof of arbitrary Go execution.

Some workflow checks establish a source contract, such as an explicit dependency. They do not execute GitHub's scheduler. External-action behavior is an explicit model assumption. No test uses a live account, sends real secrets, publishes a release, or starts a remote workflow.

## Read and draw results

Read every TLA+ result with `/statechart`. Use each manifest's model/config and the corresponding view named in its subsystem README. A compact chart configuration, where supplied, selects an important complete flow within the full model's declared scope. It is not a replacement for the default baseline check. Read counterexamples as operations in the actual source and replay them before calling them product defects.

The important user flows are device/token login, confirmation before deletion, create-and-send recovery, generation cancellation, stream completion, file export, and private credential publication. The task's statechart reports include Mermaid text and Pele-rendered SVGs. Statechart execution and rendering stay outside the repository; follow the installed `/statechart` skill rather than copying its current commands into these skills or scripts.

## Assumptions and non-guarantees

- A pass covers the stated model, bounds, input classes, and assumptions. It is not a proof of arbitrary source execution, backend correctness, or external scheduler behavior.
- Backend persistence, token grants/revocation, retry/undo eligibility, and conflict resolution are server-owned observations. A later read or output error does not undo a successful remote action.
- A stream's done/error event is not EOF. Later events and final return precedence matter. No eventual-EOF or fairness guarantee is implied.
- Logout does not clear environment tokens. Device login stores its token before the optional profile lookup. These are modeled source behaviors, not guarantees of revocation or profile verification.
- Filesystem atomic rename, mode semantics and stable paths are explicit assumptions. No model establishes hostile-path safety, Windows ACL privacy, or directory-sync power-loss durability.
- Local release can leave partial artifacts. Relative output paths can change meaning in the Windows packaging step. The shell's Git-status gate differs from the Go version initializer.
- Models and test harnesses are trusted descriptions that require review after source changes. The sync commit is a review boundary, not a cryptographic proof of fidelity.

## Maintain

Use the owning skill when its covered source changes. Compare the source with the recorded sync commit; reconcile every changed operation and error path; update the model, real-code checks, source/model defect pairs, and view together. Run the focused self-test, all baselines, full Go tests, and `make messgo`. Move the sync commit forward only after the mapping is confirmed. Use `/maintain-formal-verification-skill` when intervening changes did not recheck the models.
