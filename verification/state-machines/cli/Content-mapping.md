# S07: content acquisition and patch construction

Source revision: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

`Select` models `command.ReadContent` and the `ChapterInput` file-presence gate. A rewrite always calls `ReadContent`; a chapter patch calls it only for a visited nonempty file path. `Read` models `io.ReadAll` or `os.ReadFile`; `consumed` records an attempted stdin read, not a byte count. A read error is separate from the conflict and rewrite validation errors. File bytes and failures are environment inputs, not filesystem state. `Build` models `ChapterInput` and the resulting content field: `absent` means the JSON key is omitted; the other classes mean it is present. `Validate` models `rewriteSelection` and `rewriteInput`, including selection-first validation and nonempty prompt precedence even for whitespace.

The finite text classes are empty, whitespace-only, and nonblank; paths are empty, stdin, or file. Presence and content are consistent with flag defaults. Reads either fail or return one text class. This does not model partial byte consumption, file handles, arbitrary text, flags other than content/file, authentication, HTTP, or request delivery. Each transition abstracts a synchronous helper operation. File/stdin read effects are not interchangeable.

**NoConflictConsumption:** a nonempty inline value and nonempty path must not lead to a stdin read when the flow invokes `ReadContent`. A violation can consume data before the caller sees the conflict. The paired scratch defect removes that guard in Go and the model: a conflicting stdin path reaches `Read`, and the real test detects the unexpected read and missing error. The test uses a counting reader. The other tests check actual patch presence/JSON and rewrite validation but are not separately mutation-proved invariants.

Run `python3 verification/state-machines/verify.py cli --self-test` from the repository root. The CLI manifest contains the model and its source/model defect pair. Read TLC results with `/statechart`.
