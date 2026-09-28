package chatcmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func executeChatSend(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	convFlag := fs.String("conversation", "", "Existing conversation ID to continue")
	fs.StringVar(convFlag, "c", "", "Existing conversation ID to continue (shorthand)")
	titleFlag := fs.String("title", "", "Title for newly created conversation")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChatSendHelp)
	}

	bookID, message, err := messageInput(posArgs, *convFlag, "send")
	if err != nil {
		fmt.Fprintf(c.Err, "error: %v\n", err)
		return 1
	}

	target := client.TurnTarget{BookID: bookID, ConversationID: *convFlag, Title: *titleFlag}
	result := sendTurn(c, target, message, nil)
	if result == nil {
		return 1
	}

	if *jsonFlag {
		writeTurnJSON(c, result)
		return 0
	}

	if result.Response.Message != nil {
		fmt.Fprintln(c.Out, result.Response.Message.Content)
	}
	return 0
}

func printChatSendHelp(c *command.Environment) {
	help := `Send a message turn to a novel chat assistant.

Usage:
  prosie chat send <book-id> <message> [flags]
  prosie chat send --conversation <id> <message> [flags]

Flags:
  -c, --conversation string   Existing conversation ID to continue
  -h, --help                  Show help for command
      --json                  Format output as JSON
      --title string          Title for new conversation thread
`
	fmt.Fprint(c.Out, help)
}
