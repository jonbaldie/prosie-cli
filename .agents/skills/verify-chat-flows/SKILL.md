---
name: verify-chat-flows
description: Verify chat conversation creation, send failure recovery, and streamed output when changing chat client or command flows.
---

# Verify chat flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A created conversation's ID and created flag must survive a failed send. Otherwise the user can lose the way to resume it. JSON streams must suppress delta text, or the output is not valid JSON.

Coverage: `internal/client/chat.go` `SendTurn`, `CreateConversation`, `SendChatMessage`, `StreamChatMessage`; `internal/commands/chat/turn.go` `sendTurn`, `writeTurnJSON`; `internal/commands/chat/chat_send.go` `executeChatSend`; `internal/commands/chat/chat_stream.go` `executeChatStream`. Last source sync: `3dd1171b29bd4dccc8c4e995438f658f18b818b1`.

1. Inspect changes: `git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/client/chat.go internal/commands/chat/turn.go internal/commands/chat/chat_send.go internal/commands/chat/chat_stream.go`.
2. From the root run `python3 verification/state-machines/verify.py chat`. Read EVERY result with `/statechart` by name.
3. Run `python3 verification/state-machines/verify.py chat --self-test` to check both real-source and model defects and repeat the baseline. These tools add no CLI runtime dependency.
4. If a check fails, distinguish a real regression from stale model, test, or tool setup. Replay the trace through the real functions and compare actual request order, result, errors and output. A model pass is not a Go proof.
5. For an intentional change reconcile source, model actions, test observations, mutants, bounds and README. Confirm each defect fails for the intended reason and baseline passes before advancing the sync commit. Use `/maintain-formal-verification-skill` for stale verification.

The baseline covers one turn, up to two tokens, ordinary and stream output modes, and created IDs 77 and zero. `Chart.cfg` selects a smaller stream scenario for `/statechart`. The model abstracts transport, server behavior and partial-write bytes; it does not prove persistence, rollback, output success, or liveness. The README records the source-to-action map and limitations.
