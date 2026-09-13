package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
)

func (c *RootCmd) executeChatDelete(args []string) int {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	yesFlag := fs.Bool("yes", false, "Skip confirmation prompt")
	fs.BoolVar(yesFlag, "y", false, "Skip confirmation prompt (shorthand)")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChatDeleteHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: conversation ID is required. Usage: prosie chat delete <conversation-id> [flags]")
		return 1
	}

	convID := posArgs[0]

	if !*yesFlag {
		fmt.Fprintf(c.Out, "Are you sure you want to delete conversation %s? [y/N]: ", convID)
		scanner := bufio.NewScanner(c.In)
		if scanner.Scan() {
			ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if ans != "y" && ans != "yes" {
				fmt.Fprintln(c.Out, "Deletion cancelled.")
				return 0
			}
		} else {
			fmt.Fprintln(c.Out, "Deletion cancelled.")
			return 0
		}
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	if err := cli.DeleteConversation(context.Background(), convID); err != nil {
		fmt.Fprintf(c.Err, "error deleting conversation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(map[string]any{
			"id":      convID,
			"deleted": true,
		})
		return 0
	}

	fmt.Fprintf(c.Out, "Deleted conversation %s.\n", convID)
	return 0
}

func (c *RootCmd) PrintChatDeleteHelp() {
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
