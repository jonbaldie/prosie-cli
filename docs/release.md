# Release workflow

Follow every step in order when shipping a CLI release. Each step ends on a completion criterion; a step is finished only when its criterion is observable in the output.

## 1. Quality gates

From the CLI root, run `make messgo`.

- `make messgo`: completion criterion is a clean exit (code 0). The ruleset `quality-gates/messgo-ruleset.xml` is the single home of rule exceptions; `ExitExpression` is the one documented exception (`main()` must call `os.Exit` to propagate exit codes). Extend that ruleset for a new unavoidable finding instead of arguing it away.
- `make mutago` (mutation testing, MSI target 80%) is **not** a local merge or release gate: it takes well over 15 minutes on a dev machine. It is to run in CI instead; see [issue #7](https://github.com/jonbaldie/prosie-cli/issues/7). Until that lands, run it opportunistically in the background when touching test-heavy code, and never hold a release for it.

## 2. Land, tag

`main` is PR-protected: a direct push is rejected with `GH013`. Push a branch, open the PR with `gh pr create`, wait for `gh pr checks` to pass, then `gh pr merge --squash --delete-branch`. Pull `main` and confirm the merge commit's CI run is green (`gh run list --branch main`).

Choose the version from everything unreleased, not just this change: `git log --oneline vLAST..main`. Any new command or flag bumps minor; fixes only bump patch. Write the tag message as the changelog, one line per PR.

Tag with `git tag -a vX.Y.Z` and push the tag. A push to `main` alone publishes nothing; only tags trigger asset builds.

Completion criterion: `gh run list --workflow release.yml` shows the tag run `success`, and `gh release view vX.Y.Z` lists all five platform assets plus `checksums.txt`.

## 3. Verify the release

Download the release assets with `gh release download vX.Y.Z` into a scratch directory, then verify against them:

- Compare each file's size with `gh release view vX.Y.Z --json assets -q '.assets[] | "\(.size) \(.name)"'`. A download cut short by a tool timeout leaves a plausible-looking but truncated archive that fails every checksum; re-download rather than debugging the hash.
- `shasum -a 256 -c checksums.txt` — every line reports OK.
- Extract the binary for the platform under test, confirm `prosie --version` reports the new tag, and replay the fixed user journeys on that binary.

Completion criterion: every checksum OK, version matches, and each replayed journey shows the fixed behaviour with exit 0 and empty stderr. Verify the released binary, not a local build.

## 4. Homebrew

Update `Formula/prosie.rb` in `jonbaldie/homebrew-tap`: new URLs for the tag and sha256 values taken from that release's `checksums.txt`.

- Hashes come from the official release assets. Local builds use a different Go toolchain than official CI; their hashes do not belong in the formula.
- Bump the `assert_match` version in the formula's test block.
- Commit and push the tap.

Check the hashes mechanically, not by eye: parse the `url`/`sha256` pairs out of the formula and compare each to `checksums.txt` (a five-line script; four pairs must match).

Then install from the tap on the dev machine. `brew update` can hang on the network; skip it and refresh only the tap:

```bash
git -C "$(brew --repository jonbaldie/tap)" pull
HOMEBREW_NO_AUTO_UPDATE=1 brew install jonbaldie/tap/prosie   # or `brew upgrade`
HOMEBREW_NO_AUTO_UPDATE=1 brew test jonbaldie/tap/prosie
prosie --version
```

Completion criterion: the tap commit is pushed, every URL and hash in the formula traces to the published `checksums.txt`, and `prosie --version` from the brew-installed binary reports the new tag. If `brew` is blocked by a pending Xcode license, formula correctness rests on the mechanical hash check and the brew-level check is left to the user.

## 5. Close out

Close the GitHub issues fixed by the release. Then record the release in the backend tracker: the `.scratch/<feature>/issues/` file in the prosie checkout gets the tag, checksum result, the replayed journey's exit code and output, and the tap commit SHA. The cross-repo rules for that tracker are in the prosie checkout at `docs/agents/cli-ecosystem.md`.

Completion criterion: `git status -sb` in both repos shows `## main...origin/main` with nothing unpushed.
