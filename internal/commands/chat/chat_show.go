package chatcmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strings"
)

func executeChatShow(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChatShowHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: conversation ID is required. Usage: prosie chat show <conversation-id> [flags]")
		return 1
	}

	conversationID := posArgs[0]

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	conv, err := cli.Conversations().GetConversation(context.Background(), conversationID)
	if err != nil {
		fmt.Fprintf(c.Err, "error showing conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, conv)
		return 0
	}

	return printConversation(c, conv)
}

func printChatShowHelp(c *command.Environment) {
	help := `Show message history for a chat conversation.

Usage:
  prosie chat show <conversation-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}

func printConversation(c *command.Environment, conv *client.Conversation) int {
	fmt.Fprintf(c.Out, "Title:       %s\n", conv.DisplayTitle())
	if conv.Fidelity != "" {
		fmt.Fprintf(c.Out, "Fidelity:    %s\n", conv.Fidelity)
	}
	if conv.Model != nil && *conv.Model != "" {
		fmt.Fprintf(c.Out, "Model:       %s\n", *conv.Model)
	}
	fmt.Fprintf(c.Out, "Book ID:     %d\n", conv.StoryID)
	fmt.Fprintf(c.Out, "Updated:     %s\n", command.FormatTimestamp(conv.UpdatedAt))

	if len(conv.Messages) == 0 {
		fmt.Fprintln(c.Out, "\nNo messages found.")
		return 0
	}

	fmt.Fprintf(c.Out, "\nMessages (%d):\n", len(conv.Messages))
	for _, m := range conv.Messages {
		speaker := "You"
		if m.Role == "assistant" {
			speaker = "Assistant"
		} else if m.Role != "user" && m.Role != "" {
			speaker = strings.ToUpper(m.Role[:1]) + m.Role[1:]
		}
		fmt.Fprintf(c.Out, "\n[%s]\n%s\n", speaker, m.Content)
	}

	return 0
}
