package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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

func (c *Client) uploadRequest(ctx context.Context, path string, file UploadFile, field string, fields map[string]string) (*http.Request, error) {
	body, contentType, err := encodeUpload(file.Reader, filepath.Base(file.Filename), field, fields)
	if err != nil {
		return nil, err
	}
	return c.newRequest(ctx, http.MethodPost, path, body, contentType)
}

func encodeUpload(file io.Reader, filename, field string, fields map[string]string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("failed to write %s field: %w", name, err)
		}
	}
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create %s form field: %w", field, err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("failed to copy file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}
	return body, writer.FormDataContentType(), nil
}
