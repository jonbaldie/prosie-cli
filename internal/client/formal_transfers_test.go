package client

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type formalTransferBadReader struct{}

func (formalTransferBadReader) Read([]byte) (int, error) { return 0, errors.New("input failed") }

func TestFormalTransfersUpload(t *testing.T) {
	c := New("http://example.invalid", "", nil)
	for _, tc := range []struct {
		name      string
		reader    io.Reader
		wantError bool
	}{
		{"good", strings.NewReader("hello"), false}, {"bad", formalTransferBadReader{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := (UploadFile{Filename: "book.docx", Reader: tc.reader}).postRequest(context.Background(), c, "/import", "file", map[string]string{"title": "example"})
			if tc.wantError {
				if err == nil || req != nil {
					t.Fatalf("failed input must not produce request: %v %v", req, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := req.ParseMultipartForm(4096); err != nil {
				t.Fatal(err)
			}
			f, header, err := req.FormFile("file")
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			data, err := io.ReadAll(f)
			if err != nil || string(data) != "hello" || header.Filename != "book.docx" || req.FormValue("title") != "example" {
				t.Fatalf("bad multipart: %q %q %v", data, header.Filename, err)
			}
		})
	}
}
