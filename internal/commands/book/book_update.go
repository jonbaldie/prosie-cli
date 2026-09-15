package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func executeBookUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Book title")
	premise := fs.String("premise", "", "Story premise and outline")
	lore := fs.String("lore", "", "Lore and worldbuilding details")
	characters := fs.String("characters", "", "Characters description")
	targetWords := fs.Int("target-words", 0, "Target word count")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printBookUpdateHelp)
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

	visited := command.VisitedFlags(fs)
	params := client.UpdateBookParams{
		Title:           command.Provided(visited, "title", title),
		Premise:         command.Provided(visited, "premise", premise),
		Lore:            command.Provided(visited, "lore", lore),
		Characters:      command.Provided(visited, "characters", characters),
		TargetWordCount: command.Provided(visited, "target-words", targetWords),
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Books().UpdateBook(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated book %d (%q).\n", book.ID, book.Title)
	return 0
}

func printBookUpdateHelp(c *command.Environment) {
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
