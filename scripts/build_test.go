package scripts_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var releasePlatforms = []struct {
	goos   string
	goarch string
	ext    string
}{
	{"darwin", "arm64", ".tar.gz"},
	{"darwin", "amd64", ".tar.gz"},
	{"linux", "amd64", ".tar.gz"},
	{"linux", "arm64", ".tar.gz"},
	{"windows", "amd64", ".zip"},
}

func getRepoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to locate git root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func TestCrossCompilation(t *testing.T) {
	repoRoot := getRepoRoot(t)

	for _, p := range releasePlatforms {
		t.Run(fmt.Sprintf("%s_%s", p.goos, p.goarch), func(t *testing.T) {
			tempDir := t.TempDir()
			binName := "prosie"
			if p.goos == "windows" {
				binName = "prosie.exe"
			}
			outPath := filepath.Join(tempDir, binName)

			buildCmd := exec.Command("go", "build", "-trimpath", "-o", outPath, repoRoot)
			buildCmd.Env = append(os.Environ(),
				"CGO_ENABLED=0",
				"GOOS="+p.goos,
				"GOARCH="+p.goarch,
			)

			output, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Fatalf("cross-compilation for %s/%s failed: %v\nOutput: %s", p.goos, p.goarch, err, string(output))
			}

			info, err := os.Stat(outPath)
			if err != nil {
				t.Fatalf("failed to stat generated binary: %v", err)
			}
			if info.Size() == 0 {
				t.Fatalf("expected non-empty binary, got size 0")
			}
		})
	}
}

func TestBuildReleaseScriptWithExplicitVersion(t *testing.T) {
	repoRoot := getRepoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "build-release.sh")

	distDir := t.TempDir()

	cmd := exec.Command(scriptPath, "v0.4.0")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"DIST_DIR="+distDir,
		"PATH="+os.Getenv("PATH"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build-release.sh execution failed: %v\nOutput: %s", err, string(output))
	}

	checksumFile := filepath.Join(distDir, "checksums.txt")
	checksumBytes, err := os.ReadFile(checksumFile)
	if err != nil {
		t.Fatalf("checksums.txt not generated: %v", err)
	}
	checksumContent := string(checksumBytes)

	for _, p := range releasePlatforms {
		archiveName := fmt.Sprintf("prosie_0.4.0_%s_%s%s", p.goos, p.goarch, p.ext)
		archivePath := filepath.Join(distDir, archiveName)

		archiveInfo, err := os.Stat(archivePath)
		if err != nil {
			t.Fatalf("expected archive %s does not exist: %v", archiveName, err)
		}
		if archiveInfo.Size() == 0 {
			t.Fatalf("archive %s is empty", archiveName)
		}

		// Verify archive sha256 is in checksums.txt
		hash := sha256OfFile(t, archivePath)
		if !strings.Contains(checksumContent, hash) {
			t.Errorf("checksums.txt does not contain sha256 %s for archive %s", hash, archiveName)
		}
		if !strings.Contains(checksumContent, archiveName) {
			t.Errorf("checksums.txt does not mention archive filename %s", archiveName)
		}

		// Verify files inside archive
		binaryName := "prosie"
		if p.goos == "windows" {
			binaryName = "prosie.exe"
		}

		if p.ext == ".tar.gz" {
			verifyTarGzContents(t, archivePath, []string{binaryName, "LICENSE"})
		} else if p.ext == ".zip" {
			verifyZipContents(t, archivePath, []string{binaryName, "LICENSE"})
		}
	}

	// Verify the host platform binary actually runs and returns the expected version
	hostExt := ".tar.gz"
	hostArchive := fmt.Sprintf("prosie_0.4.0_%s_%s%s", runtime.GOOS, runtime.GOARCH, hostExt)
	hostArchivePath := filepath.Join(distDir, hostArchive)
	if _, err := os.Stat(hostArchivePath); err == nil {
		unpackDir := t.TempDir()
		extractCmd := exec.Command("tar", "-xzf", hostArchivePath, "-C", unpackDir)
		if out, err := extractCmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to extract host archive: %v (%s)", err, string(out))
		}

		binPath := filepath.Join(unpackDir, "prosie")
		verCmd := exec.Command(binPath, "version")
		verOut, err := verCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("running extracted binary failed: %v (%s)", err, string(verOut))
		}
		if string(verOut) != "prosie version 0.4.0\n" {
			t.Errorf("expected version output %q, got %q", "prosie version 0.4.0\n", string(verOut))
		}
	}
}

