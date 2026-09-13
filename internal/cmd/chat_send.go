package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func (c *RootCmd) executeChatSend(args []string) int {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	convFlag := fs.String("conversation", "", "Existing conversation ID to continue")
	fs.StringVar(convFlag, "c", "", "Existing conversation ID to continue (shorthand)")
	titleFlag := fs.String("title", "", "Title for newly created conversation")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatSendHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	var convID string
	var bookID string
	var message string

	if *convFlag != "" {
		convID = *convFlag
		if len(posArgs) == 0 {
			fmt.Fprintln(c.Err, "error: message is required. Usage: prosie chat send [book-id] <message> [flags]")
			return 1
		}
		if len(posArgs) == 1 {
			message = posArgs[0]
		} else {
			message = strings.Join(posArgs[1:], " ")
		}
	} else {
		if len(posArgs) == 0 {
			fmt.Fprintln(c.Err, "error: book ID and message are required. Usage: prosie chat send <book-id> <message> [flags]")
			return 1
		}
		if len(posArgs) == 1 {
			fmt.Fprintln(c.Err, "error: message is required. Usage: prosie chat send <book-id> <message> [flags]")
			return 1
		}
		bookID = posArgs[0]
		message = strings.Join(posArgs[1:], " ")
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if convID == "" {
		conv, err := cli.CreateConversation(context.Background(), bookID, *titleFlag)
		if err != nil {
			fmt.Fprintf(c.Err, "error creating conversation: %v\n", err)
			return 1
		}
		convID = strconv.Itoa(conv.ID)
	}

	resp, err := cli.SendChatMessage(context.Background(), convID, message)
	if err != nil {
		fmt.Fprintf(c.Err, "error sending message: %v\n", err)
		return 1
	}

	if *jsonFlag {
		convIDInt, _ := strconv.Atoi(convID)
		_ = c.WriteJSON(map[string]any{
			"conversation_id": convIDInt,
			"message":         resp.Message,
			"model":           resp.Model,
			"usage":           resp.Usage,
		})
		return 0
	}

	if resp.Message != nil {
		fmt.Fprintln(c.Out, resp.Message.Content)
	}
	return 0
}

func (c *RootCmd) PrintChatSendHelp() {
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
