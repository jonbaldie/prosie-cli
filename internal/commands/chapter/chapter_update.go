package chaptercmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/jonbaldie/prosie-cli/internal/command"
	"io"
)

func executeChapterUpdate(c *command.Environment, args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Chapter title")
	content := fs.String("content", "", "Chapter prose content")
	summary := fs.String("summary", "", "Chapter summary")
	filePath := fs.String("file", "", "Path to file containing chapter prose")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := command.ParseFlagsAndArgs(fs, args)
	if err != nil {
		return command.FlagError(c, err, printChapterUpdateHelp)
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter update <id> [flags]")
		return 1
	}

	id := posArgs[0]

	params, err := command.ChapterInput(fs, title, content, summary, filePath)
	if err != nil {
		fmt.Fprintf(c.Err, "error reading file: %v\n", err)
		return 1
	}

	cli, err := command.Client(c)
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.Chapters().UpdateChapter(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = command.WriteJSON(c, chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated chapter %d (%q).\n", chapter.ID, chapter.Display().Title)
	return 0
}

func printChapterUpdateHelp(c *command.Environment) {
	help := `Update an existing chapter.

Usage:
  prosie chapter update <id> [flags]

Flags:
      --content string   Chapter prose content
      --file string      Path to file containing chapter prose
  -h, --help             Show help for command
      --json             Format output as JSON
      --summary string   Chapter summary
      --title string     Chapter title
`
	fmt.Fprint(c.Out, help)
}
