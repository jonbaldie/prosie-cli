# Agent templates: A01 and A02

`DefaultPreserved` checks that a completed wizard retains the captured public default when the answer is empty. `NoSummaryOnEOF` checks that the HITL final capture does not report an observation after EOF. Both have source/model self-test mutants. These are bounded checks, not proofs of arbitrary generated scripts.

## Wizard model

`Wizard.tla` models the Stripe example: one stage, two independently initialized keys (absent, empty, or nonempty), and at most two invocations. Absence and empty map to the same prompt default. The banner pause can consume EOF. Later reads on that stream remain at EOF; an unterminated nonempty read is `partial`, returns failure, supplies bytes, and also sets EOF. A rerun after completion or abort resets run-local state and starts a fresh stream, while carrying destination values.

The write phases follow touch, mktemp, grep-copy, append, mv and written-array recording. An unhandled failure aborts. The temporary map and existence flag capture retained temporary effects, including a partial append; old temporary artifacts are not tracked across runs. File existence independent of key presence, arbitrary malformed lines, exact bytes and temporary filenames are abstracted. Failure at external `mv` permits unchanged, empty, or fully published destination: this is an overapproximate operation contract, not a claim about a particular OS or a proof of atomic rename across filesystems. Failures elsewhere are overapproximated by `Abort`.

`CopiesWork = TRUE` is the baseline successful-copy assumption. With `FALSE`, a suppressed grep error may yield an empty copy, after which append and mv still run. The original invariant then fails. This is a known source response to injected tool failure, not a deliberate mutant or evidence of a live incident. Reproduce:

```sh
python3 verification/state-machines/agent-templates/replay-copy-fault.py
python3 verification/state-machines/verify.py agent-templates
```

The first command runs only the real-source replay. The second also checks the baseline and requires the known fault. The replay requires `REPLAYED_WIZARD_SUPPRESSED_COPY_LOSS`; TLC must report `Invariant DefaultPreserved is violated`. The injected grep behaves normally for `_existing` and the first copy; its second copy exits 2 without output. Both saved defaults are displayed, the command succeeds, and the public key disappears from the actual temporary env file. No product source is changed and no real service is contacted.

The model tracks written env names, successful secret names and skipped secret/variable names. `SetVariable` is an optional reusable-library call, not part of the Stripe example. A successful variable is printed, not added to a `WRITTEN_VAR` array. Summary behavior is observed by `check.py`, not asserted by a separate model invariant. Browser and gh outcomes are external abstractions; no remote state, TTY behavior or durability is proved.

## HITL model and verification

`Hitl.tla` models each of the three read decisions and the actual `set -e` EOF abort. Partial lines and output failures are outside the bound. Human observations are not validated.

From the repository root:

```sh
python3 verification/state-machines/verify.py agent-templates
python3 verification/state-machines/verify.py agent-templates --self-test
```

Read results with /statechart. `Wizard-chart.cfg` is a selected scenario, not the unpruned baseline. `check.py` executes both source templates with temporary files and fake gh/browser. Self-tests intentionally mutate source and model in pairs and then rerun the baseline. The known grep finding is separate from those deliberate defects. Runner, TLC, Bash, stubs and OS are trusted infrastructure.
