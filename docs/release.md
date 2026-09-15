# Release workflow

Follow every step in order when shipping a CLI release. Each step ends on a completion criterion; a step is finished only when its criterion is observable in the output.

## 1. Quality gates

From the CLI root, run `make messgo`.

- `make messgo`: completion criterion is a clean exit (code 0). The ruleset `quality-gates/messgo-ruleset.xml` is the single home of rule exceptions; `ExitExpression` is the one documented exception (`main()` must call `os.Exit` to propagate exit codes). Extend that ruleset for a new unavoidable finding instead of arguing it away.
- `make mutago` (mutation testing, MSI target 80%) is **not** a local merge or release gate: it takes well over 15 minutes on a dev machine. It is to run in CI instead; see [issue #7](https://github.com/jonbaldie/prosie-cli/issues/7). Until that lands, run it opportunistically in the background when touching test-heavy code, and never hold a release for it.

## 2. Commit, push, tag

Commit and push to `main`, confirm CI is green (`gh run list`), then tag `vX.Y.Z` and push the tag.

Completion criterion: `gh run list --workflow release.yml` shows the tag run `success`, and `gh release view vX.Y.Z` lists all five platform assets plus `checksums.txt`.

A push to `main` alone publishes nothing; only tags trigger asset builds.

## 3. Verify the release

Download the release assets, then verify against them:

- `shasum -a 256 -c checksums.txt` — every line reports OK.
- Extract the binary for the platform under test, confirm `prosie --version` reports the new tag, and replay the fixed user journeys on that binary.

Completion criterion: every checksum OK, version matches, and each replayed journey shows the fixed behaviour with exit 0 and empty stderr. Verify the released binary, not a local build.

## 4. Homebrew

Update `Formula/prosie.rb` in `jonbaldie/homebrew-tap`: new URLs for the tag and sha256 values taken from that release's `checksums.txt`.

- Hashes come from the official release assets. Local builds use a different Go toolchain than official CI; their hashes do not belong in the formula.
- Bump the `assert_match` version in the formula's test block.
- Commit and push the tap.

Completion criterion: the tap commit is pushed and every URL and hash in the formula traces to the published `checksums.txt`. `brew` on the dev machine may be blocked by a pending Xcode license; when so, formula correctness is established by inspection against the downloaded assets, and the brew-level check is left to the user.

## 5. Close out

Close the GitHub issues fixed by the release, update the backend task tracker (`.scratch/` in the prosie checkout) with the shipped state and evidence paths, and commit both repos.
