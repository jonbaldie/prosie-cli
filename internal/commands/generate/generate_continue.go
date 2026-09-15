package generatecmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/client"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func executeGenerateContinue(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("continue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	persistFlag := fs.Bool("persist", true, "Save generated prose to chapter")
	noPersistFlag := fs.Bool("no-persist", false, "Do not save changes to chapter")
	streamFlag := fs.Bool("stream", true, "Stream tokens in real time")
	noStreamFlag := fs.Bool("no-stream", false, "Disable token streaming")
	jsonFlag := fs.Bool("json", false, "Output in JSON format (disables streaming)")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printGenerateContinueHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie generate continue <chapter-id> [flags]")
		return 1
	}

	id := posArgs[0]
	persist := *persistFlag && !*noPersistFlag
	stream := useStreaming(*streamFlag, *noStreamFlag, *jsonFlag)

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := generationContext(c)
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if stream {
		return streamContinue(c, cli.Generation(), ctx, id, persist)
	}

	res, err := cli.Generation().Continue(ctx, id, persist)
	if err != nil {
		return continuationError(c, cli.Generation(), ctx, id, err)
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, res)
		return 0
	}

	fmt.Fprintln(c.Out, res.Prose)
	return 0
}

// printGenerateContinueHelp prints help for the generate continue command.
func printGenerateContinueHelp(c *command.Environment) {
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

func streamContinue(c *command.Environment, generation *client.Generation, ctx context.Context, id string, persist bool) int {
	output := newTokenOutput(c.Out)
	res, err := generation.StreamContinue(ctx, id, persist, output.write)
	if err != nil {
		return continuationError(c, generation, ctx, id, err)
	}
	if res != nil {
		output.finish(res.Prose)
	}
	return 0
}

func continuationError(c *command.Environment, generation *client.Generation, ctx context.Context, id string, err error) int {
	if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = generation.CancelContinue(cancelCtx, id)
		fmt.Fprintln(c.Err, "Generation cancelled.")
		return 1
	}
	fmt.Fprintf(c.Err, "error generating continuation: %v\n", err)
	return 1
}

func useStreaming(stream, noStream, jsonOutput bool) bool { return stream && !noStream && !jsonOutput }
