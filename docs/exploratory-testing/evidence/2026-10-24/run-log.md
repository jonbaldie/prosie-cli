# Run log

- 2026-10-24 UTC: Start on local `main` at `b95709d8dc1f6c0dfeebfee614c61e35939c27c3`. `build/prosie` was already present and untracked. It was not changed.
- Read the exploratory-testing skill, issue tracker rules, domain rules, previous report, PR skill, and ship-pr skill. No root context file or ADR directory was present.
- `git fetch origin` found two newer commits. Created `docs/exploratory-2026-10-24` from `origin/main` at `4710664d789b0e325cffdbae4edd36e35c07e867`, then rebuilt for the test pass.
- Go 1.26.3; `go.mod` has no external modules and no lockfile. No install was needed. Used `GOFLAGS=-mod=readonly` for build and tests. Build stayed in `build/exploratory-2026-10-24/`; no symlink was used.
- Shared `GOCACHE` was `/Users/jonathanbaldie/.fleet/caches/go-build`. `GOMODCACHE` was `/Users/jonathanbaldie/go/pkg/mod`. No dependency changes were required.
- Used the supplied `TMPDIR` for the driver scratch directory and issue/PR body files. No real user config or API credentials were used by the CLI journeys.
- The first driver run exited 0 and confirmed the DOCX JSON failure. Added a raw-stdout export control and disabled Python bytecode writes. Removed the bytecode from the first import.
- Repeated the full driver with new fixture and config state. Exit 0. The retained transcript is the second run. Both book and chapter JSON exports failed twice in each run. File and raw-stdout controls preserved the source bytes.
- Expected negative-path diagnostics are retained in `transcript.txt`: unauthenticated status, zero target, missing export directory, bad JSON import, and failed chat send. The DOCX ZIP failure is also retained there.
- `GOFLAGS=-mod=readonly go test ./...`: exit 0. `make messgo`: exit 0. Logs are adjacent to this file.
- Filed confirmed bug #103 with labels `bug` and `needs-triage`. No existing issue described this binary-data failure.
- The classic protection endpoint, `gh api repos/jonbaldie/prosie-cli/branches/main/protection`, exited 1 with HTTP 404: `Branch not protected`. This was a diagnostic endpoint mismatch, not permission to push to main. The rules endpoint, `gh api repos/jonbaldie/prosie-cli/rules/branches/main`, succeeded. It requires a pull request, blocks deletion and non-fast-forward pushes, and requires zero approving reviews. The run uses a PR and normal host merge without an administrator bypass.
- Cleanup: the driver stopped and closed its loopback server in `finally`; its temporary directory was removed. Removed `build/exploratory-2026-10-24/`. Preserved the pre-existing `build/prosie`. No Docker command was used, and no container, image, volume, or network was created.
