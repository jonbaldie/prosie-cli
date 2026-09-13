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
```
