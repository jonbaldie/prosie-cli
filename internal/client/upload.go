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

// withDefaultFilename returns the upload with fallback as its filename when it has none.
func (f UploadFile) withDefaultFilename(fallback string) UploadFile {
	if f.Filename == "" {
		f.Filename = fallback
	}
	return f
}

// postRequest builds a multipart POST to path that sends the upload in field beside fields.
func (f UploadFile) postRequest(ctx context.Context, transport *Client, path, field string, fields map[string]string) (*http.Request, error) {
	body, contentType, err := f.encode(field, fields)
	if err != nil {
		return nil, err
	}
	return transport.newRequest(ctx, http.MethodPost, path, body, contentType)
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
