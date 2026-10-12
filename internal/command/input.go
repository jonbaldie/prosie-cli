package command

import (
	"flag"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

func VisitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	return visited
}

// Provided preserves an explicitly empty flag value when building a patch.
func Provided[T any](visited map[string]bool, name string, value *T) *T {
	if visited[name] {
		return value
	}
	return nil
}

func OptionalText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func WordTarget(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func ChapterInput(c *Environment, fs *flag.FlagSet, title, content, summary, filePath *string) (client.UpdateChapterParams, error) {
	visited := VisitedFlags(fs)
	params := client.UpdateChapterParams{
		Name:    Provided(visited, "title", title),
		Summary: Provided(visited, "summary", summary),
		Content: Provided(visited, "content", content),
	}
	if visited["file"] && *filePath != "" {
		text, err := ReadContent(c, *content, *filePath, "content")
		if err != nil {
			return params, err
		}
		params.Content = &text
	}
	return params, nil
}
