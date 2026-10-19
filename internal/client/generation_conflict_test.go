package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAPIErrorDecodingPreservesConflictAndValidationDetails(t *testing.T) {
	cases := []struct {
		name, body, message, code, candidate, detail string
		errors                                       map[string][]string
	}{
		{"candidate", `{"message":"Nothing was saved.","error":"conflict","generated_content":"Candidate.","errors":[]}`, "Nothing was saved.", "conflict", "Candidate.", "Nothing was saved.\nGenerated text was not saved. Copy it before retrying.\nCandidate.", nil},
		{"validation", `{"message":"Invalid input.","errors":{"content":["Required.",17],"ignored":"not a list"}}`, "Invalid input.", "", "", "Invalid input. (content: Required.)", map[string][]string{"content": {"Required."}}},
		{"description", `{"error_description":"Expired."}`, "Expired.", "", "", "Expired.", nil},
		{"code", `{"error":"conflict"}`, "", "conflict", "", "conflict", nil},
		{"plain text", "  Gateway unavailable.  ", "Gateway unavailable.", "", "", "Gateway unavailable.", nil},
		{"malformed JSON", `{"message":`, `{"message":`, "", "", `{"message":`, nil},
		{"mixed fields", `{"message":17,"error_description":"Expired.","error":[],"generated_content":false,"errors":[] ,"status_code":409}`, "Expired.", "", "", "Expired.", nil},
	}
	type errorOperation struct {
		name   string
		status int
		run    func(*Client) (bool, error)
	}
	operations := []errorOperation{
		{"http", http.StatusConflict, func(cli *Client) (bool, error) {
			result, err := cli.Generation().Continue(context.Background(), "101", ContinueParams{Persist: true})
			return result == nil, err
		}},
	}
	for _, operation := range streamTestOperations() {
		operations = append(operations, errorOperation{operation.name, 0, func(cli *Client) (bool, error) {
			result, err := operation.run(cli, context.Background(), nil)
			return result == "", err
		}})
	}
	for _, operation := range operations {
		for _, tc := range cases {
			t.Run(operation.name+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if operation.status != 0 {
						w.WriteHeader(operation.status)
						_, _ = w.Write([]byte(tc.body))
						return
					}
					w.Header().Set("Content-Type", "text/event-stream")
					_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", tc.body)
				}))
				defer server.Close()
				noResult, err := operation.run(New(server.URL, "fixture", server.Client()))
				var apiErr *ApiError
				if !noResult || !errors.As(err, &apiErr) {
					t.Fatalf("want no result and typed API error, got noResult=%v and %v", noResult, err)
				}
				if apiErr.StatusCode != operation.status || apiErr.Message != tc.message || apiErr.ErrorCode != tc.code || apiErr.GeneratedContent != tc.candidate || !reflect.DeepEqual(apiErr.Errors, tc.errors) {
					t.Fatalf("incorrect decoded error: %+v", apiErr)
				}
				want := "stream error: " + tc.detail
				if operation.status != 0 {
					want = "API error (409): " + tc.detail
				}
				if err.Error() != want {
					t.Fatalf("error text %q, want %q", err.Error(), want)
				}
			})
		}
	}
}

func TestStreamConflictReturnsCandidateWithoutSuccessfulResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: delta\ndata: {\"delta\":\"Generated \"}\n\n"))
		_, _ = w.Write([]byte("event: error\ndata: {\"message\":\"Nothing was saved.\",\"error\":\"conflict\",\"generated_content\":\"Candidate.\"}\n\n"))
	}))
	defer server.Close()
	result, err := New(server.URL, "fixture", server.Client()).Generation().StreamContinue(context.Background(), "101", ContinueParams{Persist: true}, nil)
	if result != nil || err == nil || err.Error() != "stream error: Nothing was saved.\nGenerated text was not saved. Copy it before retrying.\nCandidate." {
		t.Fatalf("want retained candidate and no result, got %+v and %v", result, err)
	}
	var apiErr *ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want a typed API error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 0 || apiErr.Message != "Nothing was saved." || apiErr.ErrorCode != "conflict" || apiErr.GeneratedContent != "Candidate." {
		t.Fatalf("incorrect decoded stream error: %+v", apiErr)
	}
}
