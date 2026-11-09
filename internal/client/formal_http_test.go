package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type formalHTTPTransport func(*http.Request) (*http.Response, error)

func (f formalHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type formalHTTPBody struct {
	io.Reader
	closed bool
}

func (b *formalHTTPBody) Close() error { b.closed = true; return nil }

func TestFormalHTTPResponseClosed(t *testing.T) {
	for _, status := range []int{200, 400} {
		body := &formalHTTPBody{Reader: strings.NewReader(`{"message":"bad"}`)}
		c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
		})})
		_, err := c.request(context.Background(), "GET", "/x", nil)
		if !body.closed || (status == 200 && err != nil) || (status == 400 && err == nil) {
			t.Fatalf("status=%d closed=%v err=%v", status, body.closed, err)
		}
	}
}

type formalHTTPPartial struct{ sent bool }

func (r *formalHTTPPartial) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.EOF
	}
	r.sent = true
	return copy(p, `[{"id":42}]`), errors.New("interrupted")
}

func TestFormalHTTPPartialChapterRead(t *testing.T) {
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
		var body io.Reader = strings.NewReader(`{"id":7}`)
		if strings.HasSuffix(r.URL.Path, "/scenes") {
			body = &formalHTTPPartial{}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(body), Header: make(http.Header)}, nil
	})})
	book, err := c.Books().GetBook(context.Background(), 7)
	if err != nil || len(book.Chapters) != 1 || book.Chapters[0].ID != 42 {
		t.Fatalf("book=%+v err=%v", book, err)
	}
}

func TestFormalHTTPSeriesBestEffort(t *testing.T) {
	var paths []string
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/series/7" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":7}`)), Header: make(http.Header)}, nil
		}
		return nil, errors.New("secondary unavailable")
	})})
	s, err := c.Series().GetSeries(context.Background(), 7)
	if err != nil || s == nil || s.ID != 7 || len(paths) != 3 || paths[1] != "/api/stories" || paths[2] != "/api/series/7/codex-entries" {
		t.Fatalf("series=%+v paths=%v err=%v", s, paths, err)
	}
}

func TestFormalHTTPListSeriesIndexOrder(t *testing.T) {
	var paths []string
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		body := `[{"id":7,"story_ids":[2,3,1]},{"id":8,"story_ids":[1]}]`
		if r.URL.Path == "/api/stories" {
			body = `[{"id":1},{"id":2}]`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	series, err := c.Series().ListSeries(context.Background())
	if err != nil || len(series) != 2 || len(series[0].Books) != 2 || series[0].Books[0].ID != 2 || series[0].Books[1].ID != 1 || len(series[1].Books) != 1 || series[1].Books[0].ID != 1 || len(paths) != 2 || paths[1] != "/api/stories" {
		t.Fatalf("series=%+v paths=%v err=%v", series, paths, err)
	}
}

func TestFormalHTTPStreamOwnership(t *testing.T) {
	for _, status := range []int{200, 400} {
		body := &formalHTTPBody{Reader: strings.NewReader(`event: done\ndata: {"prose":"ok"}\n\n`)}
		ctx, cancel := context.WithCancel(context.Background())
		transport := formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
			if r.Context() != ctx {
				t.Fatal("caller context changed")
			}
			return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
		})
		httpClient := &http.Client{Transport: transport, Timeout: time.Second}
		c := New("http://example.invalid", "", httpClient)
		resp, err := c.openSSEStream(ctx, "/stream", nil)
		if status == 400 {
			if err == nil || !body.closed {
				t.Fatalf("rejected: closed=%v err=%v", body.closed, err)
			}
		} else {
			if err != nil || body.closed {
				t.Fatalf("accepted: closed=%v err=%v", body.closed, err)
			}
			resp.Body.Close()
		}
		if httpClient.Timeout != time.Second {
			t.Fatal("original timeout changed")
		}
		cancel()
	}
}

func TestFormalHTTPStreamingCallerCloses(t *testing.T) {
	body := &formalHTTPBody{Reader: strings.NewReader("event: done\ndata: {\"prose\":\"ok\"}\n\n")}
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
	})})
	result, err := c.Generation().StreamContinue(context.Background(), "1", ContinueParams{}, nil)
	if err != nil || result == nil || result.Prose != "ok" || !body.closed {
		t.Fatalf("result=%+v closed=%v err=%v", result, body.closed, err)
	}
}

func TestFormalHTTPSeriesFallbackAndCodexPresent(t *testing.T) {
	var paths []string
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		body := `{"id":7,"codex_entries":[{"id":9}]}`
		if r.URL.Path == "/api/stories" {
			body = `[{"id":1,"series_id":7},{"id":2,"series_id":7},{"id":3,"series_id":8}]`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})})
	s, err := c.Series().GetSeries(context.Background(), 7)
	if err != nil || s == nil || len(s.Books) != 2 || s.Books[0].ID != 1 || s.Books[1].ID != 2 || len(s.CodexEntries) != 1 || len(paths) != 2 || paths[1] != "/api/stories" {
		t.Fatalf("series=%+v paths=%v err=%v", s, paths, err)
	}
}

func TestFormalHTTPHydrationFailure(t *testing.T) {
	var paths []string
	c := New("http://example.invalid", "", &http.Client{Transport: formalHTTPTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/stories/7" {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":7,"title":"primary"}`)), Header: make(http.Header)}, nil
		}
		return nil, errors.New("secondary unavailable")
	})})
	book, err := c.Books().GetBook(context.Background(), 7)
	if err != nil || book == nil || book.ID != 7 || book.Title != "primary" || len(book.Chapters) != 0 {
		t.Fatalf("book=%+v err=%v", book, err)
	}
	if len(paths) != 2 || paths[0] != "/api/stories/7" || paths[1] != "/api/stories/7/scenes" {
		t.Fatalf("paths=%v", paths)
	}
}
