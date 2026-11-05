package chatcmd

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func executeChatRetry(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("retry", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", c.JSON, "Output in JSON format")
	posArgs, err := command.ParseFlagsAndArgs(fs, args, command.Args{
		Min: 1, Max: 1, Usage: "prosie chat retry <conversation-id> [flags]", Missing: "conversation ID is required",
	})
	if err != nil {
		return command.FlagError(c, err, printChatRetryHelp)
	}
	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}
	conversationID := strings.TrimSpace(posArgs[0])
	response, err := cli.Conversations().RetryConversation(c.Ctx(), conversationID)
	if err != nil {
		fmt.Fprintf(c.Err, "error retrying conversation: %v\n", err)
		return 1
	}
	if *jsonFlag {
		writeTurnJSON(c, &client.TurnResult{ConversationID: conversationID, Response: response})
	} else if response.Message != nil {
		fmt.Fprintln(c.Out, response.Message.Content)
	}
	return 0
}

func printChatRetryHelp(c *command.Environment) {
	fmt.Fprint(c.Out, `Retry the latest failed conversation turn.

Usage:
  prosie chat retry <conversation-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`)
}
