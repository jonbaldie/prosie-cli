package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// trackedBody records whether the response body was closed.
type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

// recordingTransport answers every request with one canned response and records the request.
type recordingTransport struct {
	status int
	body   string
	err    error

	method string
	path   string
	query  string
	served *trackedBody
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.method = req.Method
	t.path = req.URL.EscapedPath()
	t.query = req.URL.RawQuery
	if t.err != nil {
		return nil, t.err
	}
	t.served = &trackedBody{Reader: strings.NewReader(t.body)}
	return &http.Response{
		StatusCode: t.status,
		Header:     http.Header{},
		Body:       t.served,
		Request:    req,
	}, nil
}

func recordingClient(transport *recordingTransport) *Client {
	return New("https://prosie.test", "token", &http.Client{Transport: transport})
}

type statusOnlyCase struct {
	name   string
	method string
	path   string
	call   func(context.Context, *Client) error
}

func statusOnlyCases() []statusOnlyCase {
	return []statusOnlyCase{
		{"DeleteBook", http.MethodDelete, "/api/stories/7", func(ctx context.Context, c *Client) error {
			return c.Books().DeleteBook(ctx, 7)
		}},
		{"DeleteChapter", http.MethodDelete, "/api/scenes/a%2Fb", func(ctx context.Context, c *Client) error {
			return c.Chapters().DeleteChapter(ctx, "a/b")
		}},
		{"DeleteConversation", http.MethodDelete, "/api/conversations/c1", func(ctx context.Context, c *Client) error {
			return c.Conversations().DeleteConversation(ctx, " c1 ")
		}},
		{"DeleteCodexEntry", http.MethodDelete, "/api/codex-entries/9", func(ctx context.Context, c *Client) error {
			return c.Codex().DeleteCodexEntry(ctx, 9)
		}},
		{"DeleteSeries", http.MethodDelete, "/api/series/3", func(ctx context.Context, c *Client) error {
			return c.Series().DeleteSeries(ctx, 3)
		}},
		{"DetachSeriesBook", http.MethodDelete, "/api/series/3/stories/7", func(ctx context.Context, c *Client) error {
			return c.Series().DetachSeriesBook(ctx, 3, 7)
		}},
		{"CancelContinue", http.MethodPost, "/api/scenes/42/continue/cancel", func(ctx context.Context, c *Client) error {
			return c.Generation().CancelContinue(ctx, "42")
		}},
	}
}

