package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

func (c *RootCmd) executeBookList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := parseFlagsAndArgs(fs, args); err != nil {
		if err == flag.ErrHelp {
			c.PrintBookListHelp()
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

	books, err := cli.ListBooks(context.Background())
	if err != nil {
		fmt.Fprintf(c.Err, "error listing books: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(books)
		return 0
	}

	if len(books) == 0 {
		fmt.Fprintln(c.Out, "No books found.")
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tWORDS\tTARGET\tUPDATED")
	for _, b := range books {
		targetStr := "-"
		if b.TargetWordCount != nil && *b.TargetWordCount > 0 {
			targetStr = fmt.Sprintf("%d", *b.TargetWordCount)
		}
		updatedStr := formatTimestamp(b.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\n", b.ID, b.Title, b.WordCount, targetStr, updatedStr)
	}
	_ = w.Flush()

	return 0
}

func (c *RootCmd) PrintBookListHelp() {
	help := `List all books in your library.

Usage:
  prosie book list [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func formatTimestamp(raw string) string {
	if raw == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000Z", raw)
	}
	if err != nil {
		return raw
	}
	return t.Format("2006-01-02 15:04")
}
