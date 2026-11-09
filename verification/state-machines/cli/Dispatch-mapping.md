# Dispatch model: S01

Source revision: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

`internal/command/dispatch.go` copies the caller environment, consumes only leading exact `--json` tokens, then returns help, unknown, or calls a registered handler. `Start`, `ConsumeJSON`, and `Choose` represent these steps. `parent` is the caller's mode, `snapshot` its value at entry, `mode` the dispatch copy, and `inherited` the mode at entry. Entering the group creates a second copy. `sibling` observes the caller's mode when the child returns. The invariant `ParentCopyIsolation` requires the caller and subsequent sibling to retain the entry mode. A violation can make an unrelated command use JSON.

`ScanLeaf` represents the `--json` bool flag and `ParseFlagsAndArgs` in `internal/command/args.go`: it scans flags even after a positional, treats all tokens after `--` as positional, and applies `--json=false` to the leaf flag rather than its caller. The model's representative leaf accepts at most one positional. An unknown flag is a parse error. Help/unknown returns and the leaf's text/JSON selection are separate outcomes. Formatting, full flag syntax, flag values, and other command-specific validation are outside this model. The separate S02 model covers the wider scanner. The finite `Inputs` set enumerates the listed leaf tails with zero to two leading JSON flags at each of two dispatch levels; it is not every possible argument sequence.

`Output` abstracts a successful or failed stdout write; `Start` resets the latch on each of two root invocations, as does `RootCmd.Execute` in `internal/cmd/root.go`. It does not model the full output latch protocol; S03 does. The Go root test checks the actual exit code and output after a failed first write, and checks the inherited JSON override. The command test calls real nested `Dispatch` and `ParseFlagsAndArgs`, observes the caller and sibling, checks the separator, help, and unknown cases. All fixtures are local.

The paired scratch defect additionally writes the consumed JSON flag into the caller. TLC then reaches `parent = TRUE` while `snapshot = FALSE`; the actual command test fails because the nested caller was changed. The temporary Go overlay and mutated TLA file do not alter product source. The invariant does not claim that every possible inheritance or parser regression is detected.

Run `python3 verification/state-machines/verify.py cli --self-test` from the repository root. The CLI manifest contains this model and its source/model defect pair. Read TLC results with `/statechart`.
