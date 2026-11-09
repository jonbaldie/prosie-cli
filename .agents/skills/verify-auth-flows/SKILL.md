---
name: verify-auth-flows
description: Check authentication verification, device polling, status, or logout changes against the bounded auth model and real-code tests.
---

# Verify auth flows

Use Python 3, Java 11+, curl and Go on Linux or macOS. Read `verification/state-machines/README.md` for pinned tool versions. First-run downloads can need network access. HTTP fixtures use loopback only.

A personal token must pass verification before it is saved. A denied device request must stop polling. A status lookup must select an environment token, including after logout. A poll must not start when cancellation was already observed at loop entry. A defect can replace credentials, accept a token after denial, ignore an active environment token, or send an unwanted request.

Coverage: `internal/auth/auth.go` (`LoginWithToken`, `InspectStatus`, `Logout`), `internal/auth/device.go` (`RequestDeviceCode`, `PollForToken`, `pollUntilAuthorized`, `pollTokenOnce`, `tokenAttemptFailure`), `internal/commands/auth/auth_login.go` (`executeAuthLogin`, `loginWithDevice`, `showDeviceAuthorization`, `deviceTiming`), and `internal/config/config.go` (`Load`, `Save`, `Clear`, `ResolveToken`, `ResolveApiURL`). Successful status, config removal, device saving and profile availability are observations, not additional guarantees beyond the four listed rules. See `verification/state-machines/auth/README.md` for mappings and limits.

The sync commit is `3dd1171b29bd4dccc8c4e995438f658f18b818b1`. From the root, inspect changes with:

```sh
git diff 3dd1171b29bd4dccc8c4e995438f658f18b818b1 -- internal/auth/auth.go internal/auth/device.go internal/commands/auth/auth_login.go internal/config/config.go
python3 verification/state-machines/verify.py auth
python3 verification/state-machines/verify.py auth --self-test
```

Read EVERY result with /statechart. Use `Auth-chart.cfg` for the device-flow chart. The model separates command config load from direct-helper verification, and retains URL and scope classes through save and status. See the README for string and output limits; S03 covers the shared output protocol. The model bounds pending and slow responses to two, not token lifetime, and abstracts filesystem and network calls. It does not prove real timing, server token validity, logout revocation, or all executions. Device login saves before profile lookup, including when an issued token is empty; an environment token survives logout. The checks add no CLI runtime dependency.

On failure, replay the path against the real code and tests. Separate a real regression from a stale model, setup error, or changed contract. Review each source and model mutant in `manifest.json`; both must fail for the stated rule while the baseline passes. Reconcile intentional behavior changes in the model, tests, mutants, documentation and chart; rerun both commands. Move the sync commit only after confirming every covered change is represented. Use /maintain-formal-verification-skill for stale verification.
