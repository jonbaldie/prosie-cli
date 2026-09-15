package generatecmd

import (
	"context"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"strings"
)

func generationContext(c *command.Environment) context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

type tokenOutput struct {
	out  io.Writer
	last string
}

func (o *tokenOutput) write(token string) { fmt.Fprint(o.out, token); o.last = token }
func (o *tokenOutput) finish(prose string) {
	if !strings.HasSuffix(prose, "\n") && o.last != "" && !strings.HasSuffix(o.last, "\n") {
		fmt.Fprintln(o.out)
	}
}
