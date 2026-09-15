package bookcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"os"
)

func executeBookImport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Book title (defaults to filename)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printBookImportHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: DOCX file path is required. Usage: prosie book import <file.docx> [flags]")
		return 1
	}

	filePath := posArgs[0]
	if _, err := os.Stat(filePath); err != nil {
		fmt.Fprintf(c.Err, "error: file not found: %s\n", filePath)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Books().ImportDocx(context.Background(), filePath, *title)
	if err != nil {
		fmt.Fprintf(c.Err, "error importing book: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, book)
		return 0
	}

	fmt.Fprintf(c.Out, "Imported book %q (ID: %d) with %d chapters.\n", book.Title, book.ID, len(book.Chapters))
	return 0
}

func printBookImportHelp(c *command.Environment) {
	help := `Import a DOCX manuscript file to create a new book.

Usage:
  prosie book import <file.docx> [flags]

Flags:
  -h, --help           Show help for command
      --json           Format output as JSON
      --title string   Book title (defaults to filename)
`
	fmt.Fprint(c.Out, help)
}
