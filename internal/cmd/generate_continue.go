package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func (c *RootCmd) executeGenerateContinue(args []string) int {
	fs := flag.NewFlagSet("continue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	persistFlag := fs.Bool("persist", true, "Save generated prose to chapter")
	noPersistFlag := fs.Bool("no-persist", false, "Do not save changes to chapter")
	streamFlag := fs.Bool("stream", true, "Stream tokens in real time")
	noStreamFlag := fs.Bool("no-stream", false, "Disable token streaming")
	jsonFlag := fs.Bool("json", false, "Output in JSON format (disables streaming)")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintGenerateContinueHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate continue <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]
	persist := *persistFlag && !*noPersistFlag
	stream := *streamFlag && !*noStreamFlag && !*jsonFlag

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if stream {
		var lastToken string
		onToken := func(token string) {
			fmt.Fprint(c.Out, token)
			lastToken = token
		}

		res, err := cli.StreamContinue(ctx, id, persist, onToken)
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = cli.CancelContinue(cancelCtx, id)
				fmt.Fprintln(c.Err, "Generation cancelled.")
				return 1
			}
			fmt.Fprintf(c.Err, "error generating continuation: %v\n", err)
			return 1
		}

		if res != nil && !strings.HasSuffix(res.Prose, "\n") && lastToken != "" && !strings.HasSuffix(lastToken, "\n") {
			fmt.Fprintln(c.Out)
		}
		return 0
	}

	res, err := cli.Continue(ctx, id, persist)
	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = cli.CancelContinue(cancelCtx, id)
			fmt.Fprintln(c.Err, "Generation cancelled.")
			return 1
		}
		fmt.Fprintf(c.Err, "error generating continuation: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Prose)
	return 0
}

// PrintGenerateContinueHelp prints help for the generate continue command.
func (c *RootCmd) PrintGenerateContinueHelp() {
	help := `Generate AI prose continuation for a chapter.

Usage:
  prosie generate continue <chapter-id> [flags]

Flags:
  -h, --help         Show help for command
      --no-persist   Do not save generated prose to chapter
      --persist      Save generated prose to chapter (default true)
      --stream       Stream tokens in real time (default true)
      --no-stream    Disable token streaming
      --json         Output result as JSON (disables streaming)
`
	fmt.Fprint(c.Out, help)
}
