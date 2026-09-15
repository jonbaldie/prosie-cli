package chatcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"os"
)

func executeChatImport(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChatImportHelp)
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

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	conv, err := cli.Conversations().ImportConversation(context.Background(), bookID, filePath)
	if err != nil {
		fmt.Fprintf(c.Err, "error importing conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, conv)
		return 0
	}

	if conv.Title != nil && *conv.Title != "" {
		fmt.Fprintf(c.Out, "Imported conversation %d (%q) into book %s.\n", conv.ID, conv.DisplayTitle(), bookID)
	} else {
		fmt.Fprintf(c.Out, "Imported conversation %d into book %s.\n", conv.ID, bookID)
	}
	return 0
}

func printChatImportHelp(c *command.Environment) {
	help := `Import a JSON conversation file into a book.

Usage:
  prosie chat import <book-id> <file.json> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
