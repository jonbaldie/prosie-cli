package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
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

// responseData accepts a bare JSON payload or extracts the data envelope.
func responseData(body []byte) ([]byte, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	if body[0] == '{' {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(body, &object); err != nil {
			return nil, fmt.Errorf("invalid response JSON: %w", err)
		}
		if data, ok := object["data"]; ok {
			return bytes.TrimSpace(data), nil
		}
		if len(object) == 0 {
			return nil, fmt.Errorf("empty response object")
		}
	}
	return body, nil
}

// resourceFields reports whether the object has a field defined by the resource.
// Unknown fields remain allowed, so new API fields do not break older clients.
func resourceFields(t reflect.Type, object map[string]json.RawMessage) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && resourceFields(field.Type, object) {
			return true
		}
		if fieldMatches(field, object) {
			return true
		}
	}
	return false
}

func fieldMatches(field reflect.StructField, object map[string]json.RawMessage) bool {
	if field.PkgPath != "" {
		return false
	}
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return false
	}
	if name == "" {
		name = field.Name
	}
	for key := range object {
		if strings.EqualFold(key, name) {
			return true
		}
	}
	return false
}

func decodeResource[T any](body []byte) (T, error) {
	var value T
	data, err := responseData(body)
	if err != nil {
		return value, err
	}
	var object map[string]json.RawMessage
	if len(data) == 0 || data[0] != '{' {
		return value, fmt.Errorf("expected a resource object")
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return value, fmt.Errorf("invalid resource: %w", err)
	}
	if !resourceFields(reflect.TypeOf(value), object) {
		return value, fmt.Errorf("resource has no recognized fields")
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("invalid resource: %w", err)
	}
	if normalizer, ok := any(&value).(interface{ normalize() }); ok {
		normalizer.normalize()
	}
	return value, nil
}

func decodeList[T any](body []byte) ([]T, error) {
	data, err := responseData(body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || data[0] != '[' {
		return nil, fmt.Errorf("expected a resource list")
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("invalid resource list: %w", err)
	}
	values := make([]T, 0, len(items))
	for i, item := range items {
		value, err := decodeResource[T](item)
		if err != nil {
			return nil, fmt.Errorf("invalid resource at index %d: %w", i, err)
		}
		values = append(values, value)
	}
	return values, nil
}
