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
	write  func(string)
	finish func(string)
}

func newTokenOutput(out io.Writer) tokenOutput {
	last := ""
	return tokenOutput{
		write: func(token string) {
			fmt.Fprint(out, token)
			last = token
		},
		finish: func(prose string) {
			if !strings.HasSuffix(prose, "\n") && last != "" && !strings.HasSuffix(last, "\n") {
				fmt.Fprintln(out)
			}
		},
	}
}
