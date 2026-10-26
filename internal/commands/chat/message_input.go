package chatcmd

import (
	"fmt"
	"strings"

	"github.com/jonbaldie/prosie-cli/internal/command"
)

func messageInput(args []string, conversationID, operation string) (string, string, error) {
	// The required arguments depend on --conversation, after flag parsing.
	if conversationID != "" {
		limits := command.Args{Min: 1, Max: -1, Missing: "message is required",
			Usage: fmt.Sprintf("prosie chat %s <message> [flags]", operation)}
		if err := limits.Validate(args); err != nil {
			return "", "", err
		}
		return "", strings.Join(args, " "), nil
	}
	missing := "message is required"
	if len(args) == 0 {
		missing = "book ID and message are required"
	}
	limits := command.Args{Min: 2, Max: -1, Missing: missing,
		Usage: fmt.Sprintf("prosie chat %s <book-id> <message> [flags]", operation)}
	if err := limits.Validate(args); err != nil {
		return "", "", err
	}
	return args[0], strings.Join(args[1:], " "), nil
}
