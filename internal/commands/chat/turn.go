package chatcmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
)

// sendTurn sends one message turn and writes the failure to stderr. It returns
// nil when the turn failed.
func sendTurn(c *command.Environment, target client.TurnTarget, message string, onToken func(string)) *client.TurnResult {
	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return nil
	}

	result, err := cli.Conversations().SendTurn(context.Background(), target, message, onToken)
	if err == nil {
		return &result
	}
	if target.ConversationID == "" && !result.Created {
		fmt.Fprintf(c.Err, "error creating conversation: %v\n", err)
		return nil
	}
	fmt.Fprintf(c.Err, "error sending message: %v\n", err)
	if result.Created {
		fmt.Fprintf(c.Err, "conversation %s was created; continue with --conversation %s\n", result.ConversationID, result.ConversationID)
	}
	return nil
}

func writeTurnJSON(c *command.Environment, result *client.TurnResult) {
	convID, _ := strconv.Atoi(result.ConversationID)
	_ = command.WriteJSON(c, map[string]any{
		"conversation_id": convID,
		"message":         result.Response.Message,
		"model":           result.Response.Model,
		"usage":           result.Response.Usage,
	})
}
