---
name: verify-http-flows
description: Check HTTP response ownership or best-effort book and series relation lookup after changes to the client.
---

# Verify HTTP flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

An ordinary `readResponse` must close its response before returning; a leak can exhaust connections. A failed chapter lookup must not discard a decoded book; otherwise the user loses access to the book.

Coverage: `internal/client/client.go` `NewRequest`, `newRequest`, `Do`, `CheckResponse`; `internal/client/response.go` `request`, `readResponse`, `execute`; `internal/client/generate.go` `openSSEStream`; `internal/client/books.go` `GetBook`, `bookChapters`, `ListBooks`; `internal/client/series.go` `ListSeries`, `GetSeries`, `populateSeriesBooks`, `newSeriesBookIndex`, `members`. The assertions cover only the narrower paths in `verification/state-machines/http/README.md`.

Last synchronized with `3dd1171b29bd4dccc8c4e995438f658f18b818b1`. Inspect changes with:

```sh
git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/client/client.go internal/client/response.go internal/client/generate.go internal/client/books.go internal/client/series.go
```

From the repository root run:

```sh
python3 verification/state-machines/verify.py http
python3 verification/state-machines/verify.py http --self-test
```

Read EVERY result with /statechart. The first command checks bounded models and Go tests; the second also checks scratch source and model defects. These tools add no CLI runtime dependency. Each model run has one request flow and finite abstract outcomes. HTTP transport and server state are external assumptions. A stream caller close is modeled but is not a liveness guarantee. The Go test checks `StreamContinue` closes its body after parsing. Secondary read errors can still produce decodable bytes. Cancellation does not imply server rollback. The refined model checks rejected-stream body closure as well. Other stream and series behavior is observational. Read the README for the remaining model gaps and use its specified view for each model.

For a failure, replay the named path against the source and the real test. A model counterexample alone is not a Go bug. Review changed atomic steps, tests, assumptions, and each mutant; update them together for intended behavior, then run both commands again. Move the sync commit only when all covered functions have been reconciled. For stale verification use /maintain-formal-verification-skill.
