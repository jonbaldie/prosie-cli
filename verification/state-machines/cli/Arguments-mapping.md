# S02: argument scanning and presence

Source checked at `3dd1171b29bd4dccc8c4e995438f658f18b818b1`: `internal/command/args.go` (`splitArgs`, `flagNeedsValue`, `ParseFlagsAndArgs`, `Args.Validate`) and `internal/command/input.go` (`VisitedFlags`, `Provided`).

`input` is a sequence of at most three tokens. The alphabet has a positional value, `-`, `--`, a string flag with or without an explicit empty value, a boolean flag with and without explicit false, an unknown flag, and help. `ScanStep` follows the scan index, keeps flag and positional lists separately, consumes the next token for a value flag even if it is `--`, and records a missing final value. `FinishScan` adds the parser boundary only when the positional list is nonempty and the value is not missing. `Scan` is an independent recursive description of the expected complete split. `ScannerRetention` compares all three outputs: a flag value must not be lost or scanned again, and positionals must stay in order.

`ParseStep` represents Go's `flag` package as a finite external parser for this alphabet. It tracks visited flags and values, help, unknown-flag errors, missing-value errors, and positional count validation with a maximum of one. It does not model arbitrary flags, malformed boolean values, aliases, parser diagnostics, or flag-set reuse. The invariant checks the scanner, not the external parser or presence behavior. The Go tests observe real parsing and presence for explicit false and empty values. The Go value-consumption test uses four tokens to observe both consumption of `--` and a following flag; the model bound is three.

The paired defect leaves the index at the value-taking flag and appends that flag again instead of its value. The model defect makes the corresponding scanner step. TLC rejects `ScannerRetention`; the real `TestFormalArgumentsValueConsumption` fails with the Go overlay. No production file is changed by the test.

Run `python3 verification/state-machines/verify.py cli --self-test` from the repository root. Read the TLC result with `/statechart`. These checks establish behavior only for the stated bounded model and concrete Go test inputs.
