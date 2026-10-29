package chatcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChatDelete(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chat delete <conversation-id> [flags]", Missing: "conversation ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChatDeleteHelp)
	}

	convID := posArgs[0]

	return command.ExecuteDelete(c, command.DeleteOptions{Resource: "conversation", ID: convID, Yes: *yesFlag, JSON: *jsonFlag},
		func(ctx context.Context, cli *client.Client) error {
			return cli.Conversations().DeleteConversation(ctx, convID)
		})
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
