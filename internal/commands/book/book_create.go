package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func executeBookCreate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Book title (required)")
	premise := fs.String("premise", "", "Story premise and outline")
	lore := fs.String("lore", "", "Lore and worldbuilding details")
	characters := fs.String("characters", "", "Characters description")
	targetWords := fs.Int("target-words", 0, "Target word count")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	if _, err := command.ParseFlagsAndArgs(fs, args); err != nil {
		return command.FlagError(c, err, printBookCreateHelp)
	}

	if *title == "" {
		fmt.Fprintln(c.Err, "error: --title is required")
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	params := client.CreateBookParams{
		Title: *title, Premise: command.OptionalText(*premise), Lore: command.OptionalText(*lore),
		Characters: command.OptionalText(*characters), TargetWordCount: command.WordTarget(*targetWords),
	}

	book, err := cli.Books().CreateBook(context.Background(), params)
	if err != nil {
		fmt.Fprintf(c.Err, "error creating book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	fmt.Fprintf(c.Out, "Created book %q (ID: %d).\n", book.Title, book.ID)
	return 0
}

func printBookCreateHelp(c *command.Environment) {
	help := `Create a new book.

Usage:
  prosie book create --title <title> [flags]

Flags:
      --characters string     Characters description
  -h, --help                  Show help for command
      --json                  Format output as JSON
      --lore string           Lore and worldbuilding details
      --premise string        Story premise and outline
      --target-words int      Target word count
      --title string          Book title (required)
`
	fmt.Fprint(c.Out, help)
}
