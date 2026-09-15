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

// continueRequest is the parsed and validated input to `generate continue`.
type continueRequest struct {
	chapterID string
	params    client.ContinueParams
	stream    bool
	json      bool
}

// parseContinueArgs parses flags for `generate continue`. A non-nil error
// is a flag-level error to be handled via command.FlagError; a non-empty
// message is a user-facing validation failure with a fixed exit code of 1.
func parseContinueArgs(args []string) (continueRequest, string, error) {
	fs := flag.NewFlagSet("continue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	persistFlag := fs.Bool("persist", true, "Save generated prose to chapter")
	noPersistFlag := fs.Bool("no-persist", false, "Do not save changes to chapter")
	streamFlag := fs.Bool("stream", true, "Stream tokens in real time")
	noStreamFlag := fs.Bool("no-stream", false, "Disable token streaming")
	jsonFlag := fs.Bool("json", false, "Output in JSON format (disables streaming)")
	instructionFlag := fs.String("instruction", "", "Steer the continuation with a final prompt")
	wordsFlag := fs.Int("words", 0, "Target length in words (soft hint)")
	linesFlag := fs.Int("lines", 0, "Hard cap on generated lines")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return continueRequest{}, "", err
	}
	if len(posArgs) == 0 {
		return continueRequest{}, "error: chapter ID is required. Usage: prosie generate continue <chapter-id> [flags]", nil
	}
	if msg := validateLengthFlags(command.VisitedFlags(fs), *wordsFlag, *linesFlag); msg != "" {
		return continueRequest{}, msg, nil
	}

	return continueRequest{
		chapterID: posArgs[0],
		params: client.ContinueParams{
			Persist:     *persistFlag && !*noPersistFlag,
			Instruction: *instructionFlag,
			WordTarget:  *wordsFlag,
			LineLimit:   *linesFlag,
		},
		stream: useStreaming(*streamFlag, *noStreamFlag, *jsonFlag),
		json:   *jsonFlag,
	}, "", nil
}

func validateLengthFlags(visited map[string]bool, words, lines int) string {
	if visited["words"] && words <= 0 {
		return "error: --words must be a positive integer"
	}
	if visited["lines"] && lines <= 0 {
		return "error: --lines must be a positive integer"
	}
	return ""
}

func executeGenerateContinue(c *command.Environment, args []string) int {
	req, msg, err := parseContinueArgs(args)
	if err != nil {
		return command.FlagError(c, err, printGenerateContinueHelp)
	}
	if msg != "" {
		fmt.Fprintln(c.Err, msg)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	ctx := generationContext(c)
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if req.stream {
		return streamContinue(c, cli.Generation(), ctx, req.chapterID, req.params)
	}

	res, err := cli.Generation().Continue(ctx, req.chapterID, req.params)
	if err != nil {
		return continuationError(c, cli.Generation(), ctx, req.chapterID, err)
	}

	if req.json {
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
  -h, --help                 Show help for command
      --instruction string   Steer the continuation with a final prompt
                             (e.g. "Write the closing scene; resolve every thread")
      --words int            Target length in words (soft hint to the model)
      --lines int            Hard cap on generated lines
      --no-persist           Do not save generated prose to chapter
      --persist              Save generated prose to chapter (default true)
      --stream               Stream tokens in real time (default true)
      --no-stream            Disable token streaming
      --json                 Output result as JSON (disables streaming)
`
	fmt.Fprint(c.Out, help)
}

func streamContinue(c *command.Environment, generation *client.Generation, ctx context.Context, id string, params client.ContinueParams) int {
	output := newTokenOutput(c.Out)
	res, err := generation.StreamContinue(ctx, id, params, output.write)
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
