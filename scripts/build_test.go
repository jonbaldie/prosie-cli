package scripts_test

import (
	"archive/tar"
	"archive/zip"
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

func TestBuildReleaseScript(t *testing.T) {
	repoRoot := getRepoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "build-release.sh")

	distDir := t.TempDir()

	cmd := exec.Command(scriptPath, "0.1.0")
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
		archiveName := fmt.Sprintf("prosie_0.1.0_%s_%s%s", p.goos, p.goarch, p.ext)
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
	hostArchive := fmt.Sprintf("prosie_0.1.0_%s_%s%s", runtime.GOOS, runtime.GOARCH, hostExt)
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
		if !strings.Contains(string(verOut), "0.1.0") {
			t.Errorf("expected version 0.1.0, got %s", string(verOut))
		}
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
