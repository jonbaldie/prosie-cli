# Prosie CLI

Only reply in ASD-STE100 Simplified Technical English.

`prosie-cli` is the official command-line interface for the Prosie novel writing platform.

## Quality Standards

- Maintain zero external runtime dependencies for users.
- All commands must support `--json` output.
- All errors must write to standard error.
- All tests must verify external behavior and exit codes.
- Keep test coverage comprehensive. Run `go test ./...` before commit.
