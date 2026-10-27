package bookcmd

import (
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeBookImport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Book title (defaults to filename, or \"Imported Book\" for standard input)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie book import <file.docx> [flags]", Missing: "DOCX file path is required",
	})
	if err != nil {
		return command.FlagError(c, err, printBookImportHelp)
	}

	filePath := posArgs[0]
	upload, closeUpload, ok := command.OpenUpload(c, filePath)
	if !ok {
		return 1
	}
	defer closeUpload()

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	book, err := cli.Books().ImportDocx(c.Ctx(), upload, *title)
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
Use - as the file to read the DOCX data from standard input.

Usage:
  prosie book import <file.docx | -> [flags]

Flags:
  -h, --help           Show help for command
      --json           Format output as JSON
      --title string   Book title (defaults to filename, or "Imported Book" for standard input)
`
	fmt.Fprint(c.Out, help)
}
