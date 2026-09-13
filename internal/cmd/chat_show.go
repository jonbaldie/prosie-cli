package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
)

func (c *RootCmd) executeChatShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatShowHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: conversation ID is required. Usage: prosie chat show <conversation-id> [flags]")
		return 1
	}

	conversationID := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	conv, err := cli.GetConversation(context.Background(), conversationID)
	if err != nil {
		fmt.Fprintf(c.Err, "error showing conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(conv)
		return 0
	}

	fmt.Fprintf(c.Out, "Title:       %s\n", conv.DisplayTitle())
	if conv.Fidelity != "" {
		fmt.Fprintf(c.Out, "Fidelity:    %s\n", conv.Fidelity)
	}
	if conv.Model != nil && *conv.Model != "" {
		fmt.Fprintf(c.Out, "Model:       %s\n", *conv.Model)
	}
	fmt.Fprintf(c.Out, "Book ID:     %d\n", conv.StoryID)
	fmt.Fprintf(c.Out, "Updated:     %s\n", formatTimestamp(conv.UpdatedAt))

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

func (c *RootCmd) PrintChatShowHelp() {
	help := `Show message history for a chat conversation.

Usage:
  prosie chat show <conversation-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
