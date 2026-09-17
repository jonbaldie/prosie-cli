package chatcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strconv"
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

	convID := *convFlag
	bookID, message, err := messageInput(posArgs, convID, "stream")
	if err != nil {
		fmt.Fprintf(c.Err, "error: %v\n", err)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if convID == "" {
		conv, err := cli.Conversations().CreateConversation(context.Background(), bookID, *titleFlag)
		if err != nil {
			fmt.Fprintf(c.Err, "error creating conversation: %v\n", err)
			return 1
		}
		convID = strconv.Itoa(conv.ID)
	}

	onToken := func(token string) {
		fmt.Fprint(c.Out, token)
	}
	if *jsonFlag {
		onToken = func(token string) {}
	}

	resp, err := cli.Conversations().StreamChatMessage(context.Background(), convID, message, onToken)
	if err != nil {
		fmt.Fprintf(c.Err, "\nerror streaming message: %v\n", err)
		return 1
	}

	if *jsonFlag {
		convIDInt, _ := strconv.Atoi(convID)
		_ = command.WriteJSON(c, map[string]any{
			"conversation_id": convIDInt,
			"message":         resp.Message,
			"model":           resp.Model,
			"usage":           resp.Usage,
		})
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
