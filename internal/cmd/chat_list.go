package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
)

func (c *RootCmd) executeChatList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatListHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: book ID is required. Usage: prosie chat list <book-id> [flags]")
		return 1
	}

	bookID := posArgs[0]

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	conversations, err := cli.ListConversations(context.Background(), bookID)
	if err != nil {
		fmt.Fprintf(c.Err, "error listing conversations: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(conversations)
		return 0
	}

	if len(conversations) == 0 {
		fmt.Fprintln(c.Out, "No conversations found.")
		return 0
	}

	w := tabwriter.NewWriter(c.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tFIDELITY\tEXPIRES\tUPDATED")
	for _, conv := range conversations {
		expiresStr := "-"
		if conv.ExpiresInDays != nil {
			expiresStr = fmt.Sprintf("%dd", *conv.ExpiresInDays)
		}
		updatedStr := formatTimestamp(conv.UpdatedAt)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", conv.ID, conv.DisplayTitle(), conv.Fidelity, expiresStr, updatedStr)
	}
	_ = w.Flush()

	return 0
}

func (c *RootCmd) PrintChatListHelp() {
	help := `List chat conversations for a book.

Usage:
  prosie chat list <book-id> [flags]

Flags:
  -h, --help   Show help for command
      --json   Format output as JSON
`
	fmt.Fprint(c.Out, help)
}
