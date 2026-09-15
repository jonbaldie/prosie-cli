# prosie-cli

Official command-line interface for [Prosie](https://prosie.app).

## Overview

`prosie` is a command-line tool to manage books, chapters, and AI generation directly from your terminal.

## Installation

### Homebrew (macOS & Linux)

```bash
brew install jonbaldie/tap/prosie
```

### Pre-built Binaries

Download pre-built static binaries from [GitHub Releases](https://github.com/jonbaldie/prosie-cli/releases).

## Authentication

Log in with your browser using the OAuth 2.0 device authorization flow:

```bash
prosie auth login
```

Alternatively, set an environment variable:

```bash
export PROSIE_API_TOKEN="your_personal_access_token"
```

## Usage

```bash
# Books
prosie book list
prosie book show <id>
prosie book create --title "My Novel"
prosie book export <id>

# Chapters
prosie chapter list <book-id>
prosie chapter show <id>

# AI Generation
prosie generate continue <chapter-id>
prosie generate rewrite <chapter-id>
prosie generate summarize <chapter-id>

# LLM settings
prosie llm show
prosie llm show --json
prosie llm models
prosie llm models --json
prosie llm update --provider openrouter --model openai/gpt-5.6-luna
prosie llm update --provider openrouter --model openai/gpt-5.6-luna --json
```

## LLM settings

Use `prosie llm update` to change the provider or model. To store provider
keys, set one or more of these environment variables before you run the
command:

- `PROSIE_OPENAI_API_KEY`
- `PROSIE_ANTHROPIC_API_KEY`
- `PROSIE_OPENROUTER_API_KEY`

The CLI does not accept keys as command arguments and never returns stored
key values. `prosie llm show` reports only whether each key is configured.

## Saved books and exports

`book create --json` includes the saved book defaults and initial chapters. A new book starts with Chapter 1. `book show <id> --json` reports the same saved state.

`chapter reorder <book-id> <id1,id2,...>` puts the selected chapters first. Omitted chapters follow in their existing order. Repeated IDs and IDs from other books are ignored.

`chapter create --content` and `chapter update --file` accept prose. Book and chapter Markdown exports keep paragraph boundaries from plain text and HTML.

Use `prosie auth login --help`, `prosie auth status --help`, or `prosie auth logout --help` to see authentication options without starting an authentication action.
