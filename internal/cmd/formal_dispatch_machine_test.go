package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type formalDispatchFailOnce struct {
	calls int
	data  bytes.Buffer
}

func (w *formalDispatchFailOnce) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 1 {
		return 0, errors.New("failed output")
	}
	return w.data.Write(p)
}

func TestFormalDispatchRootExecutions(t *testing.T) {
	w := &formalDispatchFailOnce{}
	var stderr bytes.Buffer
	root := &RootCmd{Out: w, Err: &stderr}
	if code := root.Execute([]string{"--json", "version"}); code != 1 || !strings.Contains(stderr.String(), "error writing output") {
		t.Fatalf("first exit=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := root.Execute([]string{"version"}); code != 0 || stderr.Len() != 0 || json.Valid(w.data.Bytes()) {
		t.Fatalf("second exit=%d stdout=%q stderr=%q", code, w.data.String(), stderr.String())
	}
	w.data.Reset()
	if code := root.Execute([]string{"--json", "version", "--json=false"}); code != 0 || json.Valid(w.data.Bytes()) {
		t.Fatalf("override exit=%d stdout=%q", code, w.data.String())
	}
}
