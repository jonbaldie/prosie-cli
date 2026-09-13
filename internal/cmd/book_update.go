package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func (c *RootCmd) executeBookUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Book title")
	premise := fs.String("premise", "", "Story premise and outline")
	lore := fs.String("lore", "", "Lore and worldbuilding details")
	characters := fs.String("characters", "", "Characters description")
	targetWords := fs.Int("target-words", 0, "Target word count")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintBookUpdateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie book update <id> [flags]")
		return 1
	}

	id, err := strconv.Atoi(posArgs[0])
	if err != nil {
		fmt.Fprintf(c.Err, "error: invalid book ID %q: must be an integer\n", posArgs[0])
		return 1
	}

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	params := client.UpdateBookParams{}
	if visited["title"] {
		params.Title = title
	}
	if visited["premise"] {
		params.Premise = premise
	}
	if visited["lore"] {
		params.Lore = lore
	}
	if visited["characters"] {
		params.Characters = characters
	}
	if visited["target-words"] {
		params.TargetWordCount = targetWords
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.UpdateBook(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(book)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated book %d (%q).\n", book.ID, book.Title)
	return 0
}

func (c *RootCmd) PrintBookUpdateHelp() {
	help := `Update an existing book.

Usage:
  prosie book update <id> [flags]

Flags:
      --characters string     Characters description
  -h, --help                  Show help for command
      --json                  Format output as JSON
      --lore string           Lore and worldbuilding details
      --premise string        Story premise and outline
      --target-words int      Target word count
      --title string          Book title
`
	fmt.Fprint(c.Out, help)
}