func TestStatusOnlyActionsSendRequestAndCloseBody(t *testing.T) {
	for _, tc := range statusOnlyCases() {
		t.Run(tc.name, func(t *testing.T) {
			transport := &recordingTransport{status: http.StatusNoContent}
			if err := tc.call(context.Background(), recordingClient(transport)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if transport.method != tc.method || transport.path != tc.path {
				t.Fatalf("expected %s %s, got %s %s", tc.method, tc.path, transport.method, transport.path)
			}
			if !transport.served.closed {
				t.Fatal("expected response body to be closed")
			}
		})
	}
}

func TestStatusOnlyActionsReturnApiErrorAndCloseBody(t *testing.T) {
	for _, tc := range statusOnlyCases() {
		t.Run(tc.name, func(t *testing.T) {
			transport := &recordingTransport{status: http.StatusUnprocessableEntity, body: `{"message":"Not allowed."}`}
			err := tc.call(context.Background(), recordingClient(transport))
			var apiErr *ApiError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *ApiError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != http.StatusUnprocessableEntity || apiErr.Message != "Not allowed." {
				t.Fatalf("unexpected API error: %+v", apiErr)
			}
			if !transport.served.closed {
				t.Fatal("expected response body to be closed")
			}
		})
	}
}

func TestStatusOnlyActionsWrapTransportFailure(t *testing.T) {
	for _, tc := range statusOnlyCases() {
		t.Run(tc.name, func(t *testing.T) {
			cause := errors.New("connection refused")
			transport := &recordingTransport{err: cause}
			err := tc.call(context.Background(), recordingClient(transport))
			if !errors.Is(err, cause) {
				t.Fatalf("expected wrapped transport error, got %v", err)
			}
			if !strings.HasPrefix(err.Error(), "request to "+tc.path+" failed: ") {
				t.Fatalf("expected request failure message for %s, got %q", tc.path, err.Error())
			}
		})
	}
}

type downloadCase struct {
	name  string
	path  string
	query string
	call  func(context.Context, *Client) ([]byte, error)
}

func downloadCases() []downloadCase {
	return []downloadCase{
		{"ExportStory", "/api/stories/7/export", "format=docx", func(ctx context.Context, c *Client) ([]byte, error) {
			return c.Books().ExportStory(ctx, 7, "docx")
		}},
		{"ExportChapter", "/api/scenes/42/export", "format=markdown", func(ctx context.Context, c *Client) ([]byte, error) {
			return c.Chapters().ExportChapter(ctx, "42", "")
		}},
		{"ExportConversation", "/api/conversations/c1/export", "", func(ctx context.Context, c *Client) ([]byte, error) {
			return c.Conversations().ExportConversation(ctx, "c1")
		}},
	}
}

func TestRawDownloadsReturnBodyAndCloseIt(t *testing.T) {
	for _, tc := range downloadCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw := "PK\x03\x04 raw \x00 bytes"
			transport := &recordingTransport{status: http.StatusOK, body: raw}
			got, err := tc.call(context.Background(), recordingClient(transport))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != raw {
				t.Fatalf("expected raw body %q, got %q", raw, got)
			}
			if transport.method != http.MethodGet || transport.path != tc.path || transport.query != tc.query {
				t.Fatalf("expected GET %s?%s, got %s %s?%s", tc.path, tc.query, transport.method, transport.path, transport.query)
			}
			if !transport.served.closed {
				t.Fatal("expected response body to be closed")
			}
		})
	}
}

func TestRawDownloadsReturnApiErrorAndCloseBody(t *testing.T) {
	for _, tc := range downloadCases() {
		t.Run(tc.name, func(t *testing.T) {
			transport := &recordingTransport{status: http.StatusNotFound, body: `{"message":"Not found."}`}
			got, err := tc.call(context.Background(), recordingClient(transport))
			var apiErr *ApiError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound || apiErr.Message != "Not found." {
				t.Fatalf("expected 404 API error, got %v", err)
			}
			if got != nil {
				t.Fatalf("expected no content on error, got %q", got)
			}
			if !transport.served.closed {
				t.Fatal("expected response body to be closed")
			}
		})
	}
}

func TestGetUserClosesBodyAndReturnsApiError(t *testing.T) {
	transport := &recordingTransport{status: http.StatusOK, body: `{"id":5,"name":"Ada","email":"ada@example.com"}`}
	user, err := recordingClient(transport).GetUser(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 5 || user.Name != "Ada" || user.Email != "ada@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if transport.method != http.MethodGet || transport.path != "/api/user" || !transport.served.closed {
		t.Fatalf("expected closed GET /api/user, got %s %s closed=%v", transport.method, transport.path, transport.served.closed)
	}

	transport = &recordingTransport{status: http.StatusOK, body: `not json`}
	if _, err := recordingClient(transport).GetUser(context.Background()); err == nil || !strings.HasPrefix(err.Error(), "failed to decode user response: ") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestExecuteSendsJSONBodyAndChecksStatus(t *testing.T) {
	var gotBody, gotType string
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(req.Body)
		gotBody = string(data)
		gotType = req.Header.Get("Content-Type")
		return &http.Response{StatusCode: http.StatusInternalServerError, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("boom")), Request: req}, nil
	})
	cli := New("https://prosie.test", "token", &http.Client{Transport: transport})

	err := cli.execute(context.Background(), http.MethodPost, "/api/thing", map[string]string{"a": "b"})
	var apiErr *ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError || apiErr.Message != "boom" {
		t.Fatalf("expected 500 API error, got %v", err)
	}
	if gotBody != `{"a":"b"}` || gotType != "application/json" {
		t.Fatalf("expected JSON body, got %q (%s)", gotBody, gotType)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
