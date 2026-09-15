package seriescmd

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
	return command.Dispatch(c, args, map[string]command.Handler{
		"list":   executeSeriesList,
		"show":   executeSeriesShow,
		"create": executeSeriesCreate,
		"update": executeSeriesUpdate,
		"delete": executeSeriesDelete,
		"attach": executeSeriesAttach,
		"detach": executeSeriesDetach,
	}, printSeriesHelp, "unknown series command: %s\nRun 'prosie series --help' for usage.\n")
}

// printSeriesHelp prints help for the series command hierarchy.
func printSeriesHelp(c *command.Environment) {
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

func executeSeriesList(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := command.ParseFlagsAndArgs(fs, args); err != nil {
		return command.FlagError(c, err, printSeriesListHelp)
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	seriesList, err := cli.Series().ListSeries(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error listing series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, seriesList)
		return 0
	}

	return printSeriesList(c, seriesList)
}

func printSeriesListHelp(c *command.Environment) {
	help := `List all series and member books.

Usage:
  prosie series list [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func executeSeriesShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printSeriesShowHelp)
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

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	series, err := cli.Series().GetSeries(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, series)
		return 0
	}

	return printSeries(c, series)
}

func printSeriesShowHelp(c *command.Environment) {
	help := `Show series details, member books, and shared codex entries.

Usage:
  prosie series show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func executeSeriesCreate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Series title")
	name := fs.String("name", "", "Series name (alias for title)")
	description := fs.String("description", "", "Series description and premise")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := command.ParseFlagsAndArgs(fs, args); err != nil {
		return command.FlagError(c, err, printSeriesCreateHelp)
	}

	finalTitle := *title
	if finalTitle == "" {
		finalTitle = *name
	}

	if strings.TrimSpace(finalTitle) == "" {
		fmt.Fprintln(c.Err, "error: --title is required")
		return 1
	}

	cli, err := command.Client(c)
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

	series, err := cli.Series().CreateSeries(context.Background(), params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, series)
		return 0
	}

	fmt.Fprintf(c.Out, "Created series %q (ID: %d).\n", series.Display().Title, series.ID)
	return 0
}

func printSeriesCreateHelp(c *command.Environment) {
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

func executeSeriesUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Series title")
	name := fs.String("name", "", "Series name (alias for title)")
	description := fs.String("description", "", "Series description")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printSeriesUpdateHelp)
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
	params.Description = command.Provided(visited, "description", description)

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	series, err := cli.Series().UpdateSeries(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, series)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated series %d (%q).\n", series.ID, series.Display().Title)
	return 0
}

func printSeriesUpdateHelp(c *command.Environment) {
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

func executeSeriesDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printSeriesDeleteHelp)
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

	if !command.ConfirmDeletion(c, *yesFlag, "series", id) {
		return 0
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Series().DeleteSeries(context.Background(), id); err != nil {
		fmt.Fprintf(c.Err, "error deleting series: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]any{
			"id":      id,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted series %d.\n", id)
	return 0
}

func printSeriesDeleteHelp(c *command.Environment) {
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

func executeSeriesAttach(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("attach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printSeriesAttachHelp)
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

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Series().AttachSeriesBook(context.Background(), seriesID, bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error attaching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	fmt.Fprintf(c.Out, "Attached book %d (%q) to series %d.\n", book.ID, book.Title, seriesID)
	return 0
}

func printSeriesAttachHelp(c *command.Environment) {
	help := `Attach a book to a series.

Usage:
  prosie series attach <series-id> <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func executeSeriesDetach(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("detach", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printSeriesDetachHelp)
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

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Series().DetachSeriesBook(context.Background(), seriesID, bookID); err != nil {
		fmt.Fprintf(c.Err, "error detaching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]any{
			"series_id": seriesID,
			"book_id":   bookID,
			"detached":  true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Detached book %d from series %d.\n", bookID, seriesID)
	return 0
}

func printSeriesDetachHelp(c *command.Environment) {
	help := `Remove a book from a series.

Usage:
  prosie series detach <series-id> <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func printSeries(c *command.Environment, series *client.Series) int {
	fmt.Fprintf(c.Out, "Title:       %s\n", series.Display().Title)
	fmt.Fprintf(c.Out, "ID:          %d\n", series.ID)
	fmt.Fprintf(c.Out, "Description: %s\n", series.Display().Description)
	fmt.Fprintf(c.Out, "Updated:     %s\n", command.FormatTimestamp(series.UpdatedAt))

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
			fmt.Fprintf(c.Out, "  %d. [%s] %s (ID: %d)\n", i+1, strings.ToUpper(entry.Display().Type), entry.Name, entry.ID)
		}
	}

	return 0
}

func printSeriesList(c *command.Environment, seriesList []client.Series) int {
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
		updatedStr := command.FormatTimestamp(s.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.ID, s.Display().Title, booksStr, updatedStr)
	}
	_ = w.Flush()

	return 0
}
