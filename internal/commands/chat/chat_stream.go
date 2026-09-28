package chatcmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func executeChatStream(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("stream", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	convFlag := fs.String("conversation", "", "Existing conversation ID to continue")
	fs.StringVar(convFlag, "c", "", "Existing conversation ID to continue (shorthand)")
	titleFlag := fs.String("title", "", "Title for newly created conversation")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChatStreamHelp)
	}

	bookID, message, err := messageInput(posArgs, *convFlag, "stream")
	if err != nil {
		fmt.Fprintf(c.Err, "error: %v\n", err)
		return 1
	}

	streamed := false
	onToken := func(token string) {
		streamed = true
		fmt.Fprint(c.Out, token)
	}
	if *jsonFlag {
		onToken = func(token string) {}
	}

	target := client.TurnTarget{BookID: bookID, ConversationID: *convFlag, Title: *titleFlag}
	result := sendTurn(c, target, message, onToken)
	if result == nil {
		if streamed {
			fmt.Fprintln(c.Out)
		}
		return 1
	}

	if *jsonFlag {
		writeTurnJSON(c, result)
		return 0
	}

	fmt.Fprintln(c.Out)
	return 0
}

func printChatStreamHelp(c *command.Environment) {
	help := `Stream assistant response tokens in real time.

Usage:
  prosie chat stream <book-id> <message> [flags]
  prosie chat stream --conversation <id> <message> [flags]

Flags:
  -c, --conversation string   Existing conversation ID to continue
  -h, --help                  Show help for command
      --json                  Format output as JSON (disables token streaming)
      --title string          Title for new conversation thread
`
	fmt.Fprint(c.Out, help)
}
