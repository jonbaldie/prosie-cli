# Chat: S10 and S11

## Checked properties

- `RetainCreated`: after decoding a created conversation ID, the returned result retains that ID and the created flag through send success or failure. Losing either prevents a caller from giving the correct resume instruction.
- `JSONSuppressesDeltas`: a JSON stream never writes delta text. A leak corrupts the final JSON output. The stream still uses SSE.

These are bounded model checks, not proofs of the Go code. Other behaviors below are represented but not asserted as invariants.

## Source map and bounds

`internal/client/chat.go` `SendTurn`, `CreateConversation`, `SendChatMessage` and `StreamChatMessage`: `Existing` retains a nonempty raw ID, including whitespace; `Create` captures a separately chosen decoded ID (77 or zero), assigns it to the result, and marks it created. `CreateFail` represents creation/decoding failures without a send. `Choose` selects a nil callback for ordinary HTTP or a non-nil callback for SSE; the whitespace class fails sender validation after selection. `Success`/`Failure` observe the send result, not remote persistence. A decoded zero ID goes to the sender as `0`. Creation is atomic after decoding in this model; a remote creation can have happened before an error.

`internal/commands/chat/turn.go` `sendTurn` and `writeTurnJSON`, `chat_send.go` `executeChatSend`, `chat_stream.go` `executeChatStream`: `Delta` keeps the token sequence and output, with text and trailing-newline token classes. Text callbacks attempt each write; JSON callbacks suppress all deltas. `Report` emits ordinary message text only for non-nil messages, emits JSON on success, emits a text-stream newline on every success, and emits one on failure only if a callback occurred. Thus a trailing-newline token gets another newline, unlike generation finalization. A write may fail with partial output; the latch prevents subsequent successful output and changes an otherwise successful root exit to failure. This abstracts the root output writer: individual bytes of a partial write and stderr are not modeled. `response` distinguishes nil and non-nil message in an otherwise successful response; a nil response pointer is not invented for successful client calls.

One turn, two tokens, two token classes, four command/mode classes and two representative created IDs are bounded. HTTP/SSE framing, auth, command parsing and server state are external; request/decoder errors are coalesced when the retained result and future behavior match. A send error is not a remote rollback. Token classes stand for nonempty values; empty callback tokens, exact message contents, argument-validation failures before `SendTurn`, and exact resume-hint text are not separately represented. No liveness claim is made.

`Chart.cfg` selects a complete smaller stream scenario (at most one token and representative IDs) without pruning states. The full baseline uses `Chat.cfg` and two tokens. The chart view groups output variants; see the full model for partial-write details.

## Evidence and replay

Run `python3 verification/state-machines/verify.py chat --self-test` from the root and read every result with `/statechart`. `TestFormalChatRetainsCreatedConversation` observes ordered local HTTP requests and the actual retained result for ordinary and streamed send failures. `TestFormalChatStreamOutput` drives the real root command with local SSE for success/failure, zero/one/two tokens, trailing newlines, and JSON success/failure. It checks the actual endpoint and output. These examples do not exhaust real inputs.

The `lose-created-flag` and `lose-conversation-id` Go overlays pair with the matching model defects for `RetainCreated`. `json-delta-leak` makes the actual JSON callback print and pairs it with a model callback that prints; both must violate their checks. The common runner, HTTP and OS implementations are trusted infrastructure. No runtime dependency is added.
