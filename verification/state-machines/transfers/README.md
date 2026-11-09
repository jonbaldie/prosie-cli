# Transfer models

S08 and S09 use one transfer per run. The models are bounded abstractions, not proofs of the Go implementation or the remote server. The invariants below have paired source and model defects; other states are observations, not verified guarantees.

## S08: upload

`internal/command/upload.go` `OpenUpload` and book/chat import handlers map to `Validate`, `Open`, `OpenError`, `Auth`, and `Cleanup`. The command owns a file and closes it on all later exits; stdin is not closed. `internal/client/upload.go` `encode` maps to `Fields`, `Part`, `CopyOK`/`CopyError`, and `CloseMultipart`. The multipart close succeeds for the current bytes.Buffer. `Build`, `Send`, `Response`, `Read`, `Decode`, and `Report` represent request construction, transport, response read/decode and output. Cleanup ignores close errors. Copy precedes request construction, so input can be consumed even with a cancelled context.

`NoBadRequest` says a request requires complete encoding. `FileClosed` says an opened file is closed on return. `TestFormalTransfersUpload` tests the actual request builder with good and failing readers. `TestFormalTransfersOwnership` tests actual file and stdin selection and cleanup. The paired defects return a partial multipart body on copy failure or omit file close. Neither test proves the complete command path; the ownership test exercises the close function directly.

Input selection is one file or stdin, with one attempt and no retries. Read failure consumption is represented as a boolean, not an exact byte count. Multipart field/part errors are abstract failure choices; the current bytes.Buffer cannot itself fail writes. Auth, transport, read and decode failures converge on cleanup because their remaining local resource behavior is the same. Remote changes are not rolled back on later failure. The model does not track default filenames, HTTP status separately from other transport failures, or partial server state.

## S09: export

Book, chapter and chat export handlers validate, authenticate and fetch bytes before delivery. `FetchOK` retains independent JSON-validity and final-newline classes: valid or malformed, each with or without a final newline; `FetchError` ends before routing. `Route` selects file or stdout. `WriteOK` represents successful `internal/command/export.go` `WriteExportFile` publication through `filewrite.Replace`; `WriteError` is failure. `Render` distinguishes raw prose, JSON prose, base64 DOCX, chat JSON, and chat text with or without an added newline. Malformed chat JSON fails only on stdout JSON. `Report` follows file publication and may fail at output; the root output latch then changes the final exit status without undoing the file.

`SuccessNeedsPublication` applies to the handler's file-mode success, not stdout. `FileRaw` says a file route does not enter the stdout JSON decode-error state. `TestFormalTransfersExport` invokes the real file writer on an invalid path. `TestFormalTransfersRouting` executes the root command against a local HTTP fixture and compares malformed JSON file and stdout routes. It also checks all four validity/newline classes in text output. Their paired defects return success on write failure or incorrectly route JSON file output through decoding. `TestFormalTransfersFetchAndReport` executes the root command, observes that the destination is absent during the fetch, then checks that a failed output report leaves the published file in place and changes the exit code. It is an additional observation, not a separately asserted invariant.

`published` means successful replacement, not durability. Write errors are abstracted as failed publication; partial writes to non-regular destinations are not represented. The model does not retain byte-level output, exact JSON structure, or separate the command return from the root output latch; `outputError` records the latter's observable failure. The symbolic newline class is sufficient only for the displayed chat-text branch, not arbitrary payload parsing. Filesystem behavior and local HTTP semantics are assumptions; no concurrency, crash, or server rollback is proved.

## Replay

Run `python3 verification/state-machines/verify.py transfers` and `python3 verification/state-machines/verify.py transfers --self-test`. The latter runs real Go overlays and TLA+ mutants, then rechecks the baseline. Read every result with /statechart. Review source, tests, and transitions when a check fails; do not change a property only to make it pass.
