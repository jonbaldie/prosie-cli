package chatcmd

import (
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

func Execute(c *command.Environment, args []string) int {
	return command.Dispatch(c, args, map[string]command.Handler{
		"list":   executeChatList,
		"show":   executeChatShow,
		"send":   executeChatSend,
		"stream": executeChatStream,
		"export": executeChatExport,
		"import": executeChatImport,
		"delete": executeChatDelete,
	}, printChatHelp, "unknown chat command: %s\nRun 'prosie chat --help' for usage.\n")
}

// printChatHelp prints help for the chat command hierarchy.
func printChatHelp(c *command.Environment) {
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
