// Package filewrite provides safe replacement of saved files.
package filewrite

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Replace writes a complete file before replacing path. It follows symbolic
// links, keeps existing regular-file permissions, and uses mode for new files.
// Non-regular destinations are written in place. The temporary file is synced
// before publication; the directory is not synced for power-loss protection.
func Replace(path string, data []byte, mode fs.FileMode) error {
	// Stat follows device links such as Linux /proc/self/fd links. These must
	// be opened through the original path, not resolved as ordinary links.
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil {
		if !info.Mode().IsRegular() {
			return os.WriteFile(path, data, mode)
		}
		mode = info.Mode().Perm()
	}
	target, err := resolveLinks(path)
	if err != nil {
		return err
	}
	return replaceRegular(target, data, mode)
}

func resolveLinks(path string) (string, error) {
	for range 40 {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return path, nil
		}
		link, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(link) {
			link = filepath.Join(filepath.Dir(path), link)
		}
		path = link
	}
	return "", fmt.Errorf("too many symbolic links at %s", path)
}

func replaceRegular(path string, data []byte, mode fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	writeErr := tmp.Chmod(mode)
	if writeErr == nil {
		writeErr = writeAndSync(tmp, data)
	}
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func writeAndSync(file *os.File, data []byte) error {
	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return file.Sync()
}
