package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
)

func (c *RootCmd) executeChapterReorder(args []string) int {
	fs := flag.NewFlagSet("reorder", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterReorderHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) < 2 {
		fmt.Fprintln(c.Err, "error: book ID and chapter order are required. Usage: prosie chapter reorder <book-id> <id1,id2,...> [flags]")
		return 1
	}

	bookID := posArgs[0]

	var order []string
	for _, arg := range posArgs[1:] {
		for _, part := range strings.Split(arg, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				order = append(order, trimmed)
			}
		}
	}

	if len(order) == 0 {
		fmt.Fprintln(c.Err, "error: at least one chapter ID is required for reordering")
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapters, err := cli.ReorderChapters(context.Background(), bookID, order)
	if err != nil {
		fmt.Fprintf(c.Err, "error reordering chapters: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapters)
		return 0
	}

	fmt.Fprintf(c.Out, "Reordered %d chapters for book %s.\n", len(chapters), bookID)
	return 0
}

func (c *RootCmd) PrintChapterReorderHelp() {
	help := `Reorder chapters in a book.

Usage:
  prosie chapter reorder <book-id> <id1,id2,...> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
