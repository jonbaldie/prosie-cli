package generatecmd

import (
	"fmt"
	"io"
	"strings"
)

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
