# Prosie CLI

Only reply in ASD-STE100 Simplified Technical English.

`prosie-cli` is the official command-line interface for the Prosie novel writing platform.

## Quality Standards

- Maintain zero external runtime dependencies for users.
- All commands must support `--json` output.
- All errors must write to standard error.
- All tests must verify external behavior and exit codes.
- Keep test coverage comprehensive. Run `go test ./...` before commit.
- Quality gate for production code: `make messgo`, clean exit before any PR. Rules and exceptions live in `quality-gates/messgo-ruleset.xml`. `make mutago` is informational until it runs in CI (#7).

## API parity

The backend API is the contract. Each request field a backend responder validates (`app/Services/Api/*Responder.php` in the prosie checkout, `validate([...])`) is reachable from a CLI flag. A field the API accepts and the CLI cannot send is a parity gap: fix it here with a `*Params` struct on the client (see `client.ContinueParams`, `client.RewriteParams`), zero values omitted from the body, and a cmd-level test that captures the request body. The cross-repo rules (which repo changes when, and where the work is tracked) are in the prosie checkout at `docs/agents/cli-ecosystem.md`.

## Release workflow

When shipping a release (landing the PR, tagging, verifying released assets, updating the Homebrew formula), read `docs/release.md` and follow it end to end. `main` is PR-protected; every change lands through a pull request.

## Agent skills

### Issue tracker

Use GitHub Issues for this repository. Before issue operations, read `docs/agents/issue-tracker.md`.

### Triage labels

Use the five default triage labels. Before triage, read `docs/agents/triage-labels.md`.

### Domain docs

Use the single-context layout: root `CONTEXT.md` and `docs/adr/`. Before code exploration, read `docs/agents/domain.md`.
