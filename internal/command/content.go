package command

import (
	"fmt"
	"io"
	"os"
)

// ReadContent reads inline prose, a disk file, or standard input for StdinPath.
// It preserves the text exactly and rejects non-empty inline text with a file path.
// The label identifies the content in validation and read errors.
func ReadContent(c *Environment, inline, filePath, label string) (string, error) {
	if filePath == "" {
		return inline, nil
	}
	if inline != "" {
		return "", fmt.Errorf("error: use either inline %s or a %s file, not both", label, label)
	}
	var data []byte
	var err error
	if filePath == StdinPath {
		data, err = io.ReadAll(c.In)
	} else {
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return "", fmt.Errorf("error reading %s file: %w", label, err)
	}
	return string(data), nil
}
