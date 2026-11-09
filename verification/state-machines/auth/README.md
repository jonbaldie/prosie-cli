# Authentication: S04 and S05

## Checked rules

- `VerifiedBeforeTokenSave`: a personal-token login cannot save a token before verification. This prevents rejected credentials replacing saved credentials.
- `DeniedNotIssued`: denial stops polling. A later response must not be accepted after denial.
- `EnvironmentSelected`: when status performs a lookup with an environment token, that token is selected ahead of the file token. Logout does not remove it.
- `NoRequestAfterEntryCancel`: a cancellation already observed at a non-expired loop entry prevents the next poll. Cancellation during an in-flight request is different.

The four rules are checked on the bounded model and have matched source and model defects. They are not proofs of all Go executions. Other behavior below is modeled as observation, not asserted as a guarantee.

## Source and model mapping

- `internal/commands/auth/auth_login.go:executeAuthLogin`: `CommandLoad` loads config and selects the effective URL before `ChooseLogin` reaches either verification or the device-code request. Malformed config fails before either request. The separate direct-helper entry starts at `verify`, without the command load.
- `internal/auth/auth.go:LoginWithToken`: `TokenVerify`, `TokenReload`, `TokenSave` separate verification, config reload and publication, including failures. The helper's empty-token check is outside this model. A successful save retains nonempty existing scopes or stores default personal scopes.
- `internal/commands/auth/auth_login.go:loginWithDevice`, `showDeviceAuthorization`: `DeviceRequest`, `Display`, `DeviceSave`, `Profile` represent request failure, browser skipped/started/failed without stopping, save failure, and success with or without user information after save. Issued scopes, including empty scopes, are stored before profile lookup. Status defaults empty scopes in its returned result only; it does not change stored scopes. Output failure is not modeled here: S03 is the shared output protocol, not a duplicate formatting machine.
- `internal/auth/device.go:PollForToken`, `pollUntilAuthorized`, `pollTokenOnce`, `tokenAttemptFailure`: `Entry`, `DevicePoll`, `Backoff`, `Wait`, `Cancel`. Entry checks expiry before context; an in-flight success can be accepted after expiry or cancellation. Pending keeps the interval; slow-down adds five. Wait can cancel or reach the next entry. Poll outcomes include denial, server expiry, cancellation and other failure. A successful response can contain an empty access token; `issuedEmpty` carries that case through saving and the later status journey. Defaults and response decoding are not separately modeled.
- `internal/auth/auth.go:InspectStatus`, `Logout`; `internal/config/config.go:Load`, `ResolveToken`, `Clear`: `StatusLoad`, `Resolve`, `StatusLookup`, `Logout`, `AfterLogout`, `AfterLogin` carry file state and environment selection through logout or device login into a fresh status. Missing, token-bearing and malformed files, load/clear failure, no token, failed lookup and successful lookup have distinct states. A token's validity is an external response: the lookup can succeed or fail regardless of the token class. The filesystem and link handling are abstracted; no revocation is implied.

## Bounds and assumptions

Clock starts at 0, deadline is 6, and the initial interval is 1. Time 7 represents strictly after the deadline. An in-flight request either leaves the clock unchanged or moves it after the deadline; a wait advances by the current interval and saturates at 7. These are representative order classes, not a wall-clock proof or an exhaustive set of durations. The environment may return at most two pending/slow-down responses; its next response is terminal. This bounds response sequences, not token lifetime. Cancellation during a request is represented by an independent context transition and an observed request outcome; transport details are external.

URL classes retain the chosen environment, config, default or helper value across save and later status. `ResolvedURL` is a pure precedence transformation; classes abstract away URL strings and trailing-slash normalization, including the edge case of an all-slash value. Scope classes distinguish empty, existing nonempty, default and issued nonempty, not individual scope strings. These abstractions do not prove exact URL or scope values. The direct helper can receive a URL independent of config. Initial file classes are independent abstractions; there is no concurrent external config edit between actions. The model does not prove output behavior or rollback on writer failure.

The main model has 41,000 reachable states. `Auth-chart.cfg` selects the device flow, absent file, no environment token and browser start failure, and permits one pending/slow response. It ends at login instead of entering the subsequent status journey. It does not prune the state graph. The chart merges internal states with the same label. Neither config makes a liveness claim. No concurrency, server-side revocation, browser implementation, real clock or credential storage atomicity is proved.

## Replay and defects

Run `python3 verification/state-machines/verify.py auth --self-test` from the root, and read EVERY result with /statechart. Use `Auth-chart.cfg` for the device chart. The runner checks actual Go tests, TLC, each scratch Go overlay and each model defect, then repeats the baseline. The tests use local servers or an in-memory transport and temporary files. They check rejected-token file preservation, denial and request count, logout followed by real environment-token status, immediate cancellation without a request, and both root login paths with malformed config and no request.

The paired defects save a rejected token, treat denial as pending, ignore the environment token, and skip the entry cancellation check. Each source defect fails its named test; each corresponding model defect violates its named invariant. A model mutation approximates a source defect; TLC does not execute Go. See `manifest.json` for exact mutations.
