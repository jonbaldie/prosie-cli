package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHTTPErrorDecodingPreservesConflictAndValidationDetails(t *testing.T) {
	cases := []struct {
		name, body, message, code, candidate string
		errors                               map[string][]string
	}{
		{"candidate", `{"message":"Nothing was saved.","error":"conflict","generated_content":"Candidate.","errors":[]}`, "Nothing was saved.", "conflict", "Candidate.", nil},
		{"validation", `{"message":"Invalid input.","errors":{"content":["Required.",17],"ignored":"not a list"}}`, "Invalid input.", "", "", map[string][]string{"content": {"Required."}}},
		{"description", `{"error_description":"Expired."}`, "Expired.", "", "", nil},
		{"code", `{"error":"conflict"}`, "", "conflict", "", nil},
		{"plain text", "  Gateway unavailable.\n", "Gateway unavailable.", "", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			result, err := New(server.URL, "fixture", server.Client()).Generation().Continue(context.Background(), "101", ContinueParams{Persist: true})
			apiErr, ok := err.(*ApiError)
			if result != nil || !ok {
				t.Fatalf("want no result and API error, got %+v and %v", result, err)
			}
			if apiErr.StatusCode != 409 || apiErr.Message != tc.message || apiErr.ErrorCode != tc.code || apiErr.GeneratedContent != tc.candidate || !reflect.DeepEqual(apiErr.Errors, tc.errors) {
				t.Fatalf("incorrect decoded error: %+v", apiErr)
			}
			if tc.candidate != "" && !strings.Contains(err.Error(), "Nothing was saved.\nGenerated text was not saved. Copy it before retrying.\nCandidate.") {
				t.Fatalf("candidate missing from error: %v", err)
			}
			if tc.name == "validation" && !strings.Contains(err.Error(), "content: Required.") {
				t.Fatalf("validation detail missing from error: %v", err)
			}
		})
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
}
