package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func (c *RootCmd) executeChapterUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	title := fs.String("title", "", "Chapter title")
	content := fs.String("content", "", "Chapter prose content")
	summary := fs.String("summary", "", "Chapter summary")
	filePath := fs.String("file", "", "Path to file containing chapter prose")
	jsonFlag := fs.Bool("json", false, "Output in JSON format")

	posArgs, err := parseFlagsAndArgs(fs, args)
	if err != nil {
		if err == flag.ErrHelp {
			c.PrintChapterUpdateHelp()
			return 0
		}
		fmt.Fprintf(c.Err, "error parsing flags: %v\n", err)
		return 1
	}

	if len(posArgs) == 0 {
		fmt.Fprintln(c.Err, "error: chapter ID is required. Usage: prosie chapter update <id> [flags]")
		return 1
	}

	id := posArgs[0]

	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})

	var contentVal string
	hasContent := false

	if visited["file"] && *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(c.Err, "error reading file: %v\n", err)
			return 1
		}
		contentVal = string(data)
		hasContent = true
	} else if visited["content"] {
		contentVal = *content
		hasContent = true
	}

	params := client.UpdateChapterParams{}
	if visited["title"] {
		params.Title = title
	}
	if hasContent {
		params.Content = &contentVal
	}
	if visited["summary"] {
		params.Summary = summary
	}

	cli, err := c.Client()
	if err != nil {
		fmt.Fprintf(c.Err, "authentication error: %v\n", err)
		return 1
	}

	chapter, err := cli.UpdateChapter(context.Background(), id, params)
	if err != nil {
		fmt.Fprintf(c.Err, "error updating chapter: %v\n", err)
		return 1
	}

	if *jsonFlag {
		_ = c.WriteJSON(chapter)
		return 0
	}

	fmt.Fprintf(c.Out, "Updated chapter %d (%q).\n", chapter.ID, chapter.DisplayTitle())
	return 0
}

func (c *RootCmd) PrintChapterUpdateHelp() {
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