func TestLocalBuildsUseExactTagByDefault(t *testing.T) {
	repoRoot := getRepoRoot(t)
	projectDir := newGitProject(t, repoRoot, "v0.4.0")

	binPath := filepath.Join(t.TempDir(), "prosie")
	buildCmd := exec.Command("go", "build", "-trimpath", "-o", binPath, ".")
	buildCmd.Dir = projectDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("plain tagged build failed: %v\nOutput: %s", err, string(output))
	}
	assertVersionOutput(t, binPath, projectDir, "prosie version 0.4.0\n")

	untaggedProject := newGitProject(t, repoRoot, "")
	assertVersionOutput(t, binPath, untaggedProject, "prosie version 0.0.0-dev\n")
	assertVersionOutputWithEnv(t, binPath, t.TempDir(), "prosie version 0.0.0-dev\n", "PATH=")

	distDir := t.TempDir()
	releaseCmd := exec.Command(filepath.Join(projectDir, "scripts", "build-release.sh"))
	releaseCmd.Dir = projectDir
	releaseCmd.Env = append(os.Environ(), "DIST_DIR="+distDir, "VERSION=")
	if output, err := releaseCmd.CombinedOutput(); err != nil {
		t.Fatalf("default local release build failed: %v\nOutput: %s", err, string(output))
	}

	for _, p := range releasePlatforms {
		archiveName := fmt.Sprintf("prosie_0.4.0_%s_%s%s", p.goos, p.goarch, p.ext)
		if _, err := os.Stat(filepath.Join(distDir, archiveName)); err != nil {
			t.Errorf("expected tagged archive %s: %v", archiveName, err)
		}
	}

	hostArchive := filepath.Join(distDir, fmt.Sprintf("prosie_0.4.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH))
	unpackDir := t.TempDir()
	if output, err := exec.Command("tar", "-xzf", hostArchive, "-C", unpackDir).CombinedOutput(); err != nil {
		t.Fatalf("failed to extract tagged host archive: %v (%s)", err, string(output))
	}
	assertVersionOutput(t, filepath.Join(unpackDir, "prosie"), projectDir, "prosie version 0.4.0\n")

	fallbackDistDir := t.TempDir()
	fallbackReleaseCmd := exec.Command(filepath.Join(untaggedProject, "scripts", "build-release.sh"))
	fallbackReleaseCmd.Dir = untaggedProject
	fallbackReleaseCmd.Env = append(os.Environ(), "DIST_DIR="+fallbackDistDir, "VERSION=")
	if output, err := fallbackReleaseCmd.CombinedOutput(); err != nil {
		t.Fatalf("default untagged release build failed: %v\nOutput: %s", err, string(output))
	}
	for _, p := range releasePlatforms {
		archiveName := fmt.Sprintf("prosie_0.0.0-dev_%s_%s%s", p.goos, p.goarch, p.ext)
		if _, err := os.Stat(filepath.Join(fallbackDistDir, archiveName)); err != nil {
			t.Errorf("expected fallback archive %s: %v", archiveName, err)
		}
	}
	fallbackHostArchive := filepath.Join(fallbackDistDir, fmt.Sprintf("prosie_0.0.0-dev_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH))
	fallbackUnpackDir := t.TempDir()
	if output, err := exec.Command("tar", "-xzf", fallbackHostArchive, "-C", fallbackUnpackDir).CombinedOutput(); err != nil {
		t.Fatalf("failed to extract fallback host archive: %v (%s)", err, string(output))
	}
	assertVersionOutput(t, filepath.Join(fallbackUnpackDir, "prosie"), untaggedProject, "prosie version 0.0.0-dev\n")
}

func newGitProject(t *testing.T, sourceRoot, tag string) string {
	t.Helper()

	projectDir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project fixture: %v", err)
	}
	fileList, err := exec.Command("git", "-C", sourceRoot, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("failed to list project files: %v", err)
	}
	for _, relativePath := range strings.Split(strings.TrimSuffix(string(fileList), "\x00"), "\x00") {
		if relativePath == "" {
			continue
		}
		sourcePath := filepath.Join(sourceRoot, filepath.FromSlash(relativePath))
		destinationPath := filepath.Join(projectDir, filepath.FromSlash(relativePath))
		info, err := os.Lstat(sourcePath)
		if err != nil {
			t.Fatalf("failed to inspect project file %s: %v", relativePath, err)
		}
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			t.Fatalf("failed to create fixture directory for %s: %v", relativePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(sourcePath)
			if err != nil {
				t.Fatalf("failed to read project link %s: %v", relativePath, err)
			}
			if err := os.Symlink(target, destinationPath); err != nil {
				t.Fatalf("failed to copy project link %s: %v", relativePath, err)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		contents, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("failed to read project file %s: %v", relativePath, err)
		}
		if err := os.WriteFile(destinationPath, contents, info.Mode().Perm()); err != nil {
			t.Fatalf("failed to copy project file %s: %v", relativePath, err)
		}
	}

	runGit(t, projectDir, "init", "--quiet")
	runGit(t, projectDir, "config", "user.name", "Prosie CLI test")
	runGit(t, projectDir, "config", "user.email", "prosie-cli-test@example.invalid")
	runGit(t, projectDir, "add", "-A")
	runGit(t, projectDir, "commit", "--quiet", "-m", "test fixture")
	if tag != "" {
		runGit(t, projectDir, "tag", tag)
	}
	return projectDir
}

func runGit(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	commandArgs := append([]string{"-C", repoDir}, args...)
	output, err := exec.Command("git", commandArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, string(output))
	}
}

func assertVersionOutput(t *testing.T, binaryPath, workingDir, expected string) {
	assertVersionOutputWithEnv(t, binaryPath, workingDir, expected)
}

func assertVersionOutputWithEnv(t *testing.T, binaryPath, workingDir, expected string, env ...string) {
	t.Helper()

	cmd := exec.Command(binaryPath, "--version")
	cmd.Dir = workingDir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Errorf("%s --version failed: %v\nstdout: %s\nstderr: %s", binaryPath, err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("%s --version wrote to stderr: %s", binaryPath, stderr.String())
	}
	if stdout.String() != expected {
		t.Errorf("%s --version output = %q, want %q", binaryPath, stdout.String(), expected)
	}
}

func sha256OfFile(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open file for hashing %s: %v", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		t.Fatalf("failed to compute sha256 for %s: %v", path, err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func verifyTarGzContents(t *testing.T, path string, requiredFiles []string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open archive %s: %v", path, err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader for %s: %v", path, err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	found := make(map[string]bool)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed reading tar entry in %s: %v", path, err)
		}
		found[filepath.Base(header.Name)] = true
	}

	for _, req := range requiredFiles {
		if !found[req] {
			t.Errorf("archive %s missing required entry %s", path, req)
		}
	}
}

func verifyZipContents(t *testing.T, path string, requiredFiles []string) {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("failed to open zip file %s: %v", path, err)
	}
	defer r.Close()

	found := make(map[string]bool)
	for _, f := range r.File {
		found[filepath.Base(f.Name)] = true
	}

	for _, req := range requiredFiles {
		if !found[req] {
			t.Errorf("zip archive %s missing required entry %s", path, req)
		}
	}
}
