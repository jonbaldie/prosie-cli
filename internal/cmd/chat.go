package cmd

import (
	"fmt"
)

func (c *RootCmd) executeChat(args []string) int {
	if len(args) == 0 {
		c.PrintChatHelp()
		return 0
	}

	subCmd := args[0]
	subArgs := args[1:]

	switch subCmd {
	case "help", "--help", "-h":
		c.PrintChatHelp()
		return 0
	case "list":
		return c.executeChatList(subArgs)
	case "show":
		return c.executeChatShow(subArgs)
	case "send":
		return c.executeChatSend(subArgs)
	case "stream":
		return c.executeChatStream(subArgs)
	case "export":
		return c.executeChatExport(subArgs)
	case "import":
		return c.executeChatImport(subArgs)
	case "delete":
		return c.executeChatDelete(subArgs)
	default:
		fmt.Fprintf(c.Err, "unknown chat command: %s\nRun 'prosie chat --help' for usage.\n", subCmd)
		return 1
	}
}

// PrintChatHelp prints help for the chat command hierarchy.
func (c *RootCmd) PrintChatHelp() {
	help := `Manage novel chat conversations grounded in your manuscript.

Usage:
  prosie chat <command> [flags]

Available Commands:
  list        List chat conversations for a book
  show        Show conversation history and messages
  send        Send a message to a novel chat conversation
  stream      Stream assistant response tokens in real time
  export      Export a chat conversation to JSON
  import      Import a JSON conversation into a book
  delete      Delete a chat conversation thread

Flags:
  -h, --help   Show help for command

Use "prosie chat <command> --help" for more information about a command.
`
	fmt.Fprint(c.Out, help)
}
