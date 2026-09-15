package codexcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func Execute(c *command.Environment, args []string) int {
	if len(args) == 0 {
		printCodexHelp(c)
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		printCodexHelp(c)
		return 0
	case "list":
		return executeCodexList(c, subArgs)
	case "show":
		return executeCodexShow(c, subArgs)
	case "create":
		return executeCodexCreate(c, subArgs)
	case "update":
		return executeCodexUpdate(c, subArgs)
	case "delete":
		return executeCodexDelete(c, subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown codex command: %s\nRun 'prosie codex --help' for usage.\n", subCmd)
		return 1
	}
}

// printCodexHelp prints help for the codex command hierarchy.
func printCodexHelp(c *command.Environment) {
	help := `Manage story codex entries, character notes, and worldbuilding lore.

Usage:
  prosie codex <command> [flags]

Available Commands:
  list        List all codex entries for a book
  show        Show codex entry details
  create      Create a new codex entry for a book
  update      Update an existing codex entry
  delete      Delete a codex entry

Flags:
  -h, --help   Show help for command

Use "prosie codex <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}

func executeCodexList(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printCodexListHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie codex list <book-id> [flags]")
		return 1
	}

	bookID, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entries, err := cli.Codex().ListCodexEntries(context.Background(), bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error listing codex entries: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, entries)
		return 0
	}

	printEntries(c, bookID, entries)

	return 0
}

func printCodexListHelp(c *command.Environment) {
	help := `List all codex entries for a book.

Usage:
  prosie codex list <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func executeCodexShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printCodexShowHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: codex entry ID is required. Usage: prosie codex show <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid codex entry ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entry, err := cli.Codex().GetCodexEntry(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, entry)
		return 0
	}

	printEntry(c, entry)

	return 0
}

