package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
)

// UploadFile is a named stream sent as a multipart file part.
// The caller owns Reader and closes it when it is a file.
type UploadFile struct {
	Filename string
	Reader   io.Reader
}

// named returns the upload with fallback as its filename when it has none.
func (f UploadFile) named(fallback string) UploadFile {
	if f.Filename == "" {
		f.Filename = fallback
	}
	return f
}

// encode writes the upload and fields as a multipart body and returns its content type.
func (f UploadFile) encode(field string, fields map[string]string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("failed to write %s field: %w", name, err)
		}
	}
	part, err := writer.CreateFormFile(field, filepath.Base(f.Filename))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create %s form field: %w", field, err)
	}
	if _, err := io.Copy(part, f.Reader); err != nil {
		return nil, "", fmt.Errorf("failed to copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}
	return body, writer.FormDataContentType(), nil
}
