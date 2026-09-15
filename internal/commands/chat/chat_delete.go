package chatcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChatDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChatDeleteHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: conversation ID is required. Usage: prosie chat delete <conversation-id> [flags]")
		return 1
	}

	convID := posArgs[0]

	if !command.ConfirmDeletion(c, *yesFlag, "conversation", convID) {
		return 0
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.Conversations().DeleteConversation(context.Background(), convID); err != nil {
		fmt.Fprintf(c.Err, "error deleting conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, map[string]any{
			"id":      convID,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted conversation %s.\n", convID)
	return 0
}

func printChatDeleteHelp(c *command.Environment) {
	help := `Delete a chat conversation thread.

Usage:
  prosie chat delete <conversation-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
  -y, --yes    Skip confirmation prompt
`
	fmt.Fprint(c.Out, help)
}