func printCodexShowHelp(c *command.Environment) {
	help := `Show codex entry details and lore content.

Usage:
  prosie codex show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func executeCodexCreate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	name := fs.String("name", "", "Entry name (required)")
	entryType := fs.String("type", "", "Entry type (lore or character)")
	category := fs.String("category", "", "Entry category (alias for type)")
	details := fs.String("details", "", "Entry description and facts (required)")
	content := fs.String("content", "", "Entry content (alias for details)")
	aliases := fs.String("aliases", "", "Alternative names and aliases")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printCodexCreateHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie codex create <book-id> [flags]")
		return 1
	}

	bookID, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	finalDetails, err := creationDetails(*name, *details, *content)
	if err != nil {
		fmt.Fprintln(c.Err, err)
		return 1
	}
	finalType := firstText(*entryType, *category, "lore")

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	params := client.CreateCodexParams{
		Name:     *name,
		Type:     finalType,
		Category: finalType,
		Details:  finalDetails,
		Content:  finalDetails,
	}
	params.Aliases = optionalAliases(aliases)

	entry, err := cli.Codex().CreateCodexEntry(context.Background(), bookID, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, entry)
		return 0
	}

	fmt.Fprintf(c.Out, "Created codex entry %q (ID: %d, type: %s).\n", entry.Name, entry.ID, entry.Display().Type)
	return 0
}

func printCodexCreateHelp(c *command.Environment) {
	help := `Create a new codex entry for a book.

Usage:
  prosie codex create <book-id> --name <name> --details <details> [flags]

Flags:
      --aliases string    Alternative names and aliases
      --category string   Entry category (alias for --type)
      --content string    Entry content (alias for --details)
      --details string    Entry description and facts (required)
  -h, --help              Show help for command
      --json              Format output as JSON
      --name string       Entry name (required)
      --type string       Entry type: lore or character (default: lore)
`
	fmt.Fprint(c.Out, help)
}

func executeCodexUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	name := fs.String("name", "", "Entry name")
	entryType := fs.String("type", "", "Entry type (lore or character)")
	category := fs.String("category", "", "Entry category (alias for type)")
	details := fs.String("details", "", "Entry description and facts")
	content := fs.String("content", "", "Entry content (alias for details)")
	aliases := fs.String("aliases", "", "Alternative names and aliases")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printCodexUpdateHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: codex entry ID is required. Usage: prosie codex update <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid codex entry ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	visited := command.VisitedFlags(fs)
	finalType := providedAlias(visited, "type", entryType, "category", category)
	finalDetails := providedAlias(visited, "details", details, "content", content)
	params := client.UpdateCodexParams{
		Name: command.Provided(visited, "name", name),
		Type: finalType, Category: finalType,
		Details: finalDetails, Content: finalDetails,
		Aliases: command.Provided(visited, "aliases", aliases),
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entry, err := cli.Codex().UpdateCodexEntry(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, entry)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated codex entry %d (%q).\n", entry.ID, entry.Name)
	return 0
}

func printCodexUpdateHelp(c *command.Environment) {
	help := `Update an existing codex entry.

Usage:
  prosie codex update <id> [flags]

Flags:
      --aliases string    Alternative names and aliases
      --category string   Entry category (alias for --type)
      --content string    Entry content (alias for --details)
      --details string    Entry description and facts
  -h, --help              Show help for command
      --json              Format output as JSON
      --name string       Entry name
      --type string       Entry type: lore or character
`
	fmt.Fprint(c.Out, help)
}

func executeCodexDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printCodexDeleteHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: codex entry ID is required. Usage: prosie codex delete <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid codex entry ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	if !command.ConfirmDeletion(c, *yesFlag, "codex entry", id) {
		return 0
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Codex().DeleteCodexEntry(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted codex entry %d.\n", id)
	return 0
}

func printCodexDeleteHelp(c *command.Environment) {
	help := `Delete a codex entry.

Usage:
  prosie codex delete <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
  -y, --yes    Skip confirmation prompt
`
	fmt.Fprint(c.Out, help)
}

func printEntries(c *command.Environment, bookID int, entries []client.CodexEntry) {
	if len(entries) == 0 {
		fmt.Fprintf(c.Out, "No codex entries found for book %d.\n", bookID)
		return
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tNAME\tDETAILS\tUPDATED")
	for _, e := range entries {
		detailsPreview := e.Display().Details
		if len(detailsPreview) > 40 {
			detailsPreview = detailsPreview[:37] + "..."
		}
		updatedStr := command.FormatTimestamp(e.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", e.ID, e.Display().Type, e.Name, detailsPreview, updatedStr)
	}
	_ = w.Flush()

}

func printEntry(c *command.Environment, entry *client.CodexEntry) {
	fmt.Fprintf(c.Out, "Name:     %s\n", entry.Name)
	fmt.Fprintf(c.Out, "ID:       %d\n", entry.ID)
	fmt.Fprintf(c.Out, "Type:     %s\n", entry.Display().Type)
	fmt.Fprintf(c.Out, "Aliases:  %s\n", entry.Display().Aliases)
	if entry.StoryID != nil {
		fmt.Fprintf(c.Out, "Book ID:  %d\n", *entry.StoryID)
	}
	if entry.SeriesID != nil {
		fmt.Fprintf(c.Out, "Series:   %d\n", *entry.SeriesID)
	}
	fmt.Fprintf(c.Out, "Updated:  %s\n", command.FormatTimestamp(entry.UpdatedAt))
	fmt.Fprintln(c.Out)
	fmt.Fprintf(c.Out, "Details:\n%s\n", entry.Display().Details)

}

func firstText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func providedAlias(visited map[string]bool, name string, value *string, alias string, alternative *string) *string {
	if visited[name] {
		return value
	}
	return command.Provided(visited, alias, alternative)
}

func optionalAliases(value *string) *string {
	if strings.TrimSpace(*value) == "" {
		return nil
	}
	return value
}

func creationDetails(name, details, content string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("error: --name is required")
	}
	value := firstText(details, content)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("error: --details is required")
	}
	return value, nil
}
