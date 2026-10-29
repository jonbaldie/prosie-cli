package command

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

type DeleteOptions struct {
	Resource string
	ID       any
	Yes      bool
	JSON     bool
}

func ExecuteDelete(c *Environment, opts DeleteOptions, deleteFn func(context.Context, *client.Client) error) int {
	if !ConfirmDeletion(c, opts.Yes, opts.Resource, opts.ID) {
		return 0
	}
	cli, err := Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}
	if err := deleteFn(c.Ctx(), cli); err != nil {
		fmt.Fprintf(c.Err, "error deleting %s: %v\n", opts.Resource, err)
		return 1
	}
	if opts.JSON {
		id := opts.ID
		if s, ok := id.(string); ok {
			if n, err := strconv.Atoi(s); err == nil {
				id = n
			}
		}
		_ = WriteJSON(c, map[string]any{"id": id, "deleted": true})
		return 0
	}
	fmt.Fprintf(c.Out, "Deleted %s %v.\n", opts.Resource, opts.ID)
	return 0
}
