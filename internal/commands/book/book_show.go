package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
)

func executeBookShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printBookShowHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie book show <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Books().GetBook(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	return printBook(c, book)
}

func printBookShowHelp(c *command.Environment) {
	help := `Show book details, metadata, and chapters.

Usage:
  prosie book show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func printBook(c *command.Environment, book *client.Book) int {
	fmt.Fprintf(c.Out, "Title:       %s\n", book.Title)
	fmt.Fprintf(c.Out, "ID:          %d\n", book.ID)
	if book.TargetWordCount != nil && *book.TargetWordCount > 0 {
		if book.TargetProgressPercent != nil {
			fmt.Fprintf(c.Out, "Words:       %d / %d (%d%%)\n", book.WordCount, *book.TargetWordCount, *book.TargetProgressPercent)
		} else {
			fmt.Fprintf(c.Out, "Words:       %d / %d\n", book.WordCount, *book.TargetWordCount)
		}
	} else {
		fmt.Fprintf(c.Out, "Words:       %d\n", book.WordCount)
	}

	fmt.Fprintf(c.Out, "Premise:     %s\n", book.Display().Premise)
	fmt.Fprintf(c.Out, "Lore:        %s\n", book.Display().Lore)
	fmt.Fprintf(c.Out, "Characters:  %s\n", book.Display().Characters)
	fmt.Fprintf(c.Out, "Updated:     %s\n", command.FormatTimestamp(book.UpdatedAt))

	fmt.Fprintln(c.Out)
	if len(book.Chapters) == 0 {
		fmt.Fprintln(c.Out, "Chapters: (none)")
	} else {
		fmt.Fprintf(c.Out, "Chapters (%d):\n", len(book.Chapters))
		for i, ch := range book.Chapters {
			wordStr := "words"
			if ch.WordCount == 1 {
				wordStr = "word"
			}
			fmt.Fprintf(c.Out, "  %d. %s (%d %s)\n", i+1, ch.Display().Title, ch.WordCount, wordStr)
		}
	}

	return 0
}
