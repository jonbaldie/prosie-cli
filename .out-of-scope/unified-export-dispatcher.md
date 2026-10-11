# Unified export dispatcher

The CLI does not have one export dispatcher that handles prose exports and structured JSON exports through a single function with a format switch (for example `DispatchExport(c, ExportPayload{IsJSON: ...}, ...)`).

## Why this is out of scope

The export seam in `internal/command/export.go` already shares the part that is the same for all exports. `WriteExportFile` writes the file and prints the file result for `book export`, `chapter export`, and `chat export`. `ExportProse` adds the stdout behavior for prose.

Only the stdout path is different for `chat export`. The server returns a JSON document, so `--json` must decode it and print it as JSON, and plain output prints it with a final newline. The prose exports wrap the text in `{"id", "content", "bytes"}`. These are two different output contracts, not two copies of one contract.

A dispatcher with an `IsJSON` field puts both contracts behind one function that has one caller for each branch. It moves about fifteen lines and adds a type and a flag. It does not remove a rule that is repeated. The regression that started the request (#67: `chat export --json` printed non-JSON text with exit code 0) is fixed in `chat_export.go` and has a test.

Change this decision if a third export with a different output contract is added. Then a shared dispatcher has more than one caller for each branch.

## Prior requests

- #78: "Encapsulate document export and upload streaming across command and client modules" (the export part; the upload part was already done in #76)
