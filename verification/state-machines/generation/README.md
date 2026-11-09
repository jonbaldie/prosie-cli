# Generation: S12 and S13

## Checked guarantees

- `FreshCancel`: continuation cancellation uses a fresh context. A cancelled context can prevent the cancel request reaching the server. The model does not prove a five-second wall-clock timeout, delivery, or server termination.
- `NoEmptyNewline`: successful stream finalization adds no newline when the last token is empty or absent. An extra newline changes output.
- `JSONDisablesStreaming`: JSON continuation requests do not select the stream endpoint. Streaming JSON would change both the protocol and output.

## Source and abstraction

| Source | Actions |
| --- | --- |
| `generate_continue.go` `parseContinueArgs`, `validateLengthFlags`, `useStreaming` | `Choose`, `Validate` |
| `executeGenerateContinue`, `streamContinue`, `continuationError` | `Client`, `Success`, `ContinueError`, `CancelRequest`, `Finish` |
| `generate_rewrite.go` `executeGenerateRewrite`, `streamRewrite` | `Choose`, `Validate`, `Client`, `Token`, `Success`, `Failure`, `Finish` |
| `generate_summarize.go` `executeGenerateSummarize` | `Choose`, `Validate`, `Client`, `Success`, `Failure`, `Finish` |
| `output.go` `newTokenOutput` | `Token`, `Finish` |

The input classes include explicit stream, no-stream, JSON, persist and no-persist booleans. They include the defaults as well as overrides; they do not model the actual argument parser. The no-stream and no-persist inputs do not exist on rewrite, and no-stream does not exist on summarize. Continue words and lines distinguish omitted, visited positive, and visited nonpositive. Validation precedes client creation. Rewrite content acquisition and validation are modeled by S07, not here. Client creation can fail. Signal or caller context completion is represented by `context-done`; an error matching cancellation independently selects the same best-effort continuation cancellation. Rewrite and summarize do not issue this remote cancel. The background request error is ignored. There is no claim that the server stopped.

The environment selects the server's persisted/not-persisted/conflict response; it is not derived from the requested persist flag. The conflict and other error paths do not finish output. Nil streaming completion does not finish. The `Token` action replaces the last-token class, and may repeat arbitrarily; it does not bound the number of tokens. `Finish` covers both actual finalization and the ordinary/nil no-op. An output failure is retained observationally; this model does not reproduce the root writer's first-error latch or final exit code (S03). Exact token bytes, output operations, parsing errors, remote cancellation internals, and a separate signal-delivery timeline are outside this model. Ordinary and nil results are coalesced where their further behavior is irrelevant to these properties. No liveness is claimed.

`Generation.cfg` explores all input classes without pruning. `Generation-chart.cfg` selects one complete default continue flow before execution, rather than pruning intermediate states. It is for the statechart when the full graph exceeds 500 states. This scenario does not replace the baseline.

## Evidence and replay

Run `python3 verification/state-machines/verify.py generation --self-test` from the repository root. Read every result with /statechart. `TestFormalGenerationCancellation` calls the real error handler with a cancelled context and observes a POST to a local HTTP server, which returns an error. The dead-context overlay prevents the POST. `TestFormalGenerationOutput` invokes the real closure and observes its bytes for empty, newline, and multiple tokens. `TestFormalGenerationJSONMode` enumerates the actual `useStreaming` function's boolean inputs. Each matching model mutation violates its named invariant; Go overlays compile and fail running tests. These checks do not exercise the complete signal path or a real backend. All non-invariant behaviors above are observational, not independently proved guarantees.
