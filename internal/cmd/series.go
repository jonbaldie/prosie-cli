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

func (c *RootCmd) executeSeries(args []string) int {
	if len(args) == 0 {
		c.PrintSeriesHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintSeriesHelp()
		return 0
	case "list":
		return c.executeSeriesList(subArgs)
	case "show":
		return c.executeSeriesShow(subArgs)
	case "create":
		return c.executeSeriesCreate(subArgs)
	case "update":
		return c.executeSeriesUpdate(subArgs)
	case "delete":
		return c.executeSeriesDelete(subArgs)
	case "attach":
		return c.executeSeriesAttach(subArgs)
	case "detach":
		return c.executeSeriesDetach(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown series command: %s\nRun 'prosie series --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintSeriesHelp prints help for the series command hierarchy.
func (c *RootCmd) PrintSeriesHelp() {
	help := `Manage book series and shared universe lore.

Usage:
  prosie series <command> [flags]

Available Commands:
  list        List all series and member books
  show        Show series details, books, and shared codex entries
  create      Create a new series
  update      Update series details
  delete      Delete a series
  attach      Attach a book to a series
  detach      Remove a book from a series

Flags:
  -h, --help   Show help for command

Use "prosie series <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := parseFlagsAndArgs(fs, args); err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesListHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	seriesList, err := cli.ListSeries(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error listing series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(seriesList)
		return 0
	}

	if len(seriesList) == 0 {
		fmt.Fprintln(c.Out, "No series found.")
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tBOOKS\tUPDATED")
	for _, s := range seriesList {
		booksStr := "-"
		if len(s.Books) > 0 {
			var titles []string
			for _, b := range s.Books {
				titles = append(titles, b.Title)
			}
			booksStr = strings.Join(titles, ", ")
		} else if len(s.StoryIDs) > 0 {
			var ids []string
			for _, id := range s.StoryIDs {
				ids = append(ids, fmt.Sprintf("Book %d", id))
			}
			booksStr = strings.Join(ids, ", ")
		}
		updatedStr := formatTimestamp(s.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.ID, s.DisplayTitle(), booksStr, updatedStr)
	}
	_ = w.Flush()

	return 0
}

func (c *RootCmd) PrintSeriesListHelp() {
	help := `List all series and member books.

Usage:
  prosie series list [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesShowHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: series ID is required. Usage: prosie series show <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid series ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	series, err := cli.GetSeries(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(series)
		return 0
	}

	fmt.Fprintf(c.Out, "Title:       %s\n", series.DisplayTitle())
	fmt.Fprintf(c.Out, "ID:          %d\n", series.ID)
	fmt.Fprintf(c.Out, "Description: %s\n", series.DisplayDescription())
	fmt.Fprintf(c.Out, "Updated:     %s\n", formatTimestamp(series.UpdatedAt))

	fmt.Fprintln(c.Out)
	if len(series.Books) == 0 {
		fmt.Fprintln(c.Out, "Books: (none)")
	} else {
		fmt.Fprintf(c.Out, "Books (%d):\n", len(series.Books))
		for i, b := range series.Books {
			wordStr := "words"
			if b.WordCount == 1 {
				wordStr = "word"
			}
			fmt.Fprintf(c.Out, "  %d. %s (ID: %d, %d %s)\n", i+1, b.Title, b.ID, b.WordCount, wordStr)
		}
	}

	fmt.Fprintln(c.Out)
	if len(series.CodexEntries) == 0 {
		fmt.Fprintln(c.Out, "Shared Codex Entries: (none)")
	} else {
		fmt.Fprintf(c.Out, "Shared Codex Entries (%d):\n", len(series.CodexEntries))
		for i, entry := range series.CodexEntries {
			fmt.Fprintf(c.Out, "  %d. [%s] %s (ID: %d)\n", i+1, strings.ToUpper(entry.DisplayType()), entry.Name, entry.ID)
		}
	}

	return 0
}

func (c *RootCmd) PrintSeriesShowHelp() {
	help := `Show series details, member books, and shared codex entries.

Usage:
  prosie series show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Series title")
	name := fs.String("name", "", "Series name (alias for title)")
	description := fs.String("description", "", "Series description and premise")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := parseFlagsAndArgs(fs, args); err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesCreateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	finalTitle := *title
	if finalTitle == "" {
		finalTitle = *name
	}

	if strings.TrimSpace(finalTitle) == "" {
		fmt.Fprintln(c.Err, "error: --title is required")
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	params := client.CreateSeriesParams{
		Title: finalTitle,
		Name:  finalTitle,
	}
	if strings.TrimSpace(*description) != "" {
		params.Description = description
	}

	series, err := cli.CreateSeries(context.Background(), params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(series)
		return 0
	}

	fmt.Fprintf(c.Out, "Created series %q (ID: %d).\n", series.DisplayTitle(), series.ID)
	return 0
}

func (c *RootCmd) PrintSeriesCreateHelp() {
	help := `Create a new series.

Usage:
  prosie series create --title <title> [flags]

Flags:
      --description string   Series description and premise
  -h, --help                 Show help for command
      --json                 Format output as JSON
      --name string          Series name (alias for title)
      --title string         Series title (required)
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Series title")
	name := fs.String("name", "", "Series name (alias for title)")
	description := fs.String("description", "", "Series description")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesUpdateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: series ID is required. Usage: prosie series update <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid series ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	params := client.UpdateSeriesParams{}
	if visited["title"] {
		params.Title = title
		params.Name = title
	} else if visited["name"] {
		params.Title = name
		params.Name = name
	}
	if visited["description"] {
		params.Description = description
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	series, err := cli.UpdateSeries(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(series)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated series %d (%q).\n", series.ID, series.DisplayTitle())
	return 0
}

func (c *RootCmd) PrintSeriesUpdateHelp() {
	help := `Update series title and description.

Usage:
  prosie series update <id> [flags]

Flags:
      --description string   Series description
  -h, --help                 Show help for command
      --json                 Format output as JSON
      --name string          Series name (alias for title)
      --title string         Series title
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesDeleteHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: series ID is required. Usage: prosie series delete <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid series ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	if !*yesFlag {
		fmt.Fprintf(c.Out, "Are you sure you want to delete series %d? [y/N]: ", id)
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

	if err := cli.DeleteSeries(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted series %d.\n", id)
	return 0
}

func (c *RootCmd) PrintSeriesDeleteHelp() {
	help := `Delete a series.

Usage:
  prosie series delete <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
  -y, --yes    Skip confirmation prompt
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesAttach(args []string) int {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesAttachHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) < 2 {
		fmt.Fprintln(c.Err, "error: series ID and book ID are required. Usage: prosie series attach <series-id> <book-id> [flags]")
		return 1
	}

	seriesID, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid series ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	bookID, err := strconv.Atoi(posArgs[1])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[1])
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.AttachSeriesBook(context.Background(), seriesID, bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error attaching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(book)
		return 0
	}

	fmt.Fprintf(c.Out, "Attached book %d (%q) to series %d.\n", book.ID, book.Title, seriesID)
	return 0
}

func (c *RootCmd) PrintSeriesAttachHelp() {
	help := `Attach a book to a series.

Usage:
  prosie series attach <series-id> <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func (c *RootCmd) executeSeriesDetach(args []string) int {
	fs := flag.NewFlagSet("detach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintSeriesDetachHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) < 2 {
		fmt.Fprintln(c.Err, "error: series ID and book ID are required. Usage: prosie series detach <series-id> <book-id> [flags]")
		return 1
	}

	seriesID, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid series ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	bookID, err := strconv.Atoi(posArgs[1])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[1])
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.DetachSeriesBook(context.Background(), seriesID, bookID); err != nil {
		fmt.Fprintf(c.Err, "error detaching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"series_id": seriesID,
			"book_id":   bookID,
			"detached":  true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Detached book %d from series %d.\n", bookID, seriesID)
	return 0
}

func (c *RootCmd) PrintSeriesDetachHelp() {
	help := `Remove a book from a series.

Usage:
  prosie series detach <series-id> <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
