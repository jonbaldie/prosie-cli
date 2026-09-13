# Issue tracker: GitHub

Store issues and specifications in GitHub Issues for `jonbaldie/prosie-cli`. Use the `gh` CLI from this repository. Check `git remote -v` to identify the repository when necessary.

## Issue operations

- Create: `gh issue create --title "..." --body-file <file>`.
- Read: `gh issue view <number> --json number,title,body,labels,comments`.
- List: `gh issue list --state open --json number,title,body,labels,comments`. Add label or state filters as necessary.
- Comment: `gh issue comment <number> --body-file <file>`.
- Add a label: `gh issue edit <number> --add-label "..."`.
- Remove a label: `gh issue edit <number> --remove-label "..."`.
- Close: `gh issue close <number>`.

For text with multiple lines, write the exact text to a temporary file. Pass that file with `--body-file`.

When a skill says "publish to the issue tracker", create a GitHub issue. When a skill says "fetch the relevant ticket", read the issue and its comments.

## Pull requests as a triage surface

**PRs as a request surface: no.**

## Wayfinding operations

The map is one issue with the label `wayfinder:map`. Its body contains Notes, Decisions-so-far, and Fog. Use child issues for tickets.

- Link each child as a GitHub sub-issue. If sub-issues are unavailable, use a task list in the map and put `Part of #<map>` at the top of each child.
- Use `wayfinder:<type>` labels, where the type is `research`, `prototype`, `grilling`, or `task`.
- Record blockers with GitHub issue dependencies. If dependencies are unavailable, put `Blocked by: #<n>, #<n>` at the top of the child body. A ticket is available only when all its blockers are closed.
- Select the first open child in map order that has no open blockers and no assignee.
- Claim the ticket with `gh issue edit <number> --add-assignee @me` as the first write in the session.
- To resolve a ticket, add the answer as a comment, close the ticket, and add a short result with a link to Decisions-so-far in the map.
