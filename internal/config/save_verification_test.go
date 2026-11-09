package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

type saveVerificationFaultMask uint8

const (
	saveVerificationFailCreate saveVerificationFaultMask = 1 << iota
	saveVerificationFailWrite
	saveVerificationFailSync
	saveVerificationFailClose
	saveVerificationFailRename
)

type saveVerificationEvent struct {
	operation string
	requested int
	written   int
	err       error
}

type saveVerificationFile struct {
	tempFile
	t        *testing.T
	injected map[string]error
	events   *[]saveVerificationEvent
}

func (f *saveVerificationFile) Write(p []byte) (int, error) {
	f.t.Helper()
	assertPrivate(f.t, f.Name())
	data := p
	if f.injected["write"] != nil {
		data = p[:len(p)/2]
	}
	n, err := f.tempFile.Write(data)
	if injected := f.injected["write"]; injected != nil {
		err = errors.Join(err, injected)
	}
	*f.events = append(*f.events, saveVerificationEvent{
		operation: "write", requested: len(p), written: n, err: err,
	})
	return n, err
}

func (f *saveVerificationFile) Sync() error {
	err := f.injected["sync"]
	if err == nil {
		err = f.tempFile.Sync()
	}
	*f.events = append(*f.events, saveVerificationEvent{operation: "sync", err: err})
	return err
}

func (f *saveVerificationFile) Close() error {
	err := f.tempFile.Close()
	if injected := f.injected["close"]; injected != nil {
		err = errors.Join(err, injected)
	}
	*f.events = append(*f.events, saveVerificationEvent{operation: "close", err: err})
	return err
}

