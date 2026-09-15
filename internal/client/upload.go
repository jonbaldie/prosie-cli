package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func (c *Client) uploadRequest(ctx context.Context, path, filePath, field string, fields map[string]string) (*http.Request, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()
	body, contentType, err := encodeUpload(file, filePath, field, fields)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	return req, nil
}

func encodeUpload(file io.Reader, filePath, field string, fields map[string]string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", fmt.Errorf("failed to write %s field: %w", name, err)
		}
	}
	part, err := writer.CreateFormFile(field, filepath.Base(filePath))
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
