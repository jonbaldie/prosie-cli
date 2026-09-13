# Prosie CLI

Only reply in ASD-STE100 Simplified Technical English.

`prosie-cli` is the official command-line interface for the Prosie novel writing platform.

## Quality Standards

- Maintain zero external runtime dependencies for users.
- All commands must support `--json` output.
- All errors must write to standard error.
- All tests must verify external behavior and exit codes.
- Keep test coverage comprehensive. Run `go test ./...` before commit.

## Agent skills

### Issue tracker

Use GitHub Issues for this repository. Before issue operations, read `docs/agents/issue-tracker.md`.

### Triage labels

Use the five default triage labels. Before triage, read `docs/agents/triage-labels.md`.

### Domain docs

Use the single-context layout: root `CONTEXT.md` and `docs/adr/`. Before code exploration, read `docs/agents/domain.md`.