func TestSaveVerificationFailureMatrix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Log("Skipping mode assertions on Windows; file bytes, errors, order, and cleanup are still checked.")
	}
	steps := []struct {
		fault     saveVerificationFaultMask
		operation string
		err       error
	}{
		{saveVerificationFailCreate, "create", errors.New("injected create failure")},
		{saveVerificationFailWrite, "write", errors.New("injected write failure")},
		{saveVerificationFailSync, "sync", errors.New("injected sync failure")},
		{saveVerificationFailClose, "close", errors.New("injected close failure")},
		{saveVerificationFailRename, "rename", errors.New("injected rename failure")},
	}
	for _, initial := range []struct {
		name   string
		exists bool
	}{
		{"absent", false},
		{"existing", true},
	} {
		for mask := saveVerificationFaultMask(0); mask < saveVerificationFailRename<<1; mask++ {
			t.Run(fmt.Sprintf("%s/mask_%05b", initial.name, mask), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.json")
				old := []byte("{\"token\":\"old-token\"}\n")
				cfg := &Config{Token: "new-complete-credential-token"}
				if initial.exists {
					if err := os.WriteFile(path, old, 0644); err != nil {
						t.Fatal(err)
					}
					if runtime.GOOS != "windows" {
						if err := os.Chmod(path, 0644); err != nil {
							t.Fatal(err)
						}
					}
				}
				assertOriginal := func() {
					t.Helper()
					got, err := os.ReadFile(path)
					if !initial.exists {
						if !errors.Is(err, os.ErrNotExist) {
							t.Fatalf("destination appeared: bytes=%q, err=%v", got, err)
						}
						return
					}
					if err != nil || !bytes.Equal(got, old) {
						t.Fatalf("old destination changed: bytes=%q, err=%v", got, err)
					}
					if runtime.GOOS != "windows" {
						info, err := os.Stat(path)
						if err != nil {
							t.Fatal(err)
						}
						if info.Mode().Perm() != 0644 {
							t.Fatalf("old destination mode=%04o; want 0644", info.Mode().Perm())
						}
					}
				}
				assertReplacement := func(name string) {
					t.Helper()
					data, err := os.ReadFile(name)
					if err != nil {
						t.Fatal(err)
					}
					var document struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal(data, &document); err != nil {
						t.Fatalf("replacement is not complete JSON: %v", err)
					}
					if document.Token != cfg.Token {
						t.Fatalf("replacement token=%q; want %q", document.Token, cfg.Token)
					}
					assertPrivate(t, name)
				}

				injected := make(map[string]error)
				for _, step := range steps {
					if mask&step.fault != 0 {
						injected[step.operation] = step.err
					}
				}
				var events []saveVerificationEvent
				var stagingName string
				originalCreate, originalRename := createTemp, rename
				t.Cleanup(func() { createTemp, rename = originalCreate, originalRename })
				createTemp = func(parent, pattern string) (tempFile, error) {
					err := injected["create"]
					var file tempFile
					if err == nil {
						file, err = originalCreate(parent, pattern)
					}
					events = append(events, saveVerificationEvent{operation: "create", err: err})
					if err != nil {
						return nil, err
					}
					t.Cleanup(func() { _ = file.Close() })
					stagingName = file.Name()
					if parent != dir || filepath.Dir(stagingName) != dir || stagingName == path {
						t.Fatalf("staging file=%q, parent=%q; want a separate file in %q", stagingName, parent, dir)
					}
					assertPrivate(t, stagingName)
					data, err := os.ReadFile(stagingName)
					if err != nil || len(data) != 0 {
						t.Fatalf("new staging file is not empty: bytes=%q, err=%v", data, err)
					}
					assertOriginal()
					return &saveVerificationFile{
						tempFile: file, t: t, injected: injected, events: &events,
					}, nil
				}
				rename = func(from, to string) error {
					if from != stagingName || to != path {
						t.Fatalf("rename(%q, %q); want (%q, %q)", from, to, stagingName, path)
					}
					prefix := []string{"create", "write", "sync", "close"}
					if len(events) != len(prefix) {
						t.Fatalf("unsafe trace before rename: %+v", events)
					}
					for i, operation := range prefix {
						if events[i].operation != operation || events[i].err != nil {
							t.Fatalf("unsafe trace before rename: %+v", events)
						}
					}
					assertReplacement(from)
					assertOriginal()
					err := injected["rename"]
					if err == nil {
						err = originalRename(from, to)
					}
					events = append(events, saveVerificationEvent{operation: "rename", err: err})
					return err
				}

				err := Save(path, cfg)
				if (err != nil) != (mask != 0) {
					t.Errorf("Save error=%v; want failure=%v", err, mask != 0)
				}
				wantOperations := []string{"create"}
				if mask&saveVerificationFailCreate == 0 {
					wantOperations = append(wantOperations, "write")
					if mask&saveVerificationFailWrite == 0 {
						wantOperations = append(wantOperations, "sync")
					}
					wantOperations = append(wantOperations, "close")
					if mask&(saveVerificationFailWrite|saveVerificationFailSync|saveVerificationFailClose) == 0 {
						wantOperations = append(wantOperations, "rename")
					}
				}
				if len(events) != len(wantOperations) {
					t.Fatalf("trace=%+v; want operations=%v", events, wantOperations)
				}
				for i, event := range events {
					if event.operation != wantOperations[i] {
						t.Errorf("trace=%+v; want operations=%v", events, wantOperations)
					}
					if sentinel := injected[event.operation]; sentinel != nil {
						if !errors.Is(event.err, sentinel) {
							t.Errorf("%s error=%v; want injected sentinel", event.operation, event.err)
						}
					} else if event.err != nil {
						t.Errorf("unexpected %s error: %v", event.operation, event.err)
					}
					if event.operation == "write" {
						if injected["write"] != nil {
							if event.written < 0 || event.written >= event.requested || event.err == nil {
								t.Errorf("failed Write must return a short count and an error: %+v", event)
							}
						} else if event.written != event.requested {
							t.Errorf("successful Write was short: %+v", event)
						}
					}
				}
				for _, step := range steps {
					wantError := mask&step.fault != 0 && slices.Contains(wantOperations, step.operation)
					if errors.Is(err, step.err) != wantError {
						t.Errorf("Save error=%v; want %s sentinel=%v", err, step.operation, wantError)
					}
				}
				if mask == 0 {
					assertReplacement(path)
					assertLoadedToken(t, path, cfg.Token)
				} else {
					assertOriginal()
				}
				if initial.exists || mask == 0 {
					assertOnlyEntries(t, dir, "config.json")
				} else {
					assertOnlyEntries(t, dir)
				}
			})
		}
	}
}
