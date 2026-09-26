package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const fallbackVersion = "0.0.0-dev"

// Version is set by a release linker flag or by an exact Git tag for a local build.
var Version string

func init() {
	if Version != "" {
		return
	}

	Version = fallbackVersion
	for _, dir := range repositoryDirs() {
		if version, ok := exactTagVersion(dir); ok {
			Version = version
			return
		}
	}
}

func repositoryDirs() []string {
	dirs := make([]string, 0, 2)
	if executable, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(executable))
	}
	if dir, err := os.Getwd(); err == nil {
		dirs = append(dirs, dir)
	}

	unique := dirs[:0]
	for _, dir := range dirs {
		alreadyIncluded := false
		for _, included := range unique {
			if dir == included {
				alreadyIncluded = true
				break
			}
		}
		if !alreadyIncluded {
			unique = append(unique, dir)
		}
	}
	return unique
}

func exactTagVersion(dir string) (string, bool) {
	status, err := exec.Command("git", "-C", dir, "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil || len(status) != 0 {
		return "", false
	}

	tag, err := exec.Command("git", "-C", dir, "describe", "--exact-match", "--tags", "--match", "v[0-9]*", "HEAD").Output()
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(tag))
	if !strings.HasPrefix(value, "v") || len(value) == 1 {
		return "", false
	}
	return strings.TrimPrefix(value, "v"), true
}
