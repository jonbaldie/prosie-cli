package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// request reads a JSON endpoint response and owns the response body lifetime.
// Endpoint methods only construct their payload and decode their resource.
func (c *Client) request(ctx context.Context, method, path string, body any) ([]byte, error) {
	req, err := c.NewRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	return c.readResponse(req, path)
}

func (c *Client) readResponse(req *http.Request, path string) ([]byte, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return body, nil
}

// execute sends a request whose response carries only a status, and owns the
// response body lifetime. Status-only actions such as deletes and cancels use it.
func (c *Client) execute(ctx context.Context, method, path string, body any) error {
	req, err := c.NewRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer resp.Body.Close()
	return CheckResponse(resp)
}

// PostOAuth sends a JSON POST to an OAuth endpoint such as /oauth/device/code
// and returns the response body. A client with an empty token sends no
// Authorization header. A non-2xx response returns an *ApiError whose
// ErrorCode carries the OAuth "error" field.
func (c *Client) PostOAuth(ctx context.Context, path string, body any) ([]byte, error) {
	return c.request(ctx, http.MethodPost, path, body)
}
