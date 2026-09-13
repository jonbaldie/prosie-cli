package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
)

func (c *RootCmd) executeChatImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatImportHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) < 2 {
		fmt.Fprintln(c.Err, "error: book ID and file path are required. Usage: prosie chat import <book-id> <file.json> [flags]")
		return 1
	}

	bookID := posArgs[0]
	filePath := posArgs[1]

	if _, err := os.Stat(filePath); err != nil {
		fmt.Fprintf(c.Err, "error: file not found: %s\n", filePath)
		return 1
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	conv, err := cli.ImportConversation(context.Background(), bookID, filePath)
	if err != nil {
		fmt.Fprintf(c.Err, "error importing conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(conv)
		return 0
	}

	if conv.Title != nil && *conv.Title != "" {
		fmt.Fprintf(c.Out, "Imported conversation %d (%q) into book %s.\n", conv.ID, conv.DisplayTitle(), bookID)
	} else {
		fmt.Fprintf(c.Out, "Imported conversation %d into book %s.\n", conv.ID, bookID)
	}
	return 0
}

func (c *RootCmd) PrintChatImportHelp() {
	help := `Import a JSON conversation file into a book.

Usage:
  prosie chat import <book-id> <file.json> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
