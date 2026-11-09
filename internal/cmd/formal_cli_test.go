package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type formalCliWriter struct {
	calls int
	short bool
}

func (w *formalCliWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 1 {
		if w.short {
			return 0, nil
		}
		return 0, errors.New("first failure")
	}
	return len(p), nil
}

func TestFormalCliOutputLatch(t *testing.T) {
	for _, short := range []bool{false, true} {
		underlying := &formalCliWriter{short: short}
		w := &outputWriter{writer: underlying}
		_, first := w.Write([]byte("first"))
		_, second := w.Write([]byte("second"))
		if first == nil || second != first || underlying.calls != 1 || (short && first != io.ErrShortWrite) {
			t.Errorf("short=%v: errors %v / %v, calls %d", short, first, second, underlying.calls)
		}
	}
}

func TestFormalCliOutputReset(t *testing.T) {
	underlying := &formalCliWriter{short: true}
	var stderr bytes.Buffer
	root := &RootCmd{Out: underlying, Err: &stderr}
	if got := root.Execute([]string{"--help"}); got != 1 || !strings.Contains(stderr.String(), "error writing output") {
		t.Fatalf("first result=%d stderr=%q", got, stderr.String())
	}
	stderr.Reset()
	if got := root.Execute([]string{"--help"}); got != 0 || stderr.Len() != 0 || underlying.calls != 2 {
		t.Fatalf("second result=%d stderr=%q calls=%d", got, stderr.String(), underlying.calls)
	}
}
