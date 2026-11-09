package command

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

type formalContentReaderS07 struct{ calls int }

func (r *formalContentReaderS07) Read(p []byte) (int, error) { r.calls++; return 0, io.EOF }

type formalContentErrorReaderS07 struct{}

func (formalContentErrorReaderS07) Read(p []byte) (int, error) { return 0, errors.New("read failure") }

func TestFormalContentS07Conflict(t *testing.T) {
	r := &formalContentReaderS07{}
	_, err := ReadContent(&Environment{In: r}, "inline", "-", "content")
	if err == nil || r.calls != 0 {
		t.Fatalf("conflict: error=%v reads=%d", err, r.calls)
	}
}

func TestFormalContentS07Chapter(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		wantNil bool
		want    string
	}{
		{nil, true, ""}, {[]string{"--content="}, false, ""}, {[]string{"--file="}, true, ""}, {[]string{"--file=-"}, false, ""},
	} {
		fs := flag.NewFlagSet("chapter", flag.ContinueOnError)
		title := fs.String("title", "", "")
		content := fs.String("content", "", "")
		summary := fs.String("summary", "", "")
		path := fs.String("file", "", "")
		if err := fs.Parse(tc.args); err != nil {
			t.Fatal(err)
		}
		p, err := ChapterInput(&Environment{In: strings.NewReader("")}, fs, title, content, summary, path)
		if err != nil || (p.Content == nil) != tc.wantNil {
			t.Fatalf("args=%v content=%v err=%v", tc.args, p.Content, err)
		}
		if p.Content != nil && *p.Content != tc.want {
			t.Fatalf("content=%q", *p.Content)
		}
		encoded, err := json.Marshal(p)
		if err != nil || strings.Contains(string(encoded), `"content":`) == tc.wantNil {
			t.Fatalf("args=%v JSON=%s err=%v", tc.args, encoded, err)
		}
	}
	_, err := ReadContent(&Environment{In: formalContentErrorReaderS07{}}, "", "-", "content")
	if err == nil || !strings.Contains(err.Error(), "read failure") {
		t.Fatalf("read error=%v", err)
	}
}
