package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
)

func (c *RootCmd) executeChapterList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterListHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie chapter list <book-id> [flags]")
		return 1
	}

	bookID := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapters, err := cli.ListChapters(context.Background(), bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error listing chapters: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapters)
		return 0
	}

	if len(chapters) == 0 {
		fmt.Fprintln(c.Out, "No chapters found.")
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "INDEX\tID\tTITLE\tWORDS\tSUMMARY")
	for i, ch := range chapters {
		fmt.Fprintf(w, "%d\t%d\t%s\t%d\t%s\n", i+1, ch.ID, ch.DisplayTitle(), ch.WordCount, ch.DisplaySummary())
	}
	_ = w.Flush()

	return 0
}

func (c *RootCmd) PrintChapterListHelp() {
	help := `List all chapters in a book.

Usage:
  prosie chapter list <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
