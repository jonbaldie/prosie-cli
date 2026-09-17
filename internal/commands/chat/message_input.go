package chatcmd

import (
	"fmt"
	"strings"
)

func messageInput(args []string, conversationID, operation string) (string, string, error) {
	if conversationID != "" {
		if len(args) == 0 {
			return "", "", fmt.Errorf("message is required. Usage: prosie chat %s <message> [flags]", operation)
		}
		return "", strings.Join(args, " "), nil
	}
	if len(args) == 0 {
		return "", "", fmt.Errorf("book ID and message are required. Usage: prosie chat %s <book-id> <message> [flags]", operation)
	}
	if len(args) == 1 {
		return "", "", fmt.Errorf("message is required. Usage: prosie chat %s <book-id> <message> [flags]", operation)
	}
	return args[0], strings.Join(args[1:], " "), nil
}
