# Release notes

## Unreleased

### Positional argument limits (#95)

Commands with fixed argument counts now reject extra positional arguments. They exit with code 1 and write an error and usage text to standard error before any API request, confirmation prompt, or file input.

For example, `prosie book delete 1 2 3 -y` no longer deletes book 1 and reports success. It reports `unexpected argument "2"` and does not delete any books. Use one command for each book.

Scripts that passed extra arguments must remove them. Missing-argument messages do not change. `chapter reorder`, `chat send`, and `chat stream` still accept variable argument counts.

LLM commands use the same count checks, but do not print unexpected values. This keeps possible API keys out of error output.
