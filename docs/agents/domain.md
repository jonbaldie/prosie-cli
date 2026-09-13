# Domain docs

Use a single-context layout for this repository.

## Before code exploration

Read `CONTEXT.md` at the repository root, if it exists. Read the architecture decision records (ADRs) in root `docs/adr/` that apply to the work.

If these files are absent, continue without a warning or a request to create them. The `domain-modeling` skill creates them when terms or decisions are agreed.

## File layout

- `CONTEXT.md`: domain terms and definitions for the repository.
- `docs/adr/`: numbered architecture decision records.

## Domain terms

Use the terms defined in `CONTEXT.md` in issue titles, proposals, explanations, and test names. If a term is absent, check whether an existing term applies. Record missing definitions for `domain-modeling`.

## Decision conflicts

If a proposed change conflicts with an ADR, identify the ADR. Explain the conflict and why the decision needs review.
