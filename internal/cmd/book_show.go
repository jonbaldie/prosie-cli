package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
)

func (c *RootCmd) executeBookShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintBookShowHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
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

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.GetBook(context.Background(), id)
	if err != nil {
		fmt.Fprintf(c.Err, "error fetching book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(book)
		return 0
	}

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

	fmt.Fprintf(c.Out, "Premise:     %s\n", book.DisplayPremise())
	fmt.Fprintf(c.Out, "Lore:        %s\n", book.DisplayLore())
	fmt.Fprintf(c.Out, "Characters:  %s\n", book.DisplayCharacters())
	fmt.Fprintf(c.Out, "Updated:     %s\n", formatTimestamp(book.UpdatedAt))

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
			fmt.Fprintf(c.Out, "  %d. %s (%d %s)\n", i+1, ch.DisplayTitle(), ch.WordCount, wordStr)
		}
	}

	return 0
}

func (c *RootCmd) PrintBookShowHelp() {
	help := `Show book details, metadata, and chapters.

Usage:
  prosie book show <id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
