package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func (c *RootCmd) executeCodex(args []string) int {
	if len(args) == 0 {
		c.PrintCodexHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintCodexHelp()
		return 0
	case "list":
		return c.executeCodexList(subArgs)
	case "show":
		return c.executeCodexShow(subArgs)
	case "create":
		return c.executeCodexCreate(subArgs)
	case "update":
		return c.executeCodexUpdate(subArgs)
	case "delete":
		return c.executeCodexDelete(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown codex command: %s\nRun 'prosie codex --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintCodexHelp prints help for the codex command hierarchy.
func (c *RootCmd) PrintCodexHelp() {
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

func (c *RootCmd) executeCodexList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintCodexListHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entries, err := cli.ListCodexEntries(context.Background(), bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error listing codex entries: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(entries)
		return 0
	}

	if len(entries) == 0 {
		fmt.Fprintf(c.Out, "No codex entries found for book %d.\n", bookID)
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tNAME\tDETAILS\tUPDATED")
	for _, e := range entries {
		detailsPreview := e.DisplayDetails()
		if len(detailsPreview) > 40 {
			detailsPreview = detailsPreview[:37] + "..."
		}
		updatedStr := formatTimestamp(e.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", e.ID, e.DisplayType(), e.Name, detailsPreview, updatedStr)
	}
	_ = w.Flush()

	return 0
}

func (c *RootCmd) PrintCodexListHelp() {
	help := `List all codex entries for a book.

Usage:
  prosie codex list <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeCodexShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintCodexShowHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entry, err := cli.GetCodexEntry(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(entry)
		return 0
	}

	fmt.Fprintf(c.Out, "Name:     %s\n", entry.Name)
	fmt.Fprintf(c.Out, "ID:       %d\n", entry.ID)
	fmt.Fprintf(c.Out, "Type:     %s\n", entry.DisplayType())
	fmt.Fprintf(c.Out, "Aliases:  %s\n", entry.DisplayAliases())
	if entry.StoryID != nil {
		fmt.Fprintf(c.Out, "Book ID:  %d\n", *entry.StoryID)
	}
	if entry.SeriesID != nil {
		fmt.Fprintf(c.Out, "Series:   %d\n", *entry.SeriesID)
	}
	fmt.Fprintf(c.Out, "Updated:  %s\n", formatTimestamp(entry.UpdatedAt))
	fmt.Fprintln(c.Out)
	fmt.Fprintf(c.Out, "Details:\n%s\n", entry.DisplayDetails())

	return 0
}

func (c *RootCmd) PrintCodexShowHelp() {
	help := `Show codex entry details and lore content.

Usage:
  prosie codex show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeCodexCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	name := fs.String("name", "", "Entry name (required)")
	entryType := fs.String("type", "", "Entry type (lore or character)")
	category := fs.String("category", "", "Entry category (alias for type)")
	details := fs.String("details", "", "Entry description and facts (required)")
	content := fs.String("content", "", "Entry content (alias for details)")
	aliases := fs.String("aliases", "", "Alternative names and aliases")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintCodexCreateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(c.Err, "error: --name is required")
		return 1
	}

	finalDetails := *details
	if finalDetails == "" {
		finalDetails = *content
	}
	if strings.TrimSpace(finalDetails) == "" {
		fmt.Fprintln(c.Err, "error: --details is required")
		return 1
	}

	finalType := *entryType
	if finalType == "" {
		finalType = *category
	}
	if finalType == "" {
		finalType = "lore"
	}

	cli, err := c.Client()
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
	if strings.TrimSpace(*aliases) != "" {
		params.Aliases = aliases
	}

	entry, err := cli.CreateCodexEntry(context.Background(), bookID, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(entry)
		return 0
	}

	fmt.Fprintf(c.Out, "Created codex entry %q (ID: %d, type: %s).\n", entry.Name, entry.ID, entry.DisplayType())
	return 0
}

func (c *RootCmd) PrintCodexCreateHelp() {
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

func (c *RootCmd) executeCodexUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	name := fs.String("name", "", "Entry name")
	entryType := fs.String("type", "", "Entry type (lore or character)")
	category := fs.String("category", "", "Entry category (alias for type)")
	details := fs.String("details", "", "Entry description and facts")
	content := fs.String("content", "", "Entry content (alias for details)")
	aliases := fs.String("aliases", "", "Alternative names and aliases")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintCodexUpdateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	params := client.UpdateCodexParams{}
	if visited["name"] {
		params.Name = name
	}
	if visited["type"] {
		params.Type = entryType
		params.Category = entryType
	} else if visited["category"] {
		params.Type = category
		params.Category = category
	}
	if visited["details"] {
		params.Details = details
		params.Content = details
	} else if visited["content"] {
		params.Details = content
		params.Content = content
	}
	if visited["aliases"] {
		params.Aliases = aliases
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	entry, err := cli.UpdateCodexEntry(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(entry)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated codex entry %d (%q).\n", entry.ID, entry.Name)
	return 0
}

func (c *RootCmd) PrintCodexUpdateHelp() {
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

func (c *RootCmd) executeCodexDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintCodexDeleteHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	if !*yesFlag {
		fmt.Fprintf(c.Out, "Are you sure you want to delete codex entry %d? [y/N]: ", id)
		scanner := bufio.NewScanner(c.In)
		if scanner.Scan() {
			ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if ans != "y" && ans != "yes" {
				fmt.Fprintln(c.Out, "Deletion cancelled.")
				return 0
			}
		} else {
			fmt.Fprintln(c.Out, "Deletion cancelled.")
			return 0
		}
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.DeleteCodexEntry(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting codex entry: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted codex entry %d.\n", id)
	return 0
}

func (c *RootCmd) PrintCodexDeleteHelp() {
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
