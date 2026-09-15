package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strings"
)

func executeChapterReorder(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("reorder", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChapterReorderHelp)
	}

	if len(posArgs) < 2 {
		fmt.Fprintln(c.Err, "error: book ID and chapter order are required. Usage: prosie chapter reorder <book-id> <id1,id2,...> [flags]")
		return 1
	}

	bookID := posArgs[0]

	order := chapterOrder(posArgs[1:])

	if len(order) == 0 {
		fmt.Fprintln(c.Err, "error: at least one chapter ID is required for reordering")
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapters, err := cli.Chapters().ReorderChapters(context.Background(), bookID, order)
	if err != nil {
		fmt.Fprintf(c.Err, "error reordering chapters: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapters)
		return 0
	}

	fmt.Fprintf(c.Out, "Reordered %d chapters for book %s.\n", len(chapters), bookID)
	return 0
}

func printChapterReorderHelp(c *command.Environment) {
	help := `Reorder chapters in a book.

Usage:
  prosie chapter reorder <book-id> <id1,id2,...> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func chapterOrder(args []string) []string {
	var order []string
	for _, arg := range args {
		for _, part := range strings.Split(arg, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				order = append(order, trimmed)
			}
		}
	}

	return order
}
