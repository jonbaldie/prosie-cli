package command

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/jonbaldie/prosie-cli/internal/client"
)

// StdinPath is the file argument that reads an upload from standard input.
const StdinPath = "-"

// OpenUpload opens path as an upload stream, or uses standard input when path is StdinPath.
// A standard input upload has no filename. The caller must call the returned close function;
// it closes an opened file and does not close standard input. On failure it writes the error
// to standard error and returns false.
func OpenUpload(c *Environment, path string) (client.UploadFile, func(), bool) {
	if path == StdinPath {
		return client.UploadFile{Reader: c.In}, func() {}, true
	}
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(c.Err, "error: file not found: %s\n", path)
		return client.UploadFile{}, nil, false
	}
	if err != nil {
		fmt.Fprintf(c.Err, "error: failed to open file: %v\n", err)
		return client.UploadFile{}, nil, false
	}
	return client.UploadFile{Filename: path, Reader: file}, func() { _ = file.Close() }, true
}
